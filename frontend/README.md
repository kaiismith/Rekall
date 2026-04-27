# Rekall — Frontend

React + Vite + TypeScript single-page app for the Rekall call-intelligence
platform. Talks to the Go backend at [`backend/`](../backend) over HTTP / WS,
optionally renders live captions from the C++ ASR service at [`asr/`](../asr).

For the platform overview, deployment paths, and full feature list, see the
root [`README.md`](../README.md).

---

## Local development

```bash
cd frontend
npm install              # also installs the pre-commit hook (see below)
npm run dev              # Vite at http://localhost:5173
```

The app expects a Rekall backend reachable at `VITE_API_BASE_URL` (defaults to
`/api/v1` — Vite proxies in `vite.config.ts`).

## Scripts

| Command                 | What it does                                                                         |
| ----------------------- | ------------------------------------------------------------------------------------ |
| `npm run dev`           | Vite dev server with HMR                                                             |
| `npm run build`         | Type-check (`tsc -p tsconfig.build.json`) + production bundle                        |
| `npm run preview`       | Serve the built bundle locally                                                       |
| `npm run test`          | Vitest one-shot                                                                      |
| `npm run test:watch`    | Vitest watch mode                                                                    |
| `npm run test:ui`       | Vitest browser UI                                                                    |
| `npm run test:coverage` | Vitest + V8 coverage                                                                 |
| `npm run test:hook`     | Integration test for the pre-commit hook (Linux/macOS only)                          |
| `npm run lint`          | ESLint, zero warnings allowed                                                        |
| `npm run lint:fix`      | ESLint with `--fix`                                                                  |
| `npm run format`        | Prettier `--write` over `src/`                                                       |
| `npm run format:check`  | Prettier `--check` (CI-safe)                                                         |
| `npm run typecheck`     | `tsc -p tsconfig.build.json --noEmit` (matches CI's build scope — excludes `tests/`) |

---

## Pre-commit hook

A local git hook runs on every `git commit` and:

1. Auto-formats staged files with **prettier** (`--write`)
2. Auto-fixes staged `.ts` / `.tsx` files with **eslint** (`--fix --max-warnings 0`)
3. Runs a project-wide **type check** (`tsc -p tsconfig.build.json --noEmit`,
   matching CI's build scope) when at least one staged file is `.ts` or `.tsx`

The hook is wired through [husky](https://typicode.github.io/husky/) +
[lint-staged](https://github.com/lint-staged/lint-staged), both installed as
frontend dev-dependencies. Wall-clock cost on a typical commit is ~2–3 seconds.

### Installation

The hook is installed automatically the first time you run `npm install` from
`frontend/`. The `prepare` script invokes
[`scripts/install-husky.cjs`](./scripts/install-husky.cjs) (a small Node helper —
cross-platform; the equivalent POSIX one-liner does not work under Windows
`cmd.exe`). It calls husky's CLI, which in turn:

- Sets `git config core.hooksPath .husky/_` (husky v9 points git at the
  generated `_/` subdirectory, not at `.husky/` itself)
- Materialises husky's helper scripts under [`.husky/_/`](../.husky/) (gitignored)
- Each generated `_/<hook>` stub sources `_/h`, which in turn dispatches to the
  user-authored hook one level up — so OUR tracked
  [`.husky/pre-commit`](../.husky/pre-commit) is what actually runs

You can verify the install with:

```bash
git config --get core.hooksPath        # should print: .husky/_
ls ../.husky/_                         # should list pre-commit + helpers
```

### Bypassing the hook

The hook is a developer convenience, not an enforcement gate. CI re-runs every
check, so a bypass on `main` will still be caught — bypass is for handoffs and
emergency commits, not the steady state.

| How                                | When                                       |
| ---------------------------------- | ------------------------------------------ |
| `git commit --no-verify` (or `-n`) | One-off bypass for a specific commit       |
| `HUSKY=0 git commit ...`           | One-off bypass via env var                 |
| `HUSKY=0` exported in your shell   | Disable the hook for this terminal session |

The Dockerfile sets `ENV HUSKY=0` in its npm-install layer so image builds
never attempt to install hooks (there is no `.git` in the build context).

### Manual smoke test

```bash
# From repo root, with frontend/node_modules already installed.
echo 'export const x: any = 1' > frontend/src/bad.ts
git add frontend/src/bad.ts
git commit -m "smoke"
# Expected: hook fires, eslint reports no-explicit-any, commit blocked.

# Cleanup
git restore --staged frontend/src/bad.ts
rm frontend/src/bad.ts
```

### Editor integration

The hook is a backstop, not the primary feedback loop. For a faster inner loop,
enable format-on-save in your editor and use the ESLint extension to surface
problems inline:

- **VS Code**: install the Prettier and ESLint extensions; set
  `"editor.formatOnSave": true` and `"editor.defaultFormatter": "esbenp.prettier-vscode"`
- **JetBrains (WebStorm / IDEA Ultimate)**: enable Prettier under Settings →
  Languages & Frameworks → JavaScript → Prettier; enable "On code reformat"
  and "On save"

### Troubleshooting

| Symptom                                            | Fix                                                                                                                                                |
| -------------------------------------------------- | -------------------------------------------------------------------------------------------------------------------------------------------------- |
| `npx: lint-staged: not found` on commit            | Run `npm install` from `frontend/`                                                                                                                 |
| Hook does not fire after fresh clone               | Run `npm install` from `frontend/` once to trigger the `prepare` script. Verify `git config --get core.hooksPath` prints `.husky/_`                |
| Hook fails on Windows with `bad interpreter`       | Ensure Git for Windows is installed (provides `sh.exe`); `git config --global core.autocrlf false` if line-ending rewrites are mangling the script |
| `tsc` reports an error in a file you did not touch | A staged change broke an unstaged importer — that is the point of the project-wide check. Fix the importer or revert the breaking change           |
| Hook is too slow                                   | Use `git commit --no-verify` for the immediate commit; report friction so the hook can be tuned                                                    |

---

## Project layout

```
frontend/
├── src/                  # app source — components, hooks, types, services
├── tests/                # vitest unit + integration tests
├── public/               # static assets served verbatim
├── eslint.config.js      # flat-config ESLint (TypeScript-aware rules)
├── .eslintrc.json        # legacy fallback config (kept for editor compat)
├── .prettierrc           # prettier formatting rules
├── .prettierignore       # files prettier must never rewrite
├── tsconfig.json         # editor / dev mode (includes tests/)
├── tsconfig.build.json   # production build + `npm run typecheck` (excludes tests/)
├── tsconfig.node.json    # vite.config.ts itself
├── vite.config.ts        # Vite dev / build config
├── vitest.config.ts      # Vitest config
├── tailwind.config.ts    # Tailwind CSS theme + content globs
└── postcss.config.js
```

The pre-commit hook script itself lives at the repo root: [`.husky/pre-commit`](../.husky/pre-commit).
