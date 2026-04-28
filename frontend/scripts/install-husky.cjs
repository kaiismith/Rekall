#!/usr/bin/env node
// Cross-platform husky install for the frontend pre-commit hook.
//
// Why a script instead of a one-liner npm prepare command:
//   - On Windows, npm runs scripts via cmd.exe — POSIX redirection (`>/dev/null`)
//     and shell chaining quirks break.
//   - We want a single behaviour across Linux/macOS/Windows: silently no-op
//     when not in a git repo, otherwise install hooks at the repo-root .husky/.
//
// Husky v9 specifics that shaped this script:
//   - `husky <dir>` REJECTS paths containing `..`, so we must cd to the
//     repo root (where .husky/ is a direct child) before invoking it.
//   - Husky v9 sets `core.hooksPath` to `<dir>/_` (not `<dir>`) and writes
//     stubs in `<dir>/_/<hook>` that source the runtime at `<dir>/_/h`.
//     The runtime then dispatches to `<dir>/<hook>` — so OUR tracked
//     hook script at .husky/pre-commit is what actually runs.

const { execSync, spawnSync } = require('node:child_process')
const path = require('node:path')
const fs = require('node:fs')

function quietExec(cmd) {
  try {
    return execSync(cmd, { stdio: 'pipe' }).toString().trim()
  } catch {
    return null
  }
}

if (process.env.HUSKY === '0') {
  // Honour the documented bypass.
  process.exit(0)
}

const repoRoot = quietExec('git rev-parse --show-toplevel')
if (!repoRoot) {
  // Not a git repo (e.g. extracted tarball, Docker build context). No-op.
  process.exit(0)
}

// Verify the repo-root .husky directory exists with our tracked hook.
const huskyDir = path.join(repoRoot, '.husky')
if (!fs.existsSync(path.join(huskyDir, 'pre-commit'))) {
  // Hook script missing — likely a partial checkout. Skip silently rather
  // than installing a half-configured hook directory.
  process.exit(0)
}

// Locate the husky binary inside frontend/node_modules.
const huskyBin = path.join(
  __dirname,
  '..',
  'node_modules',
  '.bin',
  process.platform === 'win32' ? 'husky.cmd' : 'husky',
)

if (!fs.existsSync(huskyBin)) {
  // husky devDep not installed yet (running before `npm install` finishes
  // resolving devDependencies). Nothing to do.
  process.exit(0)
}

// Run husky from the repo root so:
//   - the path argument can be relative (no `..`),
//   - husky's own `.git` existence check passes.
// Husky itself sets core.hooksPath to .husky/_ as part of this call.
const inst = spawnSync(huskyBin, ['.husky'], {
  cwd: repoRoot,
  stdio: 'inherit',
  shell: process.platform === 'win32',
})
process.exit(inst.status === null ? 0 : inst.status)
