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
	seen       map[string]map[string]string
	scopePaths []spec.NormalizedPath
	scopeReady bool
	scopeErr   error
}

func NewUniqueRule() *UniqueRule {
	return &UniqueRule{seen: map[string]map[string]string{}}
}

func (r *UniqueRule) RunRule(value interface{}, context types.RuleFunctionContext) []types.RuleFunctionResult {
	t := context.Target
	if t == nil {
		return nil
	}

	if err := r.ensureScopes(context); err != nil {
		return []types.RuleFunctionResult{{
			Message: "unique scope option invalid: " + err.Error(),
			Path:    resultPath(t.PathString()),
		}}
	}

	scopeKey, inScope := r.scopeKeyFor(t)
	if !inScope {
		return nil
	}

	ignoreMissing, err := uniqueIgnoreMissing(context)
	if err != nil {
		return []types.RuleFunctionResult{{
			Message: err.Error(),
			Path:    resultPath(t.PathString()),
		}}
	}

	if t.Field != "" && !t.Exists {
		if ignoreMissing {
			return nil
		}
		value = nil
	}

	key, display, ok := comparableKey(value)
	if !ok {
		return []types.RuleFunctionResult{{
			Message: "unique value must be primitive",
			Path:    resultPath(t.PathString()),
		}}
	}

	if r.seen == nil {
		r.seen = map[string]map[string]string{}
	}
	bucket, exists := r.seen[scopeKey]
	if !exists {
		bucket = map[string]string{}
		r.seen[scopeKey] = bucket
	}
	if firstPath, dup := bucket[key]; dup {
		return []types.RuleFunctionResult{{
			Message: fmt.Sprintf("Duplicate value %s (first seen at %s)", display, firstPath),
			Path:    resultPath(t.PathString()),
		}}
	}
	bucket[key] = t.PathString()
	return nil
}

func uniqueIgnoreMissing(ctx types.RuleFunctionContext) (bool, error) {
	if ctx.Rule == nil || ctx.Rule.Then == nil {
		return true, nil
	}

	raw, exists := ctx.Rule.Then.FunctionOptions["ignoreMissing"]
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
	if r.scopeReady {
		return r.scopeErr
	}
	r.scopeReady = true

	if ctx.Rule == nil || ctx.Rule.Then == nil || ctx.Rule.Then.FunctionOptions == nil {
		return nil
	}
	raw, _ := ctx.Rule.Then.FunctionOptions["scope"].(string)
	if raw == "" {
		return nil
	}

	p, err := jsonpath.Parse(raw)
	if err != nil {
		r.scopeErr = err
		return err
	}

	doc := ctx.Document
	if ctx.ResolvedDocument != nil {
		doc = ctx.ResolvedDocument
	}
	for _, n := range p.SelectLocated(doc) {
		r.scopePaths = append(r.scopePaths, n.Path)
	}
	return nil
}

func (r *UniqueRule) scopeKeyFor(t *selector.Target) (string, bool) {
	if len(r.scopePaths) == 0 {
		return globalScopeKey, true
	}
	bestIdx, bestLen := -1, -1
	for i, sp := range r.scopePaths {
		if selector.IsUnder(t.Path, sp) && len(sp) > bestLen {
			bestIdx, bestLen = i, len(sp)
		}
	}
	if bestIdx < 0 {
		return "", false
	}
	return r.scopePaths[bestIdx].String(), true
}

func comparableKey(value interface{}) (key string, displayValue string, supported bool) {
	switch v := value.(type) {
	case nil:
		return "null", "null", true
	case string:
		return "string:" + v, strconv.Quote(v), true
	case bool:
		if v {
			return "bool:true", "true", true
		}
		return "bool:false", "false", true
	case float64:
		return fmt.Sprintf("number:%g", v), fmt.Sprintf("%g", v), true
	default:
		return "", "", false
	}
}
