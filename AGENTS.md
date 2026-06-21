# Offprint — Agent Briefing

> Read before every change. Keep this file short and append concrete gotchas as they are discovered.

Offprint is a personal-first Go CLI that archives online publications as Markdown, HTML, and printable PDFs.

## Commands

- Build: `go build ./cmd/offprint`
- Run: `go run ./cmd/offprint --help`
- Test: `go test ./...`
- Full gate: `bin/ci`
- Install hooks after cloning: `bin/install-hooks`

## Working rules

- Plan non-trivial changes before editing. Prefer the smallest design that solves a current use case.
- Make it work, then right, then fast. Do not add platforms, plugins, or abstractions for hypothetical users.
- Add a test with every feature and a regression test with every bug fix. Unit tests must not require real credentials or network access.
- Keep network, filesystem, authentication, and Chromium boundaries explicit. Never print cookies or store them in the repository.
- Use `internal/assets` for defaults required at runtime; installed binaries must not depend on the checkout or current directory.
- Preserve stdout for results and stderr for diagnostics. Errors must be actionable and return non-zero.

## Git

- Default branch: `master`. Use Conventional Branch names and Conventional Commits.
- Keep commits atomic and green. Never stage blindly; inspect `git status` and `git diff --cached` first.
- PR descriptions explain what changed and why. Comments explain intent, not mechanics.
- Never commit cookies, generated archives, cached content, private fonts, or other secrets. The gitleaks hook is mandatory.

## Security and releases

- Treat URLs, slugs, paths, cookies, downloaded HTML, and custom CSS/fonts as untrusted input.
- Run `bin/ci` before declaring work complete. Before a release, also review the change manually for credential and filesystem risks.
- Releases are created from `v*` tags through GoReleaser; do not publish ad hoc binaries.

## Common hurdles

- Substack endpoints are internal APIs and can change without notice.
- `archive` needs a full authenticated Cookie header for paid posts; `__cf_bm` alone is insufficient.
- PDF generation requires Chrome or Chromium; Markdown export does not.
- `--disable-web-security` weakens Chromium isolation and must remain opt-in.
