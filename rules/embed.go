package rules

import (
	"embed"
	"io/fs"
)

//go:embed defaults/*.yaml
var ruleExtensionsFS embed.FS

func GetRuleDefaultsFS() fs.FS {
	sub, _ := fs.Sub(ruleExtensionsFS, "defaults")
	return sub
}
