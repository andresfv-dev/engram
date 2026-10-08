package dashboard

import (
	"context"
	"strings"
	"testing"
)

// Static assets are cached by browsers and CDNs, so every page must reference
// the stylesheet with a content-derived version that changes on each deploy.
func TestPagesReferenceVersionedStylesheet(t *testing.T) {
	if StylesVersion == "" {
		t.Fatal("StylesVersion is empty")
	}
	want := "/dashboard/static/styles.css?v=" + StylesVersion

	var layout strings.Builder
	if err := Layout("Users", "admin", "admin", true, EmptyState("x", "y")).Render(context.Background(), &layout); err != nil {
		t.Fatalf("render layout: %v", err)
	}
	if !strings.Contains(layout.String(), want) {
		t.Errorf("layout does not reference %q", want)
	}

	var login strings.Builder
	if err := LoginPage("", "").Render(context.Background(), &login); err != nil {
		t.Fatalf("render login: %v", err)
	}
	if !strings.Contains(login.String(), want) {
		t.Errorf("login page does not reference %q", want)
	}
}

// admin.js registers document-level listeners, so a page must load it exactly
// once even when it shows both the create-user dialog and a toast.
func TestAdminScriptLoadedOnceWithDialogAndToast(t *testing.T) {
	var out strings.Builder
	content := ManagedUsersPage(true, ManagedUserFilter{}, nil, 0)
	if err := LayoutWithNotice("Users", "admin", "admin", true, "User created", content).Render(context.Background(), &out); err != nil {
		t.Fatalf("render: %v", err)
	}
	if got := strings.Count(out.String(), "/dashboard/static/admin.js?v="); got != 1 {
		t.Errorf("expected admin.js exactly once, got %d", got)
	}
}
