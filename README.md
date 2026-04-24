# Rekall

> **The workspace where every call is remembered.**

Rekall is a self-hostable **call intelligence platform** — a place for teams to meet, talk, and later look back at what was said, decided, and agreed. Think of it as the space between a video-conferencing tool and a shared organizational memory: meetings happen *inside* Rekall, and the knowledge from them stays there.

---

## Why Rekall?

Most meetings evaporate the moment they end. Notes get scattered across chats, decisions get lost, and action items quietly go unowned. Teams repeat the same conversations because nobody can find what was said three weeks ago.

Rekall is built on a simple premise:

> **If the meeting happened in Rekall, the team shouldn't have to remember it — Rekall already does.**

That means real-time conversation *and* durable recall, in one place, owned by your organization.

---

## What you get

- 🏢 **A workspace.** Organizations, departments, invitations — a real team directory, not just a login screen.
- 🎥 **Video meetings that work.** Shareable join codes, peer-to-peer media, waiting-room knock flow, host controls. No client to install.
- 🙋 **A polished room.** Pre-meeting device check, virtual backgrounds, in-room chat, raise hand, emoji reactions, screen share.
- 📞 **Persistent call history.** Every meeting becomes a record in the workspace, ready for what comes next.
- 🔒 **Self-hosted by design.** Your video, your transcripts, your data — on infrastructure you control.
- 🧠 **A path to organizational memory.** Transcription, AI Notes, AI Ask, topic and sentiment classification, and observability dashboards are on the roadmap. See [`ROADMAP.md`](.docs/ROADMAP.md).

For the full picture, read the [Overview](.docs/OVERVIEW.md) and [Features](.docs/FEATURES.md).

---

## Who it's for

- **Small to mid-sized teams** who want a self-hosted alternative to *"Zoom + Notion + Otter"* stitched together.
- **Engineering-led companies** that prefer to own their data and run their own infrastructure.
- **Distributed teams** that need meetings to leave a trail behind them — decisions, action items, agreements.
- **Organizations in regulated spaces** (healthcare, legal, finance) where meeting recordings and transcripts must stay inside the org's perimeter.

If you're looking for a video-call tool that's *just* a video-call tool, Rekall is probably overkill. If you're looking for the memory layer of your organization, keep reading.

---

## Quickstart

Rekall runs anywhere Docker runs.

### Prerequisites

- [Docker](https://www.docker.com/) and Docker Compose
- 2 GB free RAM
- Ports `3000`, `8080`, `5432`, `8025` available

### Start it up

```bash
make up
```

That's it. The first run takes a few minutes while images build.

### Open the app

| | |
|---|---|
| **Rekall** | http://localhost:3000 |
| **Mail inbox** (dev) | http://localhost:8025 |
| **API** | http://localhost:8080 |

Register the first account, create an organization, and you're in.

### Other commands

```bash
make down        # stop everything
make logs        # tail container logs
make restart     # rebuild and restart
make test        # run backend + frontend tests
```

---

## Project status

Rekall is in **active early development**. The meeting and workspace layers are usable today; the recall and AI layers — transcription, summaries, AI Ask, dashboards — are what come next.

See the [Roadmap](.docs/ROADMAP.md) for the path forward.

---

## Documentation

| | |
|---|---|
| 🎯 [Overview](.docs/OVERVIEW.md) | What Rekall is, why it exists, and who it's for |
| ✨ [Features](.docs/FEATURES.md) | What you can do with Rekall today |
| 🗺️ [Roadmap](.docs/ROADMAP.md) | Where Rekall is headed next |

---

## License

See [LICENSE](LICENSE).
