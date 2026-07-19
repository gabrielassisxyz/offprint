# Offprint — Agent Briefing

> Read before every change. Living spec: short, imperative. On every gotcha or decision discovered, append one line here.

> **What it is:** a Go CLI that archives online publications as Markdown, HTML, and printable PDFs.

## Scope (current)

- **Current scope:** archives publications to Markdown / HTML / PDF for one user's library. Don't expand beyond it without a present need; if a change drifts past it, STOP and flag it.

## Commands

- Build: `go build ./cmd/offprint`
- Run: `go run ./cmd/offprint --help`
- Test: `go test ./...`
- Full gate: `bin/ci`
- Install hooks after cloning: `bin/install-hooks`

<!-- BEGIN universal-principles v1 -->
## Working principles

- **The human defines the WHAT; the agent decides the HOW.** Don't wait for line-by-line dictation. Plan first for non-trivial tasks: show the plan + to-do list, wait for approval.
- **Think before coding — don't assume, don't hide confusion.** State assumptions explicitly; if multiple interpretations exist, present them — don't pick silently. If a simpler approach exists, say so and push back. If a task is impossible under the stated constraints, or info is missing, say so — don't guess. (For trivial tasks, use judgment; this is bias, not ritual.)
- **Surgical changes — touch only what you must.** Every changed line traces to the task. Don't "improve" adjacent code, reformat, or refactor what isn't broken; match existing style even if you'd do it differently. Flag unrelated dead code — don't delete it. Remove only the imports / variables / functions your own change orphaned.
- **Goal-driven execution — define the success check, then loop to it.** Turn the task into something verifiable before coding: "add validation" → write tests for invalid inputs, then pass them; "fix the bug" → write a failing repro test, then pass it; "refactor X" → tests green before and after. For multi-step work, state a brief plan with a verify step each.
- **KISS — don't solve a problem you don't have yet.** Simplicity isn't "write less code"; it's not building for a need that doesn't exist. Let structure emerge from the code.
- **YAGNI & flat.** No preventive abstractions, no single-use interfaces. Interfaces for real boundaries only. Architecture is *extracted* once a pattern proves itself in real use — never designed up front for a user who doesn't exist yet. Need pulls architecture.
- **Order: make it work → make it right → make it fast** (Kent Beck), in that order. Most over-engineering is doing "right"/"fast" before a working thing exists to justify it.
- **Flag scope creep — a standing duty, not a suggestion.** When a solo tool starts being framed as a public / multi-user / multi-tenant / plugin-system / configurable-N-backends platform before a real, present need exists, STOP and ask: "Is this needed now?" Justify future-proofing against a need that exists *today*.
- **No silent decisions (comprehension debt).** Never make a silent architectural or design call — state it and record the rationale, so the reasoning is recoverable later.

## Git: branches, commits, PRs, comments

- **Default branch is `main`.** Never commit directly to it; branch, then PR.
- **Branches** — Conventional Branch (conventionalbranch.org): `<type>/<kebab-description>`, types `feature/`, `bugfix/`, `hotfix/`, `chore/`, `release/`, `docs/`.
- **Commits** — Conventional Commits (conventionalcommits.org): `<type>(scope): <description>`, types `feat`, `fix`, `docs`, `refactor`, `test`, `chore`, `ci`, `build`, `perf`, `style`. Breaking change → `!` after the type or a `BREAKING CHANGE:` footer.
- **Atomic commits** — one logical change per commit, each independently green and revertible. Never `git add .` blind; split unrelated changes.
- **Pull requests** — describe **what + why**. *What*: a 1–3 line summary. *Why* (the bulk): decisions, trade-offs, rejected alternatives. The diff shows the what; the PR explains why.
- **Comments** — always **WHY, not WHAT**: explain intent, never restate the obvious mechanics. Keep existing comments; they carry intent.

## Code style (baseline)

- Functions: 4–40 lines, one thing each (SRP). Files: under ~500 lines, split by responsibility.
- Names specific and unique — avoid `data`, `handler`, `Manager`, `util`.
- Explicit types. Early returns over nested ifs; max ~2 levels of indentation.
- Inject dependencies; wrap third-party libs behind a thin interface this project owns.
- No duplication — but don't extract *too early*. Tolerate duplication while the pattern is still forming; extract the abstraction *from* proven, repeated code, never ahead of it.
- **Refactoring is not automatic.** After a large feature, list refactoring candidates (files > ~500 lines, duplicated logic, long functions, hardcoded config) and ask before pruning — the human decides, the tests are the safety net. Consolidate when the thing works and the seams are obvious, not before.
<!-- END universal-principles v1 -->

## Project rules & boundaries

- Add a test with every feature and a regression test with every bug fix. Unit tests must not require real credentials or network access.
- Keep network, filesystem, authentication, and Chromium boundaries explicit. Never print cookies or store them in the repository.
- Use `internal/assets` for defaults required at runtime; installed binaries must not depend on the checkout or current directory.
- Preserve stdout for results and stderr for diagnostics. Errors must be actionable and return non-zero.
- Never commit cookies, generated archives, cached content, private fonts, or other secrets. The gitleaks pre-commit hook is the deterministic backstop; inspecting `git diff --cached` before committing is the habit.

## Security and releases

- Treat URLs, slugs, paths, cookies, downloaded HTML, and custom CSS/fonts as untrusted input.
- Run `bin/ci` before declaring work complete. Before a release, also review the change manually for credential and filesystem risks.
- Releases are created from `v*` tags through GoReleaser; do not publish ad hoc binaries.

## Common hurdles

- Substack endpoints are internal APIs and can change without notice.
- `archive` needs a full authenticated Cookie header for paid posts; `__cf_bm` alone is insufficient.
- PDF generation requires Chrome or Chromium; Markdown export does not.
- `--disable-web-security` weakens Chromium isolation and must remain opt-in.
