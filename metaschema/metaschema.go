// Package metaschema selects and loads the OpenRPC meta-schema matching a
// document's declared specification version.
package metaschema

import (
	"errors"
	"fmt"
	"strings"
	"sync"

	v1_3 "github.com/open-rpc/spec-types/generated/packages/go/v1_3"
	v1_4 "github.com/open-rpc/spec-types/generated/packages/go/v1_4"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

const (
	Supported              = "1.0.x-1.3.x, 1.4.x"
	schemaURL              = "https://meta.open-rpc.org/"
	draft7URL              = "http://json-schema.org/draft-07/schema#"
	metaJSONSchemaURL      = "https://meta.json-schema.tools"
	metaJSONSchemaSlashURL = "https://meta.json-schema.tools/"
	metaJSONSchemaRefURL   = "https://meta.json-schema.tools/#/definitions/JSONSchemaObject/properties/$ref"
	draft7RefURL           = "http://json-schema.org/draft-07/schema#/properties/$ref"
)

var ErrUnsupportedVersion = errors.New("unsupported OpenRPC version")

type MetaSchema struct {
	VersionFamily string
	root          map[string]any
}

// Root returns the decoded root schema object.
func (m *MetaSchema) Root() map[string]any {
	return m.root
}

// Resolve follows a local "#/definitions/<name>" reference. External
// references are outside the OpenRPC selector's indexing boundary.
func (m *MetaSchema) Resolve(ref string) map[string]any {
	name, local := strings.CutPrefix(ref, "#/definitions/")
	if !local {
		return nil
	}
	definitions, _ := m.root["definitions"].(map[string]any)
	target, _ := definitions[name].(map[string]any)
	return target
}

// Compile compiles this OpenRPC schema entirely from embedded resources.
func (m *MetaSchema) Compile() (*jsonschema.Schema, error) {
	return m.compileWith(jsonschema.NewCompiler())
}

func (m *MetaSchema) compileWith(compiler *jsonschema.Compiler) (*jsonschema.Schema, error) {
	if err := compiler.AddResource(schemaURL, m.root); err != nil {
		return nil, err
	}
	return compiler.Compile(schemaURL)
}

var (
	loadV13 = sync.OnceValues(func() (*MetaSchema, error) {
		return parse("1.0.x-1.3.x", v1_3.RawOpenrpcDocument)
	})
	loadV14 = sync.OnceValues(func() (*MetaSchema, error) {
		return parse("1.4.x", v1_4.RawOpenrpcDocument)
	})
)

// For selects the meta-schema matching doc's root openrpc field. A missing or
// non-string field uses Latest so validation can report the malformed field.
func For(doc any) (*MetaSchema, error) {
	version, err := Version(doc)
	if err != nil {
		return Latest()
	}
	return ForVersion(version)
}

// ForVersion selects a schema family explicitly. Add a case here whenever
// spec-types publishes a new OpenRPC specification family.
func ForVersion(version string) (*MetaSchema, error) {
	switch {
	case version == "1.4" || strings.HasPrefix(version, "1.4."):
		return loadV14()
	case strings.HasPrefix(version, "1.0."),
		strings.HasPrefix(version, "1.1."),
		strings.HasPrefix(version, "1.2."),
		strings.HasPrefix(version, "1.3."):
		return loadV13()
	default:
		return nil, fmt.Errorf("%w %q (supported: %s)", ErrUnsupportedVersion, version, Supported)
	}
}

func Version(doc any) (string, error) {
	root, ok := doc.(map[string]any)
	if !ok {
		return "", fmt.Errorf("document is not an OpenRPC document")
	}
	version, ok := root["openrpc"].(string)
	if !ok {
		return "", fmt.Errorf("document has no openrpc version")
	}
	return version, nil
}

// Latest returns the newest meta-schema known to this linter.
func Latest() (*MetaSchema, error) {
	return loadV14()
}

func parse(version, raw string) (*MetaSchema, error) {
	root, err := jsonschema.UnmarshalJSON(strings.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("parsing embedded OpenRPC %s meta-schema: %w", version, err)
	}
	rootMap, ok := root.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("parsing embedded OpenRPC %s meta-schema: root is not an object", version)
	}
	rewriteJSONSchemaRefs(rootMap)
	return &MetaSchema{VersionFamily: version, root: rootMap}, nil
}

// rewriteJSONSchemaRefs replaces the external meta.json-schema.tools aliases
// emitted by spec-types with the equivalent draft-07 resources embedded in
// jsonschema/v6. Validation therefore never needs network access.
func rewriteJSONSchemaRefs(value any) {
	switch node := value.(type) {
	case map[string]any:
		for _, key := range []string{"$schema", "$ref"} {
			ref, ok := node[key].(string)
			if !ok {
				continue
			}
			switch ref {
			case metaJSONSchemaURL, metaJSONSchemaSlashURL:
				node[key] = draft7URL
			case metaJSONSchemaRefURL:
				node[key] = draft7RefURL
			}
		}
		for _, child := range node {
			rewriteJSONSchemaRefs(child)
		}
	case []any:
		for _, child := range node {
			rewriteJSONSchemaRefs(child)
		}
	}
}
