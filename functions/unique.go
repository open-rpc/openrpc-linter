package functions

import (
	"fmt"

	"github.com/open-rpc/openrpc-linter/selector"
	"github.com/open-rpc/openrpc-linter/types"
	"github.com/theory/jsonpath"
	"github.com/theory/jsonpath/spec"
)

const globalScopeKey = ""

type UniqueRule struct {
	scope *uniqueScope
}

type uniqueScope struct {
	paths []spec.NormalizedPath
	err   error
	seen  map[string]map[primitiveKey]string
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
	if err := r.ensureScopes(context); err != nil {
		return ruleFailure("unique scope option invalid: "+err.Error(), path)
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

// uniqueCandidate applies presence policy and validates primitive identity.
func uniqueCandidate(value any, context types.RuleFunctionContext) (primitiveKey, bool, error) {
	ignoreMissing, err := uniqueIgnoreMissing(context)
	if err != nil {
		return primitiveKey{}, false, err
	}
	if missingField(context.Target) {
		if ignoreMissing {
			return primitiveKey{}, false, nil
		}
		value = nil
	}
	key, supported := comparableKey(value)
	if !supported {
		return primitiveKey{}, false, fmt.Errorf("unique value must be primitive")
	}
	return key, true, nil
}

// remember owns the per-scope first-seen state. It never replaces a first path.
func (s *uniqueScope) remember(scopeKey string, key primitiveKey, path string) (string, bool) {
	if s.seen == nil {
		s.seen = map[string]map[primitiveKey]string{}
	}
	bucket, exists := s.seen[scopeKey]
	if !exists {
		bucket = map[primitiveKey]string{}
		s.seen[scopeKey] = bucket
	}
	if firstPath, duplicate := bucket[key]; duplicate {
		return firstPath, true
	}
	bucket[key] = path
	return "", false
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

func (r *UniqueRule) ensureScopes(ctx types.RuleFunctionContext) error {
	if r.scope != nil {
		return r.scope.err
	}
	paths, err := uniqueScopePaths(ctx)
	r.scope = &uniqueScope{paths: paths, err: err}
	return err
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
	var paths []spec.NormalizedPath
	for _, node := range p.SelectLocated(doc) {
		paths = append(paths, node.Path)
	}
	return paths, nil
}

func (s *uniqueScope) keyFor(t *selector.Target) (string, bool) {
	if len(s.paths) == 0 {
		return globalScopeKey, true
	}
	bestIdx, bestLen := -1, -1
	for i, sp := range s.paths {
		if selector.IsUnder(t.Path, sp) && len(sp) > bestLen {
			bestIdx, bestLen = i, len(sp)
		}
	}
	if bestIdx < 0 {
		return "", false
	}
	return s.paths[bestIdx].String(), true
}
