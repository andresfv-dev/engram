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
