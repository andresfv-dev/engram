package dashboard

import (
	"context"
	"strings"
	"testing"
)

func renderLayout(t *testing.T, activeTab string, isAdmin bool) string {
	t.Helper()
	var out strings.Builder
	if err := Layout("Page", "alice", activeTab, isAdmin, EmptyState("x", "y")).Render(context.Background(), &out); err != nil {
		t.Fatalf("render layout: %v", err)
	}
	return out.String()
}

func sidebarHTML(t *testing.T, body string) string {
	t.Helper()
	start := strings.Index(body, `<aside class="shell-sidebar"`)
	end := strings.Index(body, `</aside>`)
	if start < 0 || end < start {
		t.Fatalf("sidebar <aside class=\"shell-sidebar\"> not found in layout")
	}
	return body[start:end]
}

// The top navigation bar is gone: no header bar, no horizontal tab list.
func TestLayoutHasNoTopNavigationBar(t *testing.T) {
	body := renderLayout(t, "dashboard", true)
	for _, banned := range []string{`class="shell-header"`, `class="shell-nav"`, `role="tablist"`, `role="tab"`} {
		if strings.Contains(body, banned) {
			t.Errorf("layout must not render the old top bar marker %q", banned)
		}
	}
}

// Every page renders one left sidebar holding brand, navigation and user block
// in that order, and the sidebar precedes the main content.
func TestLayoutSidebarLandmarkOrder(t *testing.T) {
	body := renderLayout(t, "dashboard", false)
	side := sidebarHTML(t, body)
	brand := strings.Index(side, `class="brand"`)
	nav := strings.Index(side, `class="sidebar-nav"`)
	user := strings.Index(side, `class="sidebar-user"`)
	if brand < 0 || nav < 0 || user < 0 || !(brand < nav && nav < user) {
		t.Fatalf("expected brand < nav < user in sidebar, got %d %d %d", brand, nav, user)
	}
	if strings.Index(body, `<aside class="shell-sidebar"`) > strings.Index(body, `<main class="shell-main"`) {
		t.Errorf("sidebar must precede main content")
	}
	if !strings.Contains(side, "alice") || !strings.Contains(side, `action="/dashboard/logout"`) {
		t.Errorf("user block must show the username and a logout form")
	}
}

func TestSidebarWorkspaceItemsAndActiveHighlight(t *testing.T) {
	side := sidebarHTML(t, renderLayout(t, "projects", false))
	if !strings.Contains(side, "WORKSPACE") {
		t.Errorf("expected Workspace group title")
	}
	for _, href := range []string{`/dashboard/"`, `/dashboard/browser"`, `/dashboard/projects"`, `/dashboard/contributors"`} {
		if !strings.Contains(side, `href="`+href) {
			t.Errorf("expected workspace link %q", href)
		}
	}
	if got := strings.Count(side, `aria-current="page"`); got != 1 {
		t.Fatalf("expected exactly one active item, got %d", got)
	}
	if !strings.Contains(side, `href="/dashboard/projects" class="side-link active" aria-current="page"`) {
		t.Errorf("expected Projects to be the active item, sidebar=%s", side)
	}
}

func TestSidebarHidesAdministrationForNonAdmins(t *testing.T) {
	body := renderLayout(t, "dashboard", false)
	for _, banned := range []string{"ADMINISTRATION", "Administration", `/dashboard/admin`} {
		if strings.Contains(body, banned) {
			t.Errorf("non-admin layout must not mention %q", banned)
		}
	}
}

func TestSidebarShowsAdministrationGroupForAdmins(t *testing.T) {
	side := sidebarHTML(t, renderLayout(t, "dashboard", true))
	if !strings.Contains(side, "ADMINISTRATION") {
		t.Errorf("expected Administration group title")
	}
	for _, href := range []string{"/dashboard/admin", "/dashboard/admin/users", "/dashboard/admin/projects", "/dashboard/admin/health", "/dashboard/admin/audit-log"} {
		if !strings.Contains(side, `href="`+href+`"`) {
			t.Errorf("expected admin link %q", href)
		}
	}
	if got := strings.Count(side, `aria-current="page"`); got != 1 {
		t.Errorf("expected exactly one active item, got %d", got)
	}
}

