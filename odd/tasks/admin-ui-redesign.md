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
- [x] T2 Users page: header, search, Role/Status/Project filters, collapsible create-user form, count, pagination.
- [x] T3 Users table rows: avatar, names, role badge, projects summary, status dot, created date.
- [x] T4 Row actions menu with the supported actions.
- [x] T5 Build image `local/engram-cloud:v2.1.0-admin-ui`, deploy, verify admin pages render and actions work; keep `v2.1.0` image for rollback.
- [x] T6 Mobile cards for the remaining dashboard tables (contributors, audit log, user detail tokens and grants, and any other `data-table` on admin or main pages): every cell labelled, rows rendered as cards below 720px.
- [ ] T7 Success toasts after admin actions (create user, enable/disable user, create/revoke token, grant/revoke project): transient, dismissible, accessible (`role="status"`), messages from a fixed allowlist of notice codes, never from raw user input.

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
- Commit: 939d9366
- `go test ./internal/cloud/dashboard/...`: RED (2 new tests failed: `TestNavTabsAdministrationEntryHighlighted`, `TestAdminSidebarRendersSectionsWithActiveHighlight`) then GREEN after implementation.
- `go vet ./internal/cloud/...`: clean.
- `go build ./cmd/engram`: clean.
- Global top bar: Administration entry now uses `navAdminTab` with `.nav-admin-highlight` class (accent color), separate from ordinary tabs.
- Admin sidebar: `adminNav` reordered to Overview, Users, Projects, System Health, Audit Log with an "ADMINISTRATION" header; CSS (`.frame-section:has(> .admin-nav)`) turns it into a sticky left sidebar with left-border active highlight, on all admin pages (Overview, Projects, Users, Health, user detail, Audit Log) without changing each page's templ structure.

### T2, T3, T4 — done (single combined commit)
- Commit: 5404bc81
- Delivered together in one commit: `ManagedUserRow` (HumanUser + granted project names) is the shared type threading T2's filters, T3's table columns and T4's row actions through `dashboard.go` and `components.templ`; splitting them into separate commits would have required re-deriving/duplicating that shared plumbing across intermediate, non-compiling states. Reported here rather than silently deviating.
- `go test ./internal/cloud/dashboard/...`: RED (new tests `TestManagedUsersPageHasSearchFiltersAndCollapsedForm`, `TestManagedUsersListFiltersBySearchRoleStatusAndProject` failed against old handler/templ signatures) then GREEN.
- `go test ./internal/cloud/...`: GREEN (all packages), including existing `cloudserver` tests asserting the `Create User` button/create-form markers still present.
- `go vet ./internal/cloud/...`: clean. `go build ./cmd/engram`: clean.
- T2: `ManagedUsersPage` now renders "Users" title + subtitle, a `q`/`role`/`status`/`project` GET filter form (also wired as an HTMX partial refresh), a `<details>`-collapsed "+ Create user" form (no longer always-open), and a server-computed "N users" count. `handleAdminUsersList`/`handleAdminUsers` apply `ManagedUserFilter` server-side (in-memory over `ListHumanUsers`, consistent with the existing "managed users are not paginated at the DB layer" design) and paginate at 20/page via `PaginationBar`, only shown when the filtered set exceeds one page.
- T3: `ManagedUsersListPartial` rows now show an initials avatar + username + muted display name, a distinct Admin/Member role badge, a "N projects: a, b, c +N" summary (needs the new `ManagedUsersStore.ListProjectGrants` method, added to the interface and satisfied by the existing `cloudstore.CloudStore.ListProjectGrants` and test stubs), a colored status dot, and the created date.
- T4: row actions use a `<details>`/`<summary>` "..." menu (no JS) with: View user (detail page), Manage projects / Manage tokens (detail page anchors `#grants`/`#tokens`), and Disable user (red) / Enable user depending on status, posting to the existing enable/disable route unchanged. "Edit user" is OMITTED — there is no existing endpoint to edit a managed user's display name/role (`internal/cloud/cloudserver/cloudserver.go` admin routes only cover create/enable/disable/tokens/grants), so it was never added rather than faked.

### Parent review fix — done
- Commit: 989266d. The row menu linked to `#tokens`/`#grants` but the detail page headings had no ids. Test extended first (RED: both anchors missing), ids added, templ regenerated (GREEN). `go test ./internal/cloud/...`, `go vet`, `go build` clean.

### Native review — unavailable
- `gentle-ai review assess`: risk medium, review due (slice budget reached). User granted consent; `review start` stopped with `lens_context_budget_exceeded` (3958 changed lines, mostly regenerated `components_templ.go`). No review authority created. User chose to deploy without native review.

### T5 — done
- Image `local/engram-cloud:v2.1.0-admin-ui` built and deployed; `v2.1.0` kept for rollback.
- Live checks on https://engram.andresfvelasquez.dev: login 303, `/dashboard/admin/users` 200 with sidebar, subtitle, search and "9 users"; list partial 200 with View user/Manage projects/Manage tokens on 9 rows, Disable user on 5 enabled and Enable user on 4 disabled users, project counts matching grants; `?status=disabled` shows 4 users; `?q=dan` shows 1 row; user detail 200 with `id="tokens"` and `id="grants"`; dashboard home 200.
- Edit user omitted: no backend endpoint edits a managed user's display name or role.

