package selector

import (
	"strings"

	"github.com/open-rpc/openrpc-linter/metaschema"
	"github.com/theory/jsonpath/spec"
)

// Candidate records a document object that the meta-schema recognizes as
// belonging to a named OpenRPC type. Fields is the set of properties that
// the matched schema declares
type Candidate struct {
	Path        spec.NormalizedPath
	Node        any
	SchemaTitle string
	Fields      map[string]struct{}
}

// Index answers two questions used by the selector:
//
//	ByField["description"]  -> every Candidate whose schema declares description
//	ByPath[path.String()]   -> the Candidate at that exact document position
type Index struct {
	ByField map[string][]*Candidate
	ByPath  map[string]*Candidate
}

// Build walks doc using meta to classify each object and collect candidates.
// Assumes $refs are already resolved.
func Build(doc any, meta *metaschema.MetaSchema) *Index {
	idx := &Index{
		ByField: make(map[string][]*Candidate),
		ByPath:  make(map[string]*Candidate),
	}
	b := &builder{meta: meta, idx: idx}
	b.walk(doc, meta.Root(), spec.NormalizedPath{})
	return idx
}

type builder struct {
	meta *metaschema.MetaSchema
	idx  *Index
}

// walk recurses over (node, schema) in lockstep. node is the document
// value; schema is the meta-schema fragment that describes it.
func (b *builder) walk(node any, schema map[string]any, path spec.NormalizedPath) {
	if schema == nil {
		return
	}

	schema = b.deref(schema)
	if schema == nil {
		// External ref (JSON Schema boundary)
		return
	}

	if branch := b.chooseOneOf(schema); branch != nil {
		schema = branch
	}

	switch v := node.(type) {
	case map[string]any:
		b.indexObject(v, schema, path)
	case []any:
		b.indexArray(v, schema, path)
	}
}

func (b *builder) indexObject(obj map[string]any, schema map[string]any, path spec.NormalizedPath) {
	if candidate := objectCandidate(obj, schema, path); candidate != nil {
		b.idx.ByPath[candidate.Path.String()] = candidate
		for field := range candidate.Fields {
			b.idx.ByField[field] = append(b.idx.ByField[field], candidate)
		}
	}

	// Recurse into every actual document key whose schema position we can
	// determine. Unknown keys (extensions, "x-*" via patternProperties, or
	// genuinely unrecognized) are skipped — they don't break indexing,
	// they just don't get classified.
	for key, child := range obj {
		childSchema := b.schemaForKey(schema, key)
		if childSchema == nil {
			continue
		}
		b.walk(child, childSchema, append(path, spec.Name(key)))
	}
}

// objectCandidate constructs the field description for a recognized object.
// A schema with no named properties contributes no candidate.
func objectCandidate(obj map[string]any, schema map[string]any, path spec.NormalizedPath) *Candidate {
	properties := schemaValue[map[string]any](schema["properties"])
	if len(properties) == 0 {
		return nil
	}
	fields := make(map[string]struct{}, len(properties))
	for field := range properties {
		fields[field] = struct{}{}
	}
	return &Candidate{
		Path:        append(spec.NormalizedPath{}, path...),
		Node:        obj,
		SchemaTitle: schemaValue[string](schema["title"]),
		Fields:      fields,
	}
}

func (b *builder) indexArray(arr []any, schema map[string]any, path spec.NormalizedPath) {
	itemSchema := schemaValue[map[string]any](schema["items"])
	if itemSchema == nil {
		return
	}
	for i, item := range arr {
		b.walk(item, itemSchema, append(path, spec.Index(i)))
	}
}

// schemaForKey finds the schema fragment governing a specific child key.
// properties wins; patternProperties is consulted next; additionalProperties
// is consulted last. Returns nil for keys the schema doesn't know about
// (which is normal for "x-" extensions when additionalProperties is false).
func (b *builder) schemaForKey(schema map[string]any, key string) map[string]any {
	properties := schemaValue[map[string]any](schema["properties"])
	if direct, exists := properties[key]; exists {
		return schemaValue[map[string]any](direct)
	}
	for pattern, candidate := range schemaValue[map[string]any](schema["patternProperties"]) {
		if matchesPattern(pattern, key) {
			return schemaValue[map[string]any](candidate)
		}
	}
	return schemaValue[map[string]any](schema["additionalProperties"])
}

// deref returns the resolved local definition if schema is a {"$ref": "#/definitions/..."}
// object. For external refs it returns nil to signal "stop here". For
// schemas with no $ref it returns schema unchanged.
func (b *builder) deref(schema map[string]any) map[string]any {
	ref, ok := schema["$ref"].(string)
	if !ok {
		return schema
	}
	return b.meta.Resolve(ref)
}

// chooseOneOf picks the concrete branch from a v1.4-style oneOf. Every
// oneOf in the meta-schema is [concreteOpenRPCType, referenceObject];
func (b *builder) chooseOneOf(schema map[string]any) map[string]any {
	for _, branch := range schemaValue[[]any](schema["oneOf"]) {
		resolved := b.deref(schemaValue[map[string]any](branch))
		if resolved == nil || schemaValue[string](resolved["title"]) == "referenceObject" {
			continue
		}
		return resolved
	}
	return nil
}

// matchesPattern recognizes the two property patterns in the embedded schema.
// The unanchored component pattern needs at least one character from 0 to z.
func matchesPattern(pattern, key string) bool {
	switch pattern {
	case "^x-":
		return strings.HasPrefix(key, "x-")
	case "[0-z]+":
		return strings.IndexFunc(key, func(character rune) bool {
			return character >= '0' && character <= 'z'
		}) >= 0
	}
	return false
}

// schemaValue reads a typed schema value; absent or mismatched values use
// the type's zero value so callers can handle malformed schema fragments.
func schemaValue[T any](value any) T {
	result, _ := value.(T)
	return result
}
