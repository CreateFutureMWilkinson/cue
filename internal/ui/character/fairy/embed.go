package fairy

import "embed"

//go:embed assets/jar_back.png
var jarBackPNG []byte

//go:embed assets/jar_front.png
var jarFrontPNG []byte

// Ensure embed is used.
var _ embed.FS
