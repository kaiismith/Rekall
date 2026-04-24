# Roadmap

Rekall ships in phases. The foundation — organizations, meetings, and the live room — is in place. What comes next is the payoff: turning every call into durable, retrievable knowledge, and turning that knowledge into insight about how your organization actually works.

---

## Today ✅

The workspace is real and usable.

- **Accounts and workspaces.** Signup, email verification, organizations, departments, invitations.
- **Real-time meetings.** Video, audio, screen share, waiting-room knock flow, host controls, emoji reactions, raise hand.
- **In-room chat.** Side-channel messaging that persists through the life of the meeting.
- **Call history.** Every meeting becomes a record in the workspace.
- **Self-hosting.** One command to run the whole stack on your own infrastructure.

---

## Next 🚧

The next chapter turns the meeting from an event into a memory.

### ASR — automatic speech recognition
Every call, captured and transcribed word-for-word — running inside your own infrastructure, not outsourced to a vendor. Each record gets a readable, timestamped transcript attached.

**Why it matters:** the 70% of teams who take no notes today get perfect notes for free. The ones who *do* take notes get their time back. And because the transcript stays in your perimeter, regulated industries can finally say yes to meeting capture.

### AI Notes
A short, structured digest of what was covered, who said what, and what was decided — generated the moment the call ends and attached to the meeting record. Not a transcript dump: a readable recap your team will actually open.

**Why it matters:** people skim notes, they don't read hour-long transcripts. AI Notes are the handshake between the meeting and the rest of the organization.

### Action-item extraction
Pull out the commitments automatically — *"Sam will send the pricing doc by Friday"* — and surface them as owned, tracked tasks tied back to the moment they were agreed.

**Why it matters:** the single biggest reason meetings fail is that nobody leaves with a clear list of who owes what. Rekall closes that loop.

### Recording & archival
Opt-in recording with the audio/video stored in your own object storage. You choose what's kept, for how long, and who can see it.

**Why it matters:** compliance, onboarding, replay for missed attendees — without sending footage to a vendor.

### Calendar — schedule meetings inside Rekall
A native calendar for scheduling: pick a time, invite teammates, set the meeting scope (open or private), and everyone gets the join code ahead of time. Recurring meetings, reminders, and a per-user week/month view all live in the workspace.

**Why it matters:** meetings stop being ad-hoc links passed around in chat. Scheduling and the meeting itself live in the same product, under the same account, tied to the same record — so the calendar event, the live call, the transcript, and the follow-through are all one thing.

---

## Later 🔜

This is where Rekall stops being a video product and becomes an **organizational memory and insight system**.

### AI Ask — search your meetings like a teammate
Ask *"what did we decide about the Q3 launch?"* or *"when did Sam first raise the pricing concern?"* and get a real answer — a paragraph, citations to the exact moments it was said, and a link to replay those clips.

**Why it matters:** the single highest-leverage feature an organization can have is the ability to not re-discuss the same thing three times. AI Ask makes Rekall the first place you look, not the third.

### Topic classification
Every meeting gets tagged with the topics it actually covered — pricing, hiring, roadmap, customer X, incident Y — inferred from the conversation itself. Topics become first-class filters across the workspace.

**Why it matters:** leadership stops relying on titles in the calendar to understand what their team is spending time on. You can finally answer *"how many calls did we spend on the Acme deal this month?"* — exactly.

### Sentiment classification
Lightweight signals about how a conversation went — energy, agreement, tension — surfaced per meeting and trended over time. Never a replacement for judgment, always a flag worth looking at.

**Why it matters:** a gradual drop in energy on a project's standups, or rising tension in a recurring customer call, are things managers would absolutely want to catch early. Sentiment signals make the invisible visible.

### Observability dashboards
A real read on how your organization meets — at the whole-org level, per department, per recurring meeting.

- How much time is the team spending in calls?
- Which departments meet the most? The least?
- What's the average meeting length — and is it trending up?
- Which meetings consistently produce action items, and which don't?
- Which recurring meetings have the highest participation, attention, and follow-through?

**Why it matters:** meetings are the single largest line item in knowledge-work cost, and the least measured. Dashboards turn that black hole into something leadership can manage like any other budget.

### Meeting effectiveness scoring
A composite score for each meeting — combining attendance, participation balance, action-item generation, and follow-through — with drill-down into why a score is what it is. Rolled up to the department and organization level.

**Why it matters:** the teams that run great meetings can see what they're doing right. The teams that don't get concrete, actionable feedback — not a vibe, a number with receipts.

### Automated briefings to leadership
Meeting summaries, action items, and effectiveness signals delivered straight to the people who need them — by email (Outlook, Gmail) to the relevant department head or organization leader, the moment a meeting ends or on a daily/weekly digest cadence.

- **Department heads** get the rollup for the meetings inside their department.
- **Organization leaders** get a higher-level digest across the whole workspace.
- Opt-in per meeting, per department, or organization-wide — with clear rules about what gets forwarded and what stays in the room.

**Why it matters:** leadership gets the pulse of the organization without attending every meeting or chasing note-takers. Decisions, blockers, and action items reach the right inbox before they become problems. Rekall stops being a tool people have to remember to check and starts being a tool that tells them what they need to know.

### Integrations
- **External calendars** (Google, Microsoft) — two-way sync so Rekall's native calendar plays nicely with the one your team already lives in.
- **Slack / Teams** — summaries and action items get posted where the team already is.
- **Task trackers** (Linear, Jira, Asana) — action items push out as tickets, with a link back to the moment they were agreed.

**Why it matters:** Rekall becomes part of the team's existing workflow instead of a new place to remember to check.

---

## Guiding principles

Every feature on this roadmap is shaped by four commitments:

1. **Self-hosting stays a first-class citizen.** No feature will ever require sending your meeting data to a Rekall-run cloud. AI models, when they arrive, will run inside your perimeter.
2. **Privacy is not a tier.** Encryption, data ownership, and sensible defaults ship in the free, self-hosted product.
3. **We ship the memory layer before we ship the AI layer.** AI Ask, sentiment, and dashboards are only as good as the foundation of clean, well-organized meeting records — so we build that first.
4. **Boring beats clever.** Rekall would rather do three things reliably than ten things magically.

---

## Where things stand

Rekall is in **active early development**. The product is genuinely usable for the meeting and workspace layer today. The recall, AI, and insight layers — the parts that make the name "Rekall" earn itself — are under construction.

If that excites you: welcome. If you want a finished, polished product: check back in a few releases.
