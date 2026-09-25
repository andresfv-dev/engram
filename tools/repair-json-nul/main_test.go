package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

func TestDryRunIsScopedAndDoesNotWrite(t *testing.T) {
	dbPath := testDatabase(t)
	before := readValue(t, dbPath, "SELECT content FROM observations WHERE id = 1")
	var output bytes.Buffer
	if err := run([]string{"--project", requiredProject, "--db", dbPath}, &output); err != nil {
		t.Fatal(err)
	}
	if got := readValue(t, dbPath, "SELECT content FROM observations WHERE id = 1"); got != before {
		t.Fatal("dry-run changed authoritative data")
	}
	if got := readValue(t, dbPath, "SELECT payload FROM sync_mutations WHERE seq = 1356"); !strings.Contains(got, `\u0000`) {
		t.Fatal("dry-run changed mutation")
	}
	if got := readValue(t, dbPath, "SELECT content FROM observations WHERE id = 2"); !strings.ContainsRune(got, 0) {
		t.Fatal("dry-run touched another project")
	}
	if !strings.Contains(output.String(), "planned_rows=2") || !strings.Contains(output.String(), "before_sha256=") {
		t.Fatalf("unexpected metadata output: %q", output.String())
	}
}

func TestApplyBacksUpReplacesAndIsIdempotent(t *testing.T) {
	dbPath := testDatabase(t)
	original, err := os.ReadFile(dbPath)
	if err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	if err := run([]string{"--apply", "--project", requiredProject, "--db", dbPath}, &output); err != nil {
		t.Fatal(err)
	}
	backup := outputLine(output.String(), "backup=")
	if backup == "" {
		t.Fatalf("missing backup output: %q", output.String())
	}
	backupInfo, err := os.Stat(filepath.Join(backup, "engram.db"))
	if err != nil {
		t.Fatal(err)
	}
	if backupInfo.Mode().Perm() != 0o640 {
		t.Fatalf("backup permissions = %o", backupInfo.Mode().Perm())
	}
	backupBytes, err := os.ReadFile(filepath.Join(backup, "engram.db"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(original, backupBytes) {
		t.Fatal("backup does not match the pre-apply database bytes")
	}
	if got := readValue(t, dbPath, "SELECT content FROM observations WHERE id = 1"); strings.ContainsRune(got, 0) || !strings.ContainsRune(got, '\uFFFD') {
		t.Fatal("authoritative row was not normalized")
	}
	payload := readValue(t, dbPath, "SELECT payload FROM sync_mutations WHERE seq = 1356")
	if strings.Contains(payload, `\u0000`) || !json.Valid([]byte(payload)) {
		t.Fatal("mutation was not normalized to valid JSON")
	}
	if got := readValue(t, dbPath, "SELECT content FROM observations WHERE id = 2"); !strings.ContainsRune(got, 0) {
		t.Fatal("apply touched another project")
	}
	output.Reset()
	if err := run([]string{"--apply", "--project", requiredProject, "--db", dbPath}, &output); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "planned_rows=0") || strings.Contains(output.String(), "backup=") {
		t.Fatalf("repair is not idempotent: %q", output.String())
	}
}

func TestRequiresExactProject(t *testing.T) {
	if err := run([]string{"--apply", "--project", "other", "--db", "unused"}, &bytes.Buffer{}); err == nil {
		t.Fatal("expected project guard")
	}
}

func testDatabase(t *testing.T) string {
	t.Helper()
	dbPath := filepath.Join(t.TempDir(), "engram.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	for _, statement := range []string{
		"CREATE TABLE observations (id INTEGER PRIMARY KEY, sync_id TEXT, session_id TEXT, type TEXT, title TEXT, content TEXT, tool_name TEXT, project TEXT, scope TEXT, topic_key TEXT)",
		"CREATE TABLE user_prompts (id INTEGER PRIMARY KEY, sync_id TEXT, session_id TEXT, content TEXT, project TEXT)",
		"CREATE TABLE sessions (id TEXT PRIMARY KEY, project TEXT, directory TEXT, summary TEXT)",
		"CREATE TABLE sync_mutations (seq INTEGER PRIMARY KEY, payload TEXT, project TEXT)",
	} {
		if _, err := db.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := db.Exec("INSERT INTO observations VALUES (1, 'obs-1', 's-1', 'note', 'title', ?, '', ?, 'project', '')", "left\x00right", requiredProject); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec("INSERT INTO observations VALUES (2, 'obs-2', 's-2', 'note', 'title', ?, '', 'other', 'project', '')", "other\x00project"); err != nil {
		t.Fatal(err)
	}
	payload := `{"sync_id":"obs-1","content":"left\u0000right","project":"saas-fundacion"}`
	if _, err := db.Exec("INSERT INTO sync_mutations VALUES (1356, ?, ?)", payload, requiredProject); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(dbPath, 0o640); err != nil {
		t.Fatal(err)
	}
	return dbPath
}

func readValue(t *testing.T, dbPath, query string) string {
	t.Helper()
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	var value string
	if err := db.QueryRow(query).Scan(&value); err != nil {
		t.Fatal(err)
	}
	return value
}

func outputLine(output, prefix string) string {
	for _, line := range strings.Split(output, "\n") {
		if strings.HasPrefix(line, prefix) {
			return strings.TrimPrefix(line, prefix)
		}
	}
	return ""
}
