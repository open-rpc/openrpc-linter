package functions

import (
	"fmt"
	"sort"

	"github.com/open-rpc/openrpc-linter/selector"
	"github.com/open-rpc/openrpc-linter/types"
	"github.com/theory/jsonpath"
	"github.com/theory/jsonpath/spec"
)

const globalScopeKey = ""

type UniqueRule struct {
	scope *uniqueScope
}

func NewUniqueRule() *UniqueRule {
	return &UniqueRule{}
}

func (r *UniqueRule) RunRule(value interface{}, context types.RuleFunctionContext) []types.RuleFunctionResult {
	t := context.Target
	if t == nil {
		return nil
	}

	path := t.PathString()
	if r.scope == nil {
		paths, err := uniqueScopePaths(context)
		r.scope = &uniqueScope{paths: paths, err: err}
	}
	if r.scope.err != nil {
		return ruleFailure("unique scope option invalid: "+r.scope.err.Error(), path)
	}

	scopeKey, inScope := r.scope.keyFor(t)
	if !inScope {
		return nil
	}

	key, included, err := uniqueCandidate(value, context)
	if err != nil {
		return ruleFailure(err.Error(), path)
	}
	if !included {
		return nil
	}
	firstPath, duplicate := r.scope.remember(scopeKey, key, path)
	if duplicate {
		return ruleFailure(fmt.Sprintf("Duplicate value %s (first seen at %s)", key.display(), firstPath), path)
	}
	return nil
}

type uniqueScope struct {
	// Nil paths mean global scope; an empty slice means no scope matches.
	paths []spec.NormalizedPath
	err   error
	seen  map[scopedValue]string
}

// remember owns the per-scope first-seen state. It never replaces a first path.
func (s *uniqueScope) remember(scopeKey string, key primitiveKey, path string) (string, bool) {
	if s.seen == nil {
		s.seen = make(map[scopedValue]string)
	}
	identity := scopedValue{scope: scopeKey, value: key}
	if firstPath, duplicate := s.seen[identity]; duplicate {
		return firstPath, true
	}
	s.seen[identity] = path
	return "", false
}

func (s *uniqueScope) keyFor(t *selector.Target) (string, bool) {
	if s.paths == nil {
		return globalScopeKey, true
	}
	for _, path := range s.paths {
		if selector.IsUnder(t.Path, path) {
			return path.String(), true
		}
	}
	return "", false
}

// scopedValue makes scope and primitive identity one collision-safe map key.
type scopedValue struct {
	scope string
	value primitiveKey
}

// uniqueCandidate applies presence policy and validates primitive identity.
func uniqueCandidate(value any, context types.RuleFunctionContext) (primitiveKey, bool, error) {
	ignoreMissing, err := uniqueIgnoreMissing(context)
	if err != nil {
		return primitiveKey{}, false, err
	}
	if missingField(context.Target) {
		return primitiveKey{kind: primitiveNull}, !ignoreMissing, nil
	}
	key, supported := comparableKey(value)
	if !supported {
		return primitiveKey{}, false, fmt.Errorf("unique value must be primitive")
	}
	return key, true, nil
}

func uniqueIgnoreMissing(ctx types.RuleFunctionContext) (bool, error) {
	raw, exists := functionOptions(ctx)["ignoreMissing"]
	if !exists {
		return true, nil
	}

	value, ok := raw.(bool)
	if !ok {
		return false, fmt.Errorf("unique function option ignoreMissing must be a boolean")
	}

	return value, nil
}

func uniqueScopePaths(ctx types.RuleFunctionContext) ([]spec.NormalizedPath, error) {
	raw, _ := functionOptions(ctx)["scope"].(string)
	if raw == "" {
		return nil, nil
	}
	p, err := jsonpath.Parse(raw)
	if err != nil {
		return nil, err
	}
	doc := ctx.Document
	if ctx.ResolvedDocument != nil {
		doc = ctx.ResolvedDocument
	}
	paths := make([]spec.NormalizedPath, 0)
	for _, node := range p.SelectLocated(doc) {
		paths = append(paths, node.Path)
	}
	// Most-specific scopes come first. Selection order breaks equal-depth ties.
	sort.SliceStable(paths, func(i, j int) bool { return len(paths[i]) > len(paths[j]) })
	return paths, nil
}
