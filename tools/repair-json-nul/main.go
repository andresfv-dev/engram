// repair-json-nul repairs the local representations that PostgreSQL JSONB
// rejects when a JSON string contains a Unicode NUL escape.
package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

const requiredProject = "saas-fundacion"

type config struct {
	apply   bool
	project string
	dbPath  string
}

type change struct {
	table   string
	id      string
	columns []string
	before  string
	after   string
	update  func(*sql.Conn) error
}

func main() {
	if err := run(os.Args[1:], os.Stdout); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string, out io.Writer) error {
	cfg, err := parseArgs(args)
	if err != nil {
		return err
	}

	db, err := sql.Open("sqlite", cfg.dbPath)
	if err != nil {
		return err
	}
	defer db.Close()

	changes, err := plan(context.Background(), db, cfg.project)
	if err != nil {
		return err
	}
	printPlan(out, changes, cfg.apply)
	if !cfg.apply || len(changes) == 0 {
		return nil
	}

	conn, err := db.Conn(context.Background())
	if err != nil {
		return err
	}
	defer conn.Close()
	if err := apply(conn, cfg.dbPath, changes); err != nil {
		return err
	}
	backupPath := lastBackupPath
	lastBackupPath = ""
	fmt.Fprintf(out, "backup=%s\nstatus=applied changed_rows=%d\n", backupPath, len(changes))
	return nil
}

// lastBackupPath is set only after a durable byte-for-byte backup exists.
// Keeping it package-local makes the transactional helper straightforward to test.
var lastBackupPath string

