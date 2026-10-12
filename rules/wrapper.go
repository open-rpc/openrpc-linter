package rules

import (
	"cmp"
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
	Rules   map[string]types.Rule `yaml:"rules,omitempty"`
}

func (rw *RulesWrapper) CheckRules() error {
	if len(rw.Rules) == 0 && len(rw.Extends) == 0 {
		return fmt.Errorf("no rules to merge, rules and extends are empty invalid rules")
	}
	return nil
}

func (rw *RulesWrapper) ResolvedRules() (map[string]types.Rule, error) {
	merged, err := getExtendedRules(rw.Extends)
	if err != nil {
		return nil, err
	}

	return mergeRules(merged, rw.Rules), nil
}

func mergeRule(base types.Rule, override types.Rule) types.Rule {
	return types.Rule{
		Description: cmp.Or(override.Description, base.Description),
		Given:       cmp.Or(override.Given, base.Given),
		Then:        cmp.Or(override.Then, base.Then),
		Extends:     inheritedExtensions(base.Extends, override.Extends),
		Severity:    cmp.Or(override.Severity, base.Severity),
	}
}

// inheritedExtensions keeps the base only when the override is absent.
func inheritedExtensions(base, override any) any {
	if override != nil {
		return override
	}
	return base
}

func mergeRules(base map[string]types.Rule, overrides map[string]types.Rule) map[string]types.Rule {
	merged := make(map[string]types.Rule, len(base))
	maps.Copy(merged, base)
	for id, overrideRule := range overrides {
		base, exists := base[id]
		if exists {
			merged[id] = mergeRule(base, overrideRule)
			continue
		}
		merged[id] = overrideRule
	}
	return merged
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

func LoadRulesFile(fsys fs.FS, filename string) (*RulesWrapper, error) {
	b, err := fs.ReadFile(fsys, filename)
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
