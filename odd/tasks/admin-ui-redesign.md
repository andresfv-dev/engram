# Admin UI redesign (self-hosted Engram Cloud)

## Objective

Redesign the Engram Cloud dashboard **Admin** section of the self-hosted deployment so it is easier to use, keeping the existing Engram color palette and visual identity.

## Problem

The current admin surface uses horizontal sub-tabs, a large always-open "create user" form and a plain table. Operators cannot search or filter users, cannot see each user's projects at a glance, and per-user actions (grants, tokens, enable/disable) are only reachable from the detail page.

## Scope (authorized)

Admin pages only (`/dashboard/admin/*`). Other dashboard pages keep their current layout, except the global navigation entry that links to Admin.

- Global top bar: brand, Dashboard, Browser, Projects, Contributors, highlighted **Administration** item, user menu with logout.
- Left admin sidebar: Overview, Users, Projects, System Health, Audit Log (active item highlighted). Replaces the horizontal admin tabs.
- Users page: title + subtitle, search (username, email, display name), filters (Role, Status, Project), "Create user" button that reveals the create form (no always-open form), user count.
- Users table: initials avatar, username + display name, role badge (Admin highlighted), project count + first project names (`+N` overflow), status dot (Active / Disabled), created date, actions menu.
- Row actions menu (`...`): View user, Edit user, Manage projects, Manage tokens, Disable user / Enable user. Only actions backed by existing server capabilities; an action without backend support is omitted and reported, never faked.
- Pagination when the list exceeds the page size.

## Constraints

- Keep the existing CSS color tokens and fonts; no new color palette.
- Keep deny-by-default and grant semantics unchanged; UI only, no auth or data-model changes.
- Server-rendered templ + existing HTMX patterns; no new JS framework. Minimal vanilla JS only if a menu toggle requires it.
- Regenerate `*_templ.go` with the templ version pinned in `go.mod` (v0.3.1001).
- Must stay easy to rebase on future upstream releases: keep changes localized to the admin templates, handlers and CSS.

## Tasks

- [x] T1 Admin shell: global top bar with Administration entry + left admin sidebar for all admin pages.
- [ ] T2 Users page: header, search, Role/Status/Project filters, collapsible create-user form, count, pagination.
- [ ] T3 Users table rows: avatar, names, role badge, projects summary, status dot, created date.
- [ ] T4 Row actions menu with the supported actions.
- [ ] T5 Build image `local/engram-cloud:v2.1.0-admin-ui`, deploy, verify admin pages render and actions work; keep `v2.1.0` image for rollback.

## Acceptance criteria

1. Every admin page shows the global top bar and the admin sidebar with the current section highlighted.
2. Searching or filtering the users list narrows the rows accordingly; clearing filters shows all users.
3. Each user row shows role, project count/names, status and created date from real data.
4. The actions menu of a row performs the real action (navigate to detail sections, disable/enable) and shows no unsupported action.
5. Existing dashboard tests pass; new tests cover the filtering and the actions menu rendering.

## Checks

- TDD: strict (global configuration). Runner: `go test ./internal/cloud/...`.
- `go vet ./internal/cloud/...`
- Manual: load `/dashboard/admin/users` on the deployed server.

## Progress

- Branch: `feat/admin-ui-redesign` (from `deploy/v2.1.0`).
- Route: delegated direct (one writer; 4+ files to map and 2+ non-trivial files to change).

### T1 — done
- Commit: (recorded after commit below)
- `go test ./internal/cloud/dashboard/...`: RED (2 new tests failed: `TestNavTabsAdministrationEntryHighlighted`, `TestAdminSidebarRendersSectionsWithActiveHighlight`) then GREEN after implementation.
- `go vet ./internal/cloud/...`: clean.
- `go build ./cmd/engram`: clean.
- Global top bar: Administration entry now uses `navAdminTab` with `.nav-admin-highlight` class (accent color), separate from ordinary tabs.
- Admin sidebar: `adminNav` reordered to Overview, Users, Projects, System Health, Audit Log with an "ADMINISTRATION" header; CSS (`.frame-section:has(> .admin-nav)`) turns it into a sticky left sidebar with left-border active highlight, on all admin pages (Overview, Projects, Users, Health, user detail, Audit Log) without changing each page's templ structure.
