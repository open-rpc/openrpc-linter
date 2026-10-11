package functions

import (
	"fmt"
	"strconv"

	"github.com/open-rpc/openrpc-linter/selector"
	"github.com/open-rpc/openrpc-linter/types"
	"github.com/theory/jsonpath"
	"github.com/theory/jsonpath/spec"
)

const globalScopeKey = ""

type UniqueRule struct {
	seen  map[string]map[string]string
	scope *uniqueScope
}

type uniqueScope struct {
	paths []spec.NormalizedPath
	err   error
}

func NewUniqueRule() *UniqueRule {
	return &UniqueRule{seen: map[string]map[string]string{}}
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

	scopeKey, inScope := r.scopeKeyFor(t)
	if !inScope {
		return nil
	}

	ignoreMissing, err := uniqueIgnoreMissing(context)
	if err != nil {
		return ruleFailure(err.Error(), path)
	}
	if t.Field != "" && !t.Exists {
		if ignoreMissing {
			return nil
		}
		value = nil
	}
	key, display, supported := comparableKey(value)
	if !supported {
		return ruleFailure("unique value must be primitive", path)
	}

	firstPath, duplicate := r.remember(scopeKey, key, path)
	if duplicate {
		return ruleFailure(fmt.Sprintf("Duplicate value %s (first seen at %s)", display, firstPath), path)
	}
	return nil
}

// remember owns the per-scope first-seen state. It never replaces a first path.
func (r *UniqueRule) remember(scopeKey, key, path string) (string, bool) {
	if r.seen == nil {
		r.seen = map[string]map[string]string{}
	}
	bucket, exists := r.seen[scopeKey]
	if !exists {
		bucket = map[string]string{}
		r.seen[scopeKey] = bucket
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

func (r *UniqueRule) scopeKeyFor(t *selector.Target) (string, bool) {
	if len(r.scope.paths) == 0 {
		return globalScopeKey, true
	}
	bestIdx, bestLen := -1, -1
	for i, sp := range r.scope.paths {
		if selector.IsUnder(t.Path, sp) && len(sp) > bestLen {
			bestIdx, bestLen = i, len(sp)
		}
	}
	if bestIdx < 0 {
		return "", false
	}
	return r.scope.paths[bestIdx].String(), true
}

func comparableKey(value interface{}) (key string, displayValue string, supported bool) {
	switch v := value.(type) {
	case nil:
		return "null", "null", true
	case string:
		return "string:" + v, strconv.Quote(v), true
	case bool:
		display := strconv.FormatBool(v)
		return "bool:" + display, display, true
	case float64:
		display := strconv.FormatFloat(v, 'g', -1, 64)
		return "number:" + display, display, true
	default:
		return "", "", false
	}
}
