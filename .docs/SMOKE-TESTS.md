# Manual Smoke Tests

Run through this list before tagging a release of the **org-scoped navigation + RBAC** slice (commits `0a09c2c`, `367c69d`, `31b40c8`, `5bcb78a`, plus the follow-up commit). Each item should take under a minute. Mark with ✅ when verified or 🐞 when filing a follow-up.

## Setup

```bash
make up
```

Then in the API container shell or via env (`docker-compose.override.yml`):

```bash
PLATFORM_ADMIN_EMAILS=admin@rekall.io,ops@rekall.io
PLATFORM_ADMIN_BOOTSTRAP_PASSWORD=changeme!     # only needed on first boot
```

Restart the API. The startup log should print
`admin reconciler done {created: N, promoted: N, demoted: N, listed_emails: 2}`.

Sign in as `admin@rekall.io` (the bootstrapped account) and as a regular user (register one through the UI).

---

## 1. Scope on the flat lists

- [ ] **Mixed scope rows render correct badges.** Open `/meetings` as a user who is a member of at least one org and has at least one open meeting + one org-scoped meeting. Each row shows either an "Open" chip or `{Org Name}` chip. Skeleton shimmer appears briefly while names load.
- [ ] **Badge click deep-links.** Click an org-scope badge on a row → URL changes to `/organizations/<id>/meetings` and the page renders pre-filtered.
- [ ] **Open badge is non-clickable.** Cursor is `default`; clicking does nothing.
- [ ] **Scope picker filter works.** From `/meetings`, click the "All scopes" chip, expand an org row, click "Personal (open)" → URL gains `?scope=open`, list filters down. Reload → filter persists.
- [ ] **Calls page mirrors the same behaviour** at `/calls`.

## 2. Org Switcher

- [ ] **TopBar shows "Personal" on `/dashboard`** (not on a scoped route).
- [ ] **TopBar shows the org name on `/organizations/:id`** and on the scoped meetings/calls subpages.
- [ ] **Personal entry navigates to `/dashboard`.**
- [ ] **Each org entry navigates to `/organizations/:id?tab=overview`.**
- [ ] **Zero-org platform admin** sees clickable "Create your first organization" → navigates to `/organizations`.
- [ ] **Zero-org non-admin** sees disabled "Contact your administrator to be added to an organization."

## 3. OrgDetailPage tabs

- [ ] **All four tabs render** (Overview / Departments / Meetings / Calls).
- [ ] **`?tab=` survives reload.** Switch to Calls → reload page → still on Calls.
- [ ] **Members tab list, invite, remove, danger-zone all behave as before** for owner/admin/member.
- [ ] **Departments tab cards link to `/organizations/:orgId/departments/:deptId`** (no in-place accordion).

## 4. DeptDetailPage

- [ ] **Three tabs render** (Overview / Meetings / Calls). `?tab=` persists.
- [ ] **"Back to organization" link** above the hero returns to `/organizations/:orgId?tab=departments`.
- [ ] **Overview tab member list** loads; "Add member" visible only to org admin/owner OR dept head OR platform admin.
- [ ] **Self-leave** — every user sees a remove icon on their own row, even if they can't manage members.
- [ ] **Promote-to-head** — the role select in the Add Member dialog shows the "Head" option only for org admins/owners or platform admins.

## 5. NewMeetingPage scope picker

- [ ] **Picker hidden for users with zero orgs.** Behaves like before — defaults to Open.
- [ ] **Picker pre-fills from `?scope=`.** Open `/meetings/new?scope=org:<uuid>` → picker shows that org.
- [ ] **"Private" toggle appears only when scope ≠ Personal.** Toggle on → meeting is created as private (verify in Meetings list).
- [ ] **Scoped Meetings page's "New Meeting" button** carries `?scope=` into the URL → picker pre-fills.

## 6. Platform admin RBAC

- [ ] **Non-admin POST /api/v1/organizations** → 403. Test with curl using a regular user's bearer token.
- [ ] **Admin POST /organizations with `owner_email: alice@x.com`** → org created with Alice as owner. Verify in `org_memberships` table.
- [ ] **Admin POST /organizations with bad `owner_email`** → 422 with "no user with that email".
- [ ] **Non-admin user does NOT see "New organization" button** on `/organizations`.
- [ ] **Platform admin can create a department in any org** (via API or UI), even when they are not a member.
- [ ] **Dept head cannot rename their dept** — UpdateDepartment via API returns 403.
- [ ] **Dept head cannot promote** — UpdateDeptMemberRole with `role=head` returns 403.

## 7. AccessDeniedState

- [ ] **Non-member visiting `/organizations/<other-org-id>`** → AccessDeniedState renders inside the Layout (sidebar + topbar still visible).
- [ ] **"Back to workspace"** returns to `/dashboard`.
- [ ] **Same behaviour for scoped meeting/call routes** under a non-member org.
- [ ] **Sign out → sign back in** (different user) — `OrgSwitcher` shows the new user's orgs (verifies `clearAuth → orgsStore.invalidate`).

## 8. Stale-permission 403 recovery

- [ ] **In one tab, demote a user from admin to member** (via API or another browser window).
- [ ] **In the demoted user's tab**, click "Invite member" → 403 toast appears: "You no longer have permission to perform this action. Refresh to see the latest state." Stores invalidate; refresh shows the new state.

## 9. Responsive

- [ ] **375 × 667 (mobile)** — `OrgSwitcher` collapses to icon-only; ScopeBreadcrumb truncates intermediate segments; ScopePicker still works (popover may need scroll on small screens).
- [ ] **Tablet 768 × 1024** — tabs scroll horizontally if needed; no overflow.

## 10. Background jobs

- [ ] **Restart API with one admin removed from `PLATFORM_ADMIN_EMAILS`.** Boot log shows `demoted: 1`. The removed user's `role` in the `users` table is now `member`.

---

When every box is ✅ the slice is ready to ship. Failed items go to GitHub issues with the smoke-step number in the title.
