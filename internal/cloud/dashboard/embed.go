package dashboard

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
)

//go:embed static
var StaticFS embed.FS

// StylesVersion is a short content hash of the embedded stylesheet. Pages
// append it to the stylesheet URL so browser and CDN caches never serve a
// stylesheet from a previous deploy.
var StylesVersion = contentVersion("static/styles.css")

func contentVersion(name string) string {
	data, err := StaticFS.ReadFile(name)
	if err != nil {
		return "dev"
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])[:12]
}