## Next step
- User checks the new admin UI in the browser. Rollback: set the compose image back to `local/engram-cloud:v2.1.0` and run `docker compose up -d cloud`.

### T6 — mobile cards for the remaining tables — done
- Added a shared `table-cards` class + generic CSS rule (`internal/cloud/dashboard/static/styles.css`, appended section) that, below 720px, hides `thead`, renders each `tr` as a bordered/padded card (same look as the existing `.admin-users-table` cards) and labels every `td[data-label]` via `::before { content: attr(data-label) }`. Action-only cells (a lone button/form) keep no `data-label` so they render without a confusing caption.
- Converted every list-shaped `table.data-table` (i.e. excluding the two key-value `<tr><th>/<td></tr>` tables in `PromptDetailPage`/`AdminHealthPage`, and the already-converted `.admin-users-table`) in `internal/cloud/dashboard/components.templ`: `DashboardActivityPartial`, `SessionsPartial`, `ContributorsListPartial`, `AdminProjectsPage`, `ManagedUserDetailPage` tokens table, `ManagedUserDetailPage` grants table, `AdminAuditLogListPartial`.
- Also converted the hand-built (non-templ) activity table string in `internal/cloud/dashboard/dashboard.go`'s `handleActivity` (the actual `/dashboard/activity` renderer — `DashboardActivityPartial` itself turned out to be dead code, unused by any handler; converted it anyway for consistency since T6 targets every `data-table`).
- `templ generate` regenerated `components_templ.go`.
- TDD: added `TestDashboardActivityTableIsCardsWithLabels`, `TestBrowserSessionsTableIsCardsWithLabels`, `TestContributorsListTableIsCardsWithLabels`, `TestAdminProjectsTableIsCardsWithLabels`, `TestAdminAuditLogTableIsCardsWithLabels` (`internal/cloud/dashboard/dashboard_test.go`) and `TestDashboardManagedUserDetailTokensAndGrantsTablesAreCardsWithLabels` (`internal/cloud/cloudserver/dashboard_admin_users_test.go`) — RED (missing `table-cards` class / `data-label` attributes), then GREEN after implementation.
- `go test -count=1 ./internal/cloud/...`: GREEN (all packages). `go vet ./internal/cloud/...`: clean. `go build -o /dev/null ./cmd/engram`: clean.
- Commit: (recorded below after commit).

### Follow-up: layout did not fit the screen — done
- Root cause 1: Cloudflare (and browsers) cached `styles.css` for 4 h (`cf-cache-status: HIT`), so the deploy served new markup with the old stylesheet. Fix 4ea75d4: stylesheet URL carries a content hash (`styles.css?v=<sha256[:12]>`); test `TestPagesReferenceVersionedStylesheet` (RED: undefined `StylesVersion`, then GREEN).
- Root cause 2: the sidebar grid placed the nav in the header's row, stretching it and leaving a large gap. Fix 6a8a17e: sidebar positioned out of flow, admin pages widened to `min(1600px, 96vw)`, filters on one row, Pico summary chevrons hidden, users table compacted.
- Visual check: headless Chrome previews at 1920 and 1366 px show all columns in view. Live: CSS link `?v=004f43fd1dc4`, served content identical to the repo file.

### Follow-up: full width and create-user modal — done
- Commit 1f07ae5: admin pages and status ribbon span the full viewport (1.5rem side margin); the inline `<details>` create-user form (it pushed the layout when opened) became a centered `<dialog>` opened by `static/admin.js` (content-versioned URL), closable with Escape, Cancel, the close button or a backdrop click.
- TDD: `TestManagedUsersPageHasSearchFiltersAndCreateUserDialog` (RED: dialog markers missing, then GREEN); `go test ./internal/cloud/...` and `go vet` clean.
- Visual check: headless Chrome at 2560 px, closed and open dialog, from live HTML; deployed.

### Follow-up: app-wide top bar, full width and responsive — done (scope extended by the user to the whole app)
- Commit 430f809: global navigation moved into the header (brand | nav right-aligned | user), compact tabs; all pages full width; breakpoints: <1280px nav on a second header row, <900px tables scroll horizontally, <720px global and admin navs scroll horizontally, <560px header stacks; project cards wider with stacked stat labels.
- TDD: `TestLayoutPlacesNavigationInsideHeader` (RED: nav rendered after the header, then GREEN); `go test ./internal/cloud/...`, `go vet` clean.
- Visual check: headless Chrome through an authenticated local proxy (live HTML + local CSS) for Dashboard, Browser, Projects, Contributors and Admin Users at 390, 820, 1280, 1440 and 2560 px.

### Follow-up: users as cards on mobile — done
- Commit: users-table cells carry `data-label`; below 720px each row renders as a card (user + actions, role + status, labelled projects and created date). Desktop table unchanged.
- TDD: `TestManagedUsersListCellsCarryColumnLabels` (RED: 0 labels, then GREEN); full `go test ./internal/cloud/...` and `go vet` clean.
- Visual check: headless Chrome via authenticated proxy at 390 px (cards) and 1440 px (table). Deployed.