func parseArgs(args []string) (config, error) {
	fs := flag.NewFlagSet("repair-json-nul", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	cfg := config{}
	fs.BoolVar(&cfg.apply, "apply", false, "write repairs")
	fs.StringVar(&cfg.project, "project", "", "required project")
	fs.StringVar(&cfg.dbPath, "db", defaultDBPath(), "SQLite database path")
	if err := fs.Parse(args); err != nil {
		return config{}, err
	}
	if fs.NArg() != 0 {
		return config{}, errors.New("positional arguments are not supported")
	}
	if cfg.project != requiredProject {
		return config{}, fmt.Errorf("--project must be exactly %q", requiredProject)
	}
	if cfg.dbPath == "" {
		return config{}, errors.New("database path is required")
	}
	return cfg, nil
}

func defaultDBPath() string {
	if dataDir := os.Getenv("ENGRAM_DATA_DIR"); dataDir != "" {
		return filepath.Join(dataDir, "engram.db")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".engram", "engram.db")
}

func plan(ctx context.Context, db *sql.DB, project string) ([]change, error) {
	var changes []change
	for _, spec := range []struct {
		table   string
		id      string
		columns []string
	}{
		{"observations", "id", []string{"sync_id", "session_id", "type", "title", "content", "tool_name", "scope", "topic_key"}},
		{"user_prompts", "id", []string{"sync_id", "session_id", "content"}},
		{"sessions", "id", []string{"directory", "summary"}},
	} {
		found, err := planTextRows(ctx, db, spec.table, spec.id, spec.columns, project)
		if err != nil {
			return nil, err
		}
		changes = append(changes, found...)
	}
	mutationChanges, err := planMutations(ctx, db, project)
	if err != nil {
		return nil, err
	}
	return append(changes, mutationChanges...), nil
}

func planTextRows(ctx context.Context, db *sql.DB, table, idColumn string, columns []string, project string) ([]change, error) {
	selectColumns := append([]string{idColumn}, columns...)
	query := fmt.Sprintf("SELECT %s FROM %s WHERE project = ?", strings.Join(selectColumns, ", "), table)
	rows, err := db.QueryContext(ctx, query, project)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []change
	for rows.Next() {
		values := make([]sql.NullString, len(selectColumns))
		scans := make([]any, len(values))
		for i := range values {
			scans[i] = &values[i]
		}
		if err := rows.Scan(scans...); err != nil {
			return nil, err
		}
		updated := make([]string, len(columns))
		changedColumns := make([]string, 0, len(columns))
		for i, value := range values[1:] {
			updated[i] = value.String
			if strings.ContainsRune(value.String, 0) {
				updated[i] = strings.ReplaceAll(value.String, "\x00", "\uFFFD")
				changedColumns = append(changedColumns, columns[i])
			}
		}
		if len(changedColumns) == 0 {
			continue
		}
		rowID := values[0].String
		before := hashStrings(values[1:])
		after := hashValues(updated)
		setters := make([]string, len(changedColumns))
		for i, column := range changedColumns {
			setters[i] = column + " = ?"
		}
		result = append(result, change{
			table: table, id: rowID, columns: changedColumns, before: before, after: after,
			update: func(conn *sql.Conn) error {
				args := make([]any, 0, len(changedColumns)+2)
				for i, column := range columns {
					if strings.Contains(updated[i], "\uFFFD") && strings.Contains(values[i+1].String, "\x00") {
						_ = column
						args = append(args, updated[i])
					}
				}
				args = append(args, rowID, project)
				query := fmt.Sprintf("UPDATE %s SET %s WHERE %s = ? AND project = ?", table, strings.Join(setters, ", "), idColumn)
				result, err := conn.ExecContext(ctx, query, args...)
				if err != nil {
					return err
				}
				count, err := result.RowsAffected()
				if err != nil {
					return err
				}
				if count != 1 {
					return fmt.Errorf("expected one %s row %s, updated %d", table, rowID, count)
				}
				return nil
			},
		})
	}
	return result, rows.Err()
}

func planMutations(ctx context.Context, db *sql.DB, project string) ([]change, error) {
	rows, err := db.QueryContext(ctx, "SELECT seq, payload FROM sync_mutations WHERE project = ?", project)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []change
	for rows.Next() {
		var seq int64
		var payload string
		if err := rows.Scan(&seq, &payload); err != nil {
			return nil, err
		}
		updated, changed, err := normalizeJSON(payload)
		if err != nil {
			return nil, fmt.Errorf("sync_mutations seq=%d: %w", seq, err)
		}
		if !changed {
			continue
		}
		rowID := fmt.Sprintf("%d", seq)
		result = append(result, change{
			table: "sync_mutations", id: rowID, columns: []string{"payload"}, before: hash(payload), after: hash(updated),
			update: func(conn *sql.Conn) error {
				res, err := conn.ExecContext(ctx, "UPDATE sync_mutations SET payload = ? WHERE seq = ? AND project = ?", updated, seq, project)
				if err != nil {
					return err
				}
				count, err := res.RowsAffected()
				if err != nil {
					return err
				}
				if count != 1 {
					return fmt.Errorf("expected one sync_mutations row %d, updated %d", seq, count)
				}
				return nil
			},
		})
	}
	return result, rows.Err()
}

func normalizeJSON(payload string) (string, bool, error) {
	decoder := json.NewDecoder(strings.NewReader(payload))
	decoder.UseNumber()
	var value any
	if err := decoder.Decode(&value); err != nil {
		return "", false, err
	}
	if decoder.More() {
		return "", false, errors.New("multiple JSON values")
	}
	changed := normalizeValue(&value)
	if !changed {
		return payload, false, nil
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return "", false, err
	}
	if !json.Valid(encoded) {
		return "", false, errors.New("normalization produced invalid JSON")
	}
	return string(encoded), true, nil
}

func normalizeValue(value *any) bool {
	switch current := (*value).(type) {
	case string:
		if !strings.ContainsRune(current, 0) {
			return false
		}
		*value = strings.ReplaceAll(current, "\x00", "\uFFFD")
		return true
	case []any:
		changed := false
		for i := range current {
			changed = normalizeValue(&current[i]) || changed
		}
		return changed
	case map[string]any:
		changed := false
		for key, nested := range current {
			if normalizeValue(&nested) {
				current[key] = nested
				changed = true
			}
		}
		return changed
	default:
		return false
	}
}

func apply(conn *sql.Conn, dbPath string, changes []change) (err error) {
	if _, err := conn.ExecContext(context.Background(), "BEGIN IMMEDIATE"); err != nil {
		return err
	}
	committed := false
	defer func() {
		if !committed {
			_, _ = conn.ExecContext(context.Background(), "ROLLBACK")
		}
	}()
	backup, err := backupDatabase(dbPath)
	if err != nil {
		return err
	}
	lastBackupPath = backup
	for _, change := range changes {
		if err := change.update(conn); err != nil {
			return err
		}
	}
	if _, err := conn.ExecContext(context.Background(), "COMMIT"); err != nil {
		return err
	}
	committed = true
	return nil
}

func backupDatabase(dbPath string) (string, error) {
	info, err := os.Stat(dbPath)
	if err != nil {
		return "", err
	}
	backupDir := filepath.Join(filepath.Dir(dbPath), "backups")
	if err := os.MkdirAll(backupDir, 0o700); err != nil {
		return "", err
	}
	if err := os.Chmod(backupDir, 0o700); err != nil {
		return "", err
	}
	name := fmt.Sprintf("%s.%s.%s", filepath.Base(dbPath), requiredProject, time.Now().UTC().Format("20060102T150405.000000000Z"))
	backupPath := filepath.Join(backupDir, name)
	if err := os.Mkdir(backupPath, 0o700); err != nil {
		return "", err
	}
	if err := copyFile(dbPath, filepath.Join(backupPath, filepath.Base(dbPath)), info.Mode().Perm()); err != nil {
		return "", err
	}
	// A WAL database needs its sidecars with the main file to be restorable.
	// BEGIN IMMEDIATE above prevents concurrent writers while all bytes are copied.
	for _, suffix := range []string{"-wal", "-shm"} {
		sidecar := dbPath + suffix
		sidecarInfo, err := os.Stat(sidecar)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return "", err
		}
		if err := copyFile(sidecar, filepath.Join(backupPath, filepath.Base(sidecar)), sidecarInfo.Mode().Perm()); err != nil {
			return "", err
		}
	}
	return backupPath, nil
}

func copyFile(sourcePath, targetPath string, mode os.FileMode) error {
	source, err := os.Open(sourcePath)
	if err != nil {
		return err
	}
	defer source.Close()
	target, err := os.OpenFile(targetPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
	if err != nil {
		return err
	}
	if _, err := io.Copy(target, source); err != nil {
		target.Close()
		return err
	}
	if err := target.Sync(); err != nil {
		target.Close()
		return err
	}
	return target.Close()
}

func printPlan(out io.Writer, changes []change, apply bool) {
	status := "dry-run"
	if apply {
		status = "apply-planned"
	}
	fmt.Fprintf(out, "status=%s planned_rows=%d\n", status, len(changes))
	for _, change := range changes {
		fmt.Fprintf(out, "row type=%s id=%s columns=%s before_sha256=%s after_sha256=%s\n", change.table, change.id, strings.Join(change.columns, ","), change.before, change.after)
	}
}

func hash(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func hashStrings(values []sql.NullString) string {
	var buffer bytes.Buffer
	for _, value := range values {
		buffer.WriteString(value.String)
		buffer.WriteByte(0x1f)
	}
	return hash(buffer.String())
}

func hashValues(values []string) string { return hash(strings.Join(values, "\x1f") + "\x1f") }
