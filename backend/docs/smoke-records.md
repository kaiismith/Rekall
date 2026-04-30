# Smoke checklist — Records

Manual verification for the Records feature ([records-list spec](../../.kiro/specs/records-list/)).

NOT executed in CI. Run end-to-end before merging anything that touches the
records list, the detail page, or the paginated transcript read.

## Prerequisites

- Backend running with migrations applied (`make migrate-up`)
- Frontend running (`npm run dev`)
- ASR service reachable (any engine — `local` or `openai`)
- At least one user account with one ended meeting + persisted transcript

## Steps

### Sidebar + list

- [ ] Log in. Sidebar shows **Records** (NOT "Calls" or "Meetings"). Icon is the same phone-in-talk glyph as before.
- [ ] Click **Records**. URL becomes `/records`. Heading reads **"Your Records"**.
- [ ] Each card shows the same metadata as the existing `MeetingCard` (title, twin status badges, date `MMM D, YYYY`, duration, participant avatars, audio-bar SVG bottom-right).
- [ ] Filter / sort / scope controls function identically to the legacy `/meetings` page.
- [ ] Empty state (filter that returns nothing): copy reads "No records match this filter".
- [ ] Empty state (no meetings at all): copy reads "No records yet"; CTA reads "Start a meeting" and links to `/records/new`.

### Detail page

- [ ] Click a completed-meeting card. URL becomes `/records/<code>` (NOT `/meeting/<code>`).
- [ ] Header card matches the list card visually. Card is non-clickable on the detail page (no hover lift, no cursor pointer).
- [ ] Breadcrumb shows `Records / <title>` and the leading "Back to Records" button works.
- [ ] Transcript timeline renders below the header:
  - [ ] Each speaker block has a 32 px circular avatar, a deterministic colour from the speaker user id, the display name, and a `M:SS` (or `H:MM:SS` for >1h) timestamp.
  - [ ] Consecutive same-speaker segments collapse into one block (one speaker header per block, multiple paragraphs).
  - [ ] Low-confidence segments (`confidence < 0.4`) render in a muted text colour with a "Low confidence" tooltip on hover.
- [ ] Click **Load more** until the button disappears. Each click fetches the next page of 50; previously loaded segments stay in place (no jump-to-top).
- [ ] If a meeting is still in progress (`status = waiting | active`): info banner reads "This record is still in progress. Refresh to see new segments." and a "Join live" link goes to `/meeting/<code>`.

### RBAC

- [ ] Visit `/records/<code-of-someone-else's-meeting>`. Page shows:
  - Header card if you have list-level access (org/scope visibility).
  - Access-denied panel ("You don’t have access to this record’s transcript") in place of the timeline. NO crash. NO infinite spinner.
- [ ] Visit `/records/<bogus-code>`. Page renders "Record not found" with the back link.

### Legacy redirects

- [ ] `/meetings` → redirected to `/records` (URL bar updates).
- [ ] `/meetings/new` → redirected to `/records/new`.
- [ ] `/meeting/<live-code>` is **NOT** redirected — the live WebRTC room still loads. (Singular `/meeting/`, not plural `/meetings/`.)
- [ ] Leaving the live meeting room takes the user to `/records` (not `/meetings`).

### Filter/sort preservation across detail navigation

- [ ] Open `/records?status=complete&sort=duration_desc`. Click into a record's detail page.
- [ ] In the detail page, click the in-page **Back to Records** button → lands on plain `/records` (filters cleared — this is the in-page button's documented behaviour).
- [ ] Repeat the flow but use the **browser back button** instead → lands on `/records?status=complete&sort=duration_desc` with the filter UI restored. (React Router default behaviour; not custom logic.)

### Backend pagination contract sanity

Run against your local backend with a test user's bearer token:

```bash
curl -s -H "Authorization: Bearer $TOKEN" \
  "http://localhost:8080/api/v1/meetings/<code>/transcript?page=1&per_page=10" \
  | jq '.data.pagination'
```

- [ ] `pagination.page == 1`, `pagination.per_page == 10`.
- [ ] `pagination.total` matches the number of persisted segments for the meeting.
- [ ] `pagination.total_pages == ceil(total / 10)`.
- [ ] `pagination.has_more` is `true` for every page except the last.

```bash
# Clamping check — server caps per_page at 200 and ignores junk values.
curl -s -H "Authorization: Bearer $TOKEN" \
  "http://localhost:8080/api/v1/meetings/<code>/transcript?per_page=99999" \
  | jq '.data.pagination.per_page'   # → 200

curl -s -H "Authorization: Bearer $TOKEN" \
  "http://localhost:8080/api/v1/meetings/<code>/transcript?per_page=abc" \
  | jq '.data.pagination.per_page'   # → 50

curl -s -H "Authorization: Bearer $TOKEN" \
  "http://localhost:8080/api/v1/meetings/<code>/transcript?page=abc" \
  | jq '.data.pagination.page'       # → 1
```

### Speaker resolution sanity

```bash
curl -s -H "Authorization: Bearer $TOKEN" \
  "http://localhost:8080/api/v1/meetings/<code>" \
  | jq '.data.speakers'
```

- [ ] Returns one entry per distinct `transcript_sessions.speaker_user_id`, with `full_name` and `initials` populated.
- [ ] For a meeting that never enabled captions, returns `[]` (NOT `null`).

### Logs

- [ ] Trigger a 403 by hitting `/meetings/<other-user's-code>/transcript` with a non-host non-participant token. Backend log emits `RECORD_TRANSCRIPT_ACCESS_DENIED` with `meeting_id`, `caller_id`, and `reason: "not_host_not_participant"`.

## Out of scope (do NOT verify here — future specs)

- Live transcript push to the detail page while a record is in progress
- Transcript editing
- Transcript export (.txt / .srt / .vtt)
- Full-text search UI
- Audio playback of stored segments
- Auto-load on scroll (intersection observer)
- Scope-level transcript visibility for org/dept members who never joined