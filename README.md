# Offprint

Archive online publications as Markdown, HTML, and print-ready PDFs.

Offprint is a personal-first Go CLI for keeping durable, distraction-free copies of writing from the web. Its primary workflow crawls an entire Substack publication—including posts available to an authenticated paid subscriber—and writes one Markdown file per post. It can also turn a URL list into a consolidated HTML document or printable A4 PDF.

## Status

Offprint is usable but still early. Substack export depends on internal API endpoints that may change. PDF extraction is heuristic and some sites need custom selectors.

## Install

Prebuilt binaries are published for Linux, macOS, and Windows on each `v*` release.

```sh
curl -fsSL "https://raw.githubusercontent.com/gabrielassisxyz/offprint/main/install.sh?$(date +%s)" | bash
```

The installer verifies the release checksum and writes to `~/.local/bin` by default. Alternatively, download a release manually or build from source:

```sh
go install github.com/gabrielassisxyz/offprint/cmd/offprint@latest
```

PDF generation additionally requires Chrome or Chromium. If none is installed, Offprint says so before it downloads anything and offers to install one for you. `--format html` and the `archive` command never launch a browser and need none.

## Quick start

Offprint writes to `~/Documents/Offprint` by default. Every command accepts `--output` to choose another directory.

### Export a Substack archive to Markdown

Copy the complete request `Cookie` header from an authenticated Substack API request. Do not use only `__cf_bm`; it is not the subscriber session.

Load it without placing the value in shell history:

```sh
read -rs SUBSTACK_COOKIE
export SUBSTACK_COOKIE

offprint archive https://publication.example/archive
unset SUBSTACK_COOKIE
```

The result is `~/Documents/Offprint/publication.example/{slug}.md`. Existing files are replaced, making repeated exports safe.

To avoid environment variables, save the Cookie header in a protected file and pass `--cookie-file`:

```sh
offprint archive --cookie-file ~/.config/offprint/substack.cookie \
  https://publication.example/archive
```

### Export several publications

Create a text file containing one `/archive` URL per line. Empty lines and comments beginning with `#` are ignored.

```text
# publications.txt
https://first.example/archive
https://second.example/archive
```

```sh
offprint archive publications.txt
# Equivalent and unambiguous in scripts:
offprint archive --input publications.txt
```

### Bundle articles into HTML and PDF

`bundle` accepts either one article URL or a file containing URLs. It downloads and cleans each article, creates a table of contents and references, then combines everything into one printable document:

```sh
offprint bundle --name reading-list --format both links.txt
offprint bundle --format pdf https://example.com/an-article
offprint bundle --format html --output /tmp/offprint links.txt
```

Available formats are `markdown` through `archive`, and `html`, `pdf`, or `both` through `bundle`.

## Fonts

Generated documents use the system serif stack (`Georgia`, `Times New Roman`, then a generic serif) by default. A private font is not bundled with the project.

Pass a local TTF, OTF, or WOFF file when needed:

```sh
offprint bundle --font ~/Library/Fonts/MyFont.ttf links.txt
```

The font is embedded into the generated document. The local `fonts/` directory is ignored by Git and may be used for personal files.

## Cookies

Substack archive authentication reads `SUBSTACK_COOKIE` or `--cookie-file` and never persists it automatically.

The generic article renderer can persist cookies for a domain:

```sh
read -rs OFFPRINT_COOKIE
export OFFPRINT_COOKIE
offprint cookies set --domain example.com
unset OFFPRINT_COOKIE
```

The store defaults to the operating system's user configuration directory, such as `~/.config/offprint/cookies.json`, and is written with user-only permissions. Cookie values are plaintext credentials: protect and rotate them accordingly.

## Chromium security option

Chromium web security remains enabled by default. Some unusual local-page workflows may require:

```sh
offprint bundle --disable-web-security links.txt
```

This weakens browser isolation. Use it only with content and assets you trust.

## CSS and site extraction profiles

Offprint ships with a generic print stylesheet and generic HTML extraction fallbacks. Personal site profiles are loaded from the operating system's user configuration directory:

```text
~/.config/offprint/
├── sites/
│   └── example.json
└── styles/
    └── example.css
```

On macOS and Windows, Offprint uses the platform-specific user configuration directory. Use `--config-dir` to select another location.

Each file in `sites/*.json` maps domains to extraction settings. Relative `custom_css_path` values are resolved from the profile file, not the current working directory. Profiles loaded later override built-in profiles for the same domain. `--site-profile` loads one additional profile with highest precedence.

Use `--css` for document-wide styling independent of the source site:

```sh
offprint bundle --css ~/styles/study.css links.txt
offprint bundle --config-dir ~/my-offprint-config links.txt
```

The generic stylesheet is available at [`internal/assets/base.css`](internal/assets/base.css). Built-in site profiles live in [`internal/assets/sites.json`](internal/assets/sites.json); new profiles should only be added with fixtures and extraction tests.

Copy [`examples/sites/example.json`](examples/sites/example.json) and [`examples/styles/example.css`](examples/styles/example.css) into your configuration directory to start a personal profile without modifying the repository.

## Output and local state

| Data | Default location | Override |
|---|---|---|
| Archives and documents | `~/Documents/Offprint` | `--output` |
| HTTP/image/title cache | OS user cache directory under `offprint/` | `OFFPRINT_CACHE_DIR` |
| Profiles, styles and cookies | OS user config directory under `offprint/` | `--config-dir` / `--store` |
| Custom font | none | `--font PATH` |

## Project structure

```text
cmd/offprint/       Binary entry point
internal/app/       CLI orchestration and current application pipeline
internal/assets/    Generic CSS and tested built-in site profiles
bin/ci              Local and hosted CI gate
bin/install-hooks   Installs the versioned gitleaks pre-commit hook
.github/workflows/  CI and tag-triggered releases
```

The code intentionally starts with a cohesive `internal/app` package instead of speculative source and output abstractions. Domain packages should be extracted when additional sources or bundle formats create proven boundaries.

## Development

Requirements: Go 1.26.4 or newer, `gitleaks`, `golangci-lint`, and `govulncheck`.

```sh
bin/install-hooks
go test ./...
bin/ci
```

The exact same `bin/ci` command runs in GitHub Actions. Tests must use local HTTP test servers and fake credentials; never add real subscriber cookies or paid article content.

To test a release without publishing it:

```sh
goreleaser release --snapshot --clean
```

Pushing a tag such as `v0.1.0` builds cross-platform archives, generates `checksums.txt`, and publishes a GitHub Release.

## Security and responsible use

- Never commit or log authentication cookies.
- Remove secrets and paid content from bug reports.
- Archive and share subscriber-only material according to the subscription terms and applicable law.
- Treat custom fonts, CSS and downloaded pages as untrusted input.
- See [SECURITY.md](SECURITY.md) for vulnerability reporting.

## Guidance for coding agents

Read [AGENTS.md](AGENTS.md) before modifying the repository. Important invariants include authenticated requests never logging cookies, Markdown export never launching Chromium, installed binaries not depending on repository-relative files, and generated content never entering Git.

## Contributing

Bug reports are welcome. This remains primarily a personal project, so submitted patches may be reviewed as references and reimplemented rather than merged directly. See [CONTRIBUTING.md](CONTRIBUTING.md).

## License

[MIT](LICENSE)