func TestSidebarActiveAdminItems(t *testing.T) {
	cases := map[string]string{
		"admin":           "/dashboard/admin",
		"admin-overview":  "/dashboard/admin",
		"admin-users":     "/dashboard/admin/users",
		"admin-projects":  "/dashboard/admin/projects",
		"admin-health":    "/dashboard/admin/health",
		"admin-audit-log": "/dashboard/admin/audit-log",
	}
	for tab, href := range cases {
		side := sidebarHTML(t, renderLayout(t, tab, true))
		want := `href="` + href + `" class="side-link active" aria-current="page"`
		if !strings.Contains(side, want) {
			t.Errorf("tab %q: expected %q active", tab, href)
		}
		if got := strings.Count(side, `aria-current="page"`); got != 1 {
			t.Errorf("tab %q: expected one active item, got %d", tab, got)
		}
	}
}

// Admin pages no longer render a second, page-level admin sidebar.
func TestAdminPagesRenderNoDuplicateAdminNav(t *testing.T) {
	var out strings.Builder
	if err := Layout("Users", "alice", "admin-users", true, ManagedUsersPage(true, ManagedUserFilter{}, nil, 0)).Render(context.Background(), &out); err != nil {
		t.Fatalf("render: %v", err)
	}
	if strings.Contains(out.String(), `class="admin-nav"`) {
		t.Errorf("admin pages must not render the old admin-nav sidebar")
	}
}

// ─── T2: collapsible groups and icons ───────────────────────────────────────

func detailsTag(t *testing.T, side, group string) string {
	t.Helper()
	marker := `<details class="nav-group nav-group-` + group + `"`
	start := strings.Index(side, marker)
	if start < 0 {
		t.Fatalf("details for group %q not found in %s", group, side)
	}
	end := strings.Index(side[start:], ">")
	return side[start : start+end+1]
}

func TestSidebarGroupsAreNativeDetails(t *testing.T) {
	side := sidebarHTML(t, renderLayout(t, "dashboard", true))
	for _, group := range []string{"workspace", "admin"} {
		tag := detailsTag(t, side, group)
		if !strings.Contains(tag, `data-nav-group="`+group+`"`) {
			t.Errorf("group %q must expose data-nav-group for state persistence: %s", group, tag)
		}
	}
	if got := strings.Count(side, `<summary class="nav-group-title"`); got != 2 {
		t.Errorf("expected a summary per group, got %d", got)
	}
}

func TestSidebarOpensOnlyTheGroupWithTheActivePage(t *testing.T) {
	cases := []struct {
		tab                      string
		workspaceOpen, adminOpen bool
	}{
		{"dashboard", true, false},
		{"contributors", true, false},
		{"admin-users", false, true},
		{"admin", false, true},
		{"unknown-tab", true, false},
	}
	for _, tc := range cases {
		side := sidebarHTML(t, renderLayout(t, tc.tab, true))
		ws := strings.Contains(detailsTag(t, side, "workspace"), " open")
		ad := strings.Contains(detailsTag(t, side, "admin"), " open")
		if ws != tc.workspaceOpen || ad != tc.adminOpen {
			t.Errorf("tab %q: workspace open=%v admin open=%v, want %v/%v", tc.tab, ws, ad, tc.workspaceOpen, tc.adminOpen)
		}
	}
}

func TestSidebarLinksCarryInlineIcons(t *testing.T) {
	side := sidebarHTML(t, renderLayout(t, "dashboard", true))
	links := strings.Count(side, `class="side-link`)
	icons := strings.Count(side, `<svg class="side-icon" aria-hidden="true"`)
	if links != 9 || icons != links {
		t.Errorf("expected 9 links each with an icon, got links=%d icons=%d", links, icons)
	}
	if !strings.Contains(side, `stroke="currentColor"`) {
		t.Errorf("icons must use currentColor")
	}
}

// Group state persists through a localStorage helper that must never throw.
func TestAdminScriptPersistsNavGroupsSafely(t *testing.T) {
	data, err := StaticFS.ReadFile("static/admin.js")
	if err != nil {
		t.Fatalf("read admin.js: %v", err)
	}
	js := string(data)
	for _, want := range []string{"data-nav-group", "localStorage", "try {", "catch"} {
		if !strings.Contains(js, want) {
			t.Errorf("admin.js must contain %q", want)
		}
	}
}
