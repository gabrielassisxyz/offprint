package assets

import _ "embed"

// BaseCSS is the generic printable-document theme shipped with Offprint.
//
//go:embed base.css
var BaseCSS string

// SitesJSON contains built-in extraction profiles. Site-specific profiles are
// intentionally sparse and must be backed by fixtures and tests.
//
//go:embed sites.json
var SitesJSON []byte
