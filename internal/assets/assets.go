package assets

import _ "embed"

//go:embed domains.json
var DomainsJSON []byte

//go:embed karlsson.css
var KarlssonCSS []byte
