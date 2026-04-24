# Features

A tour of what you can do with Rekall today, grouped by the job it does for your team.

---

## 🏢 Run your organization

Rekall gives every team a **workspace** — a single place where people, meetings, and memory live together.

- **Organizations.** Sign up, name your company, and you own the workspace. One organization, one source of truth.
- **Departments.** Group teammates into meaningful units — Engineering, Sales, Design. Meetings can be scoped to a department so the right people show up and the right people see the history.
- **Invitations by email.** Bring in a teammate by email; they click a link and they're in. No manual account provisioning.
- **Roles.** Members and admins, with the permissions you'd expect — admins manage membership and settings, members join and host meetings.

**Business value:** a lightweight team directory that doubles as the access layer for everything else in Rekall. No separate SSO setup for the MVP, no paid seat tiers.

---

## 🎥 Meet face-to-face

Rekall is a real video product — not a link that opens somewhere else.

- **Shareable join codes.** Every meeting gets a human-readable code (`abc-defg-hij`) you can drop in a chat or calendar invite.
- **Open and private meetings.** Open meetings admit anyone with the code; private meetings use a **waiting-room knock** flow where anyone already inside can let a guest in.
- **Peer-to-peer media.** Video and audio flow directly between participants. The Rekall server never sees the video stream — good for performance, good for privacy, cheap to operate.
- **Works in the browser.** No client to install, no plugin, no account friction for guests.

**Business value:** a video layer you own, that feels familiar to anyone who has used Google Meet or Zoom, but runs entirely inside your perimeter.

---

## 🙋 A rich in-room experience

Meetings are more than faces on a grid. Rekall gives the room the controls a working team actually uses:

- **Mic, camera, screen share.** The basics, polished.
- **Pre-meeting device check.** Before you join, confirm the right camera and mic are selected — no more "can you hear me?"
- **Virtual backgrounds.** Blur or swap in a clean background with one click.
- **Speaking indicators.** On-device voice detection highlights whoever is talking.
- **Raise hand and emoji reactions.** Low-friction ways to participate in a presentation without cutting someone off.
- **In-room chat.** A side channel for links, clarifying questions, and anyone who doesn't want to interrupt.
- **Host controls.** Mute disruptive participants, admit knockers from the waiting room, end the meeting for everyone.

**Business value:** the meeting feels professional and considered, not like a minimum-viable video chat. Teams stop context-switching to other tools for things the room should already handle.

---

## 📞 Keep a call history

Every call becomes a **record** in the workspace — not a file on someone's laptop, not a link in someone's DMs.

- **Browsable.** See every meeting your team has run, who was there, and when.
- **Scoped.** You see the meetings that belong to the parts of the organization you're in.
- **Ready for recall.** Each record is the anchor for transcripts, summaries, and search as they come online *(see the roadmap)*.

**Business value:** the end of "did we talk about this already?" A team's conversation history becomes an asset, not something that disappears at 5pm.

---

## 🔒 Private by design

Rekall is built for organizations that want to know exactly where their data lives.

- **Self-hostable.** One command brings up the whole stack. Runs on your infrastructure, under your policies.
- **No third-party processors for media.** Video and audio don't traverse a vendor cloud — they go peer-to-peer.
- **Your data, your database.** All state lives in a Postgres instance you control.
- **Secure by default.** Modern password storage, short-lived access tokens, HttpOnly refresh cookies, signed join tickets for real-time connections. *(Full details in the hands-on engineering documentation.)*

**Business value:** a credible answer to "where is our meeting data?" — one that holds up in procurement, security review, and regulated industries.

---

## 👤 Personal touch

Rekall is also a tool individuals spend real time in. It respects that.

- **Profile & settings.** Display name, avatar initials, password change.
- **Appearance.** Dark theme by default, built to be easy on the eyes for long call days.
- **Session persistence.** Close your laptop and come back — your session survives a refresh without re-prompting for credentials on every navigation.

**Business value:** the product feels like a home base, not a pop-up window.

---

## What's coming next

The foundation is live. The **recall and insight layers** come next — automatic transcription (ASR), AI Notes, AI Ask across the full meeting memory, topic and sentiment signals, and organization- and department-level dashboards that show how effectively your team actually meets.

See the [Roadmap](ROADMAP.md) for the path forward.
