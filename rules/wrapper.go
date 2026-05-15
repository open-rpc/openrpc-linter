package rules

import (
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path/filepath"

	"github.com/open-rpc/openrpc-linter/types"
	"gopkg.in/yaml.v3"
)

type RulesWrapper struct {
	Extends []types.RuleDefaults  `yaml:"extends,omitempty"`
	Rules   map[string]types.Rule `yaml:"rules"`
}

func (rw *RulesWrapper) ResolvedRules() (map[string]types.Rule, error) {
	merged, err := getExtendedRules(rw.Extends)
	if err != nil {
		return nil, err
	}
	maps.Copy(merged, rw.Rules) // user rules win
	return merged, nil
}

func getExtendedRules(exts []types.RuleDefaults) (map[string]types.Rule, error) {
	out := make(map[string]types.Rule)
	for _, ext := range exts {
		switch ext {
		case types.RuleExtensionRecommended:
			w, err := LoadRulesFile(GetRuleDefaultsFS(), string(ext)+".yaml")
			if err != nil {
				return nil, fmt.Errorf("loading rule extension %q: %w", ext, err)
			}
			maps.Copy(out, w.Rules)
		default:
			return nil, fmt.Errorf("unknown rule extension %q", ext)
		}
	}
	return out, nil
}

func LoadRulesFile(fsys fs.FS, name string) (*RulesWrapper, error) {
	b, err := fs.ReadFile(fsys, name)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error reading rules file: %v\n", err)
		return nil, err
	}
	var rw RulesWrapper
	err = yaml.Unmarshal(b, &rw)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error parsing rules file: %v\n", err)
		return nil, err
	}
	return &rw, nil
}

func LoadRulesFileFromPath(path string) (*RulesWrapper, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	return LoadRulesFile(os.DirFS(filepath.Dir(abs)), filepath.Base(abs))
}
