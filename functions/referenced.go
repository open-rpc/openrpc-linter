package functions

import (
	"fmt"
	"strings"

	"github.com/open-rpc/openrpc-linter/selector"
	"github.com/open-rpc/openrpc-linter/types"
	"github.com/theory/jsonpath/spec"
)

type ReferencedRule struct {
	ready bool
	used  map[string]bool
}

func NewReferencedRule() *ReferencedRule {
	return &ReferencedRule{}
}

func (r *ReferencedRule) RunRule(_ interface{}, context types.RuleFunctionContext) []types.RuleFunctionResult {
	t := context.Target
	if t == nil {
		return nil
	}

	r.ensureUsed(context.Document)

	pointer, ok := targetPointer(t)
	if !ok {
		return nil
	}
	if r.used[pointer] {
		return nil
	}

	return []types.RuleFunctionResult{{
		Message: fmt.Sprintf("unused component %q", componentDisplayName(pointer)),
		Path:    resultPath(t.PathString()),
	}}
}

func (r *ReferencedRule) ensureUsed(document interface{}) {
	if r.ready {
		return
	}
	r.ready = true
	r.used = usedComponents(document)
}

func usedComponents(document interface{}) map[string]bool {
	used := map[string]bool{}

	collectRefs(document, "#", func(path string, ref string) {
		component := componentEntryPointer(ref)
		if component == "" || componentEntryPointer(path) == component {
			return
		}
		used[component] = true
	})

	return used
}

func collectRefs(value interface{}, path string, visit func(path string, ref string)) {
	switch v := value.(type) {
	case map[string]interface{}:
		if raw, ok := v["$ref"].(string); ok && strings.HasPrefix(raw, "#/") {
			visit(path, raw)
		}
		for key, child := range v {
			collectRefs(child, path+"/"+jsonPointerEscape(key), visit)
		}
	case []interface{}:
		for i, child := range v {
			collectRefs(child, fmt.Sprintf("%s/%d", path, i), visit)
		}
	}
}

func targetPointer(t *selector.Target) (string, bool) {
	if t == nil || len(t.Path) == 0 {
		return "", false
	}

	segments := make([]string, 0, len(t.Path))
	for _, part := range t.Path {
		switch p := part.(type) {
		case spec.Name:
			segments = append(segments, jsonPointerEscape(string(p)))
		case spec.Index:
			segments = append(segments, fmt.Sprintf("%d", int(p)))
		default:
			return "", false
		}
	}
	return "#/" + strings.Join(segments, "/"), true
}

func componentEntryPointer(ref string) string {
	if !strings.HasPrefix(ref, "#/components/") {
		return ""
	}

	parts := strings.Split(ref, "/")
	if len(parts) < 4 || parts[0] != "#" || parts[1] != "components" || parts[2] == "" || parts[3] == "" {
		return ""
	}
	return strings.Join(parts[:4], "/")
}

func componentDisplayName(pointer string) string {
	parts := strings.Split(pointer, "/")
	if len(parts) < 4 {
		return pointer
	}
	return jsonPointerUnescape(parts[2]) + "." + jsonPointerUnescape(parts[3])
}

func jsonPointerEscape(value string) string {
	value = strings.ReplaceAll(value, "~", "~0")
	return strings.ReplaceAll(value, "/", "~1")
}

func jsonPointerUnescape(value string) string {
	value = strings.ReplaceAll(value, "~1", "/")
	return strings.ReplaceAll(value, "~0", "~")
}
