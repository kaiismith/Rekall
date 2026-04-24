# Rekall

> **The workspace where every call is remembered.**

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

## Documentation

| | |
|---|---|
| 🎯 [Overview](.docs/OVERVIEW.md) | What Rekall is, why it exists, and who it's for |
| ✨ [Features](.docs/FEATURES.md) | What you can do with Rekall today |
| 🗺️ [Roadmap](.docs/ROADMAP.md) | Where Rekall is headed next |

---

## License

See [LICENSE](LICENSE).
