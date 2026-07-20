# Roadmap

What exists, what is missing, and what is deliberately out of scope. See
[BACKLOG.md](BACKLOG.md) for the working list behind the "missing" section.

## What exists today

- `archive`: crawls a full Substack publication — including subscriber-only posts, given an authenticated cookie — and writes one Markdown file per post; accepts a single URL or a URL-list file.
- `bundle`: turns one article URL or a URL list into a consolidated HTML document or print-ready A4 PDF with table of contents and references; PDF rendering uses Chrome/Chromium, HTML never launches a browser.
- Cookie handling via environment variable, `--cookie-file`, or a per-domain `cookies set` store in the OS user config directory, written with user-only permissions.
- Site extraction profiles (`sites/*.json`) plus custom CSS and fonts, with generic extraction fallbacks and a print stylesheet shipped in `internal/assets`.
- Cross-platform releases from `v*` tags via GoReleaser, a checksum-verifying `install.sh`, a gitleaks pre-commit hook, and a single `bin/ci` gate that runs identically in GitHub Actions.

## Missing / natural next steps

- Fixture-based tests for the generic extraction fallback and user site profiles.
- Maintained built-in JSON profiles for well-known publishing platforms, starting with Medium and Substack web pages — shipped only when backed by fixtures and extraction tests.
- `offprint serve`: a local web application for managing archived articles, editing settings, previewing PDF configuration, and turning a visual selection into a bundle.
- A `Dockerfile` and `docker-compose.yml` for running `offprint serve`, once the server workflow exists.
- Hardening of what is currently heuristic: PDF extraction needs custom selectors on some sites, and Substack export depends on internal API endpoints that may change.

## Deliberately out of scope

- Anything beyond archiving publications to Markdown/HTML/PDF for one user's library — multi-user or platform ambitions are flagged and rejected rather than drifted into.
- Speculative source and output abstractions: domain packages are extracted only when additional sources or bundle formats create proven boundaries.
- Bundled proprietary fonts; a font is always supplied locally via `--font`.
- Direct merging of external patches: bug reports are welcome, but submitted patches may be reviewed as references and reimplemented.
