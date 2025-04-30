package ne

import "embed"

//go:embed *.zip
var EmbeddedNaturalEarthData embed.FS
