# Sidebar-only navigation (self-hosted Engram Cloud)

## Objective

Replace the top navigation bar of the Engram Cloud dashboard with a single left sidebar that holds all navigation in collapsible groups, following the user's reference mockup (`~/Descargas/Engram Cloud Cyberpunk Dashboard.png`).

## Problem

Navigation is split between a global top bar (Dashboard, Browser, Projects, Contributors, Administration, user) and a separate admin sidebar. The user finds the top bar unpleasant and wants one consistent left sidebar.

## Scope (authorized)

- Remove the top navigation bar on every dashboard page (keep a minimal brand area inside the sidebar).
- One left sidebar on every page, with collapsible groups:
  - **Workspace**: Dashboard, Browser, Projects, Contributors.
  - **Administration** (admins only): Overview, Users, Projects, System Health, Audit Log.
  - User block at the bottom: username and logout.
- The group containing the current page starts expanded; the active item is highlighted (left accent border + tinted background, as in the mockup).
- Collapsing uses native `<details>`/`<summary>` (works without JS); collapsed state may be remembered per browser with a small try/catch-guarded `localStorage` helper in `static/admin.js`.
- Line icons per item (inline SVG, `currentColor`), matching the mockup.
- Responsive: below 900px the sidebar becomes an off-canvas drawer opened by a menu button; no horizontal page scroll at 390px.
- The old admin sidebar (`adminNav`) merges into the Administration group; admin pages no longer render a second sidebar.

Out of scope for this feature (possible follow-up, only where backed by real data): the mockup's stat cards, charts, quick actions and system status widgets.

## Constraints

- Keep existing CSS color tokens and fonts; no new palette.
- UI only: no auth, route or data-model changes. Admin group visible only when `isAdmin`.
- Server-rendered templ + existing HTMX patterns; no JS framework.
- Regenerate `*_templ.go` with templ v0.3.1001 (pinned in `go.mod`).
- Keep the content-hashed stylesheet URL.
- Keep changes localized to dashboard templates, CSS and `admin.js` for easy rebases.

## Tasks

- [x] T1 Sidebar shell: remove top nav, render the left sidebar with Workspace/Administration groups, user block and active highlight on all pages; remove the duplicate admin sidebar.
- [x] T2 Collapsible groups (`<details>`), current group open by default, remembered state; icons.
- [x] T3 Responsive drawer below 900px with menu button; visual check at 390, 820, 1280, 1440, 2560px.

## Acceptance criteria

1. No page renders the top navigation bar; every page renders the left sidebar.
2. The active item is highlighted and its group is expanded on load.
3. Non-admin users never see the Administration group.
4. Groups collapse/expand without JS; with JS the state persists across reloads.
5. Existing dashboard tests pass (updated where they asserted the old top bar); new tests cover the sidebar structure, active item and admin gating.

## Checks

- TDD: strict (global configuration). Runner: `go test ./internal/cloud/...`.
- `go vet ./internal/cloud/...`, `go build -o /dev/null ./cmd/engram`.
- Visual: headless Chrome screenshots at the listed widths.

## Progress

- Branch: `feat/sidebar-nav` (from `feat/admin-ui-redesign`, e115170). Backup remote: `fork` (github.com/andresfv-dev/engram).
- Route: delegated direct (one writer; layout, components, CSS and JS are 2+ non-trivial files).
- Delivery strategy: ask-on-risk. Forecast ~600 authored lines (excluding generated `*_templ.go`).

### T1 - commit c69f893 `feat(dashboard): replace top bar with left sidebar navigation`
- RED: new `sidebar_test.go` (no top bar, landmark order, active item, admin gating, admin active items, no duplicate admin-nav) plus updated old tests failed against the top-bar markup.
- GREEN: `go test -count=1 ./internal/cloud/...` all packages ok.
- Updated old tests deliberately: removed `TestLayoutPlacesNavigationInsideHeader` (asserted the header/nav/user order of the removed top bar); `TestDashboardLayoutHTMLStructure` now expects `aside.shell-sidebar` and `nav.sidebar-nav` instead of `header.shell-header` / `nav.shell-nav`; `TestNavTabsAdministrationEntryHighlighted` became `TestSidebarAdministrationGroupMarked` (`nav-group-admin`); `TestAdminSidebarRendersSectionsWithActiveHighlight` now asserts `aria-current="page"` on Users.
- Design: layout `activeTab` now also carries the admin sub-item (`admin-overview|users|projects|health|audit-log`; legacy `admin` = overview). `adminNav`, `NavTabs`, `navTab`, `navAdminTab` removed; old top-bar and admin-sidebar CSS removed.
- Route: delegated direct (single writer). Tier/review: not run by the writer.

### T2 - commit 158f389 `feat(dashboard): collapsible sidebar groups with icons`
- RED: 4 new tests (details/summary, open-only-active-group, inline icons, admin.js guarded localStorage) failed.
- GREEN: full `./internal/cloud/...` ok.
- Groups are `<details data-nav-group>`; server opens the group holding the current page (unknown tab opens Workspace). admin.js remembers state under `engram.nav.*`, all access in try/catch; the current-page group is never forced closed.

### T3 - commit c19414a `feat(dashboard): off-canvas sidebar drawer below 900px`
- RED: 3 new tests (menu button/backdrop markup, admin.js drawer controller, stylesheet drawer rules) failed.
- GREEN: full `./internal/cloud/...` ok.
- Drawer enabled only when JS adds `drawer-enabled` (without JS the sidebar stays a stacked block, nothing hidden). Escape, backdrop and link click close it.
- Visual check (headless Google Chrome, static render of real layout + real styles.css/admin.js, page = admin Users): screenshots at 390, 820, 1280, 1440, 2560 plus open drawer at 390 and 820 and non-admin dashboard, in the session scratchpad. Probe in a 390px iframe: scrollWidth 390 = clientWidth 390 (no horizontal scroll), also with the drawer open. Found and fixed: Pico styles bare `<nav>` as `space-between` (gap between groups) and its summary chevron.

### Checks (final)
- `go test -count=1 ./internal/cloud/...`: ok (all packages).
- `go vet ./internal/cloud/...`: clean. `go build -o /dev/null ./cmd/engram`: ok.
- Not covered: interactive drawer click/Escape in a real browser beyond one scripted open (the toggle click rendered the open drawer); no JS unit-test runner exists in the repo.
- Stat cards, charts, quick actions and status widgets from the mockup remain out of scope.
