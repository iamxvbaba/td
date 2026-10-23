package semantic

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/gotd/tl"
)

// ManifestLegacyMethod promotes one superseded official RPC constructor to a
// distinct canonical semantic method. Definition is the exact official
// historical declaration; Name is the canonical method that owns it.
//
// Same-name binding would make the historical constructor a profile variant of
// its replacement, which is wrong when the replacement changed the meaning of
// a field (for example an SRP proof replaced by a temporary password). The
// promotion keeps the historical wire, fields and result unchanged under a
// reviewed name, so no field is transcoded or discarded.
type ManifestLegacyMethod struct {
	Name       string `json:"name"`
	Definition string `json:"definition"`
}

type legacyMethod struct {
	name       string
	namespace  []string
	local      string
	source     string
	definition tl.SchemaDefinition
}

func parseLegacyMethods(entries []ManifestLegacyMethod) ([]legacyMethod, error) {
	result := make([]legacyMethod, 0, len(entries))
	names := make(map[string]struct{}, len(entries))
	wires := make(map[uint32]string, len(entries))
	for i, entry := range entries {
		prefix := fmt.Sprintf("legacy_methods[%d]", i)
		name := strings.TrimSpace(entry.Name)
		if name == "" || name != entry.Name {
			return nil, fmt.Errorf("semantic: %s name %q is empty or padded", prefix, entry.Name)
		}
		if _, duplicate := names[name]; duplicate {
			return nil, fmt.Errorf("semantic: %s repeats name %q", prefix, name)
		}
		names[name] = struct{}{}
		data := []byte("---functions---\n" + entry.Definition + "\n")
		if err := validateExplicitIDs(data, prefix+" definition"); err != nil {
			return nil, err
		}
		parsed, err := tl.Parse(strings.NewReader(string(data)))
		if err != nil {
			return nil, fmt.Errorf("semantic: %s parse definition: %w", prefix, err)
		}
		if len(parsed.Definitions) != 1 || parsed.Definitions[0].Category != tl.CategoryFunction {
			return nil, fmt.Errorf("semantic: %s definition must be exactly one RPC function", prefix)
		}
		definition := parsed.Definitions[0]
		if len(definition.Definition.GenericParams) != 0 {
			return nil, fmt.Errorf("semantic: %s definition has generic parameters", prefix)
		}
		source := qualifyDefinition(definition.Definition)
		namespace, local := splitQualifiedName(name)
		if !validTLIdentifier(local) || !equalNamespace(namespace, definition.Definition.Namespace) {
			return nil, fmt.Errorf("semantic: %s name %q must be an identifier in the namespace of %q", prefix, name, source)
		}
		if name == source {
			return nil, fmt.Errorf("semantic: %s name %q does not differ from its official definition", prefix, name)
		}
		if previous, duplicate := wires[definition.Definition.ID]; duplicate {
			return nil, fmt.Errorf("semantic: %s wire ID %#08x is already promoted to %q", prefix, definition.Definition.ID, previous)
		}
		wires[definition.Definition.ID] = name
		result = append(result, legacyMethod{
			name:       name,
			namespace:  namespace,
			local:      local,
			source:     source,
			definition: definition,
		})
	}
	return result, nil
}

// applyLegacyMethods renames each reviewed historical definition in an
// official profile and adds the promoted method to the canonical profile. The
// canonical profile must define the superseding method under the official name
// and must not define the historical wire ID itself; either violation makes
// the promotion stale. It returns which methods matched a historical profile.
func applyLegacyMethods(schema *tl.Schema, layer int, canonical bool, methods []legacyMethod) ([]bool, error) {
	matched := make([]bool, len(methods))
	for i, method := range methods {
		wireIndex, sourceIndex := -1, -1
		for index, schemaDef := range schema.Definitions {
			name := qualifyDefinition(schemaDef.Definition)
			if schemaDef.Definition.ID == method.definition.Definition.ID {
				wireIndex = index
			}
			if schemaDef.Category != tl.CategoryFunction {
				continue
			}
			if name == method.name {
				return nil, fmt.Errorf("semantic: layer %d already defines legacy method %q", layer, method.name)
			}
			if name == method.source {
				sourceIndex = index
			}
		}
		if canonical {
			if wireIndex >= 0 {
				return nil, fmt.Errorf("semantic: canonical layer %d still defines legacy wire ID %#08x for %q",
					layer, method.definition.Definition.ID, method.name)
			}
			if sourceIndex < 0 {
				return nil, fmt.Errorf("semantic: canonical layer %d has no superseding method %q for %q",
					layer, method.source, method.name)
			}
			schema.Definitions = append(schema.Definitions, method.promoted())
			continue
		}
		if wireIndex < 0 {
			continue
		}
		official := schema.Definitions[wireIndex]
		if official.Category != tl.CategoryFunction || !reflect.DeepEqual(official.Definition, method.definition.Definition) {
			return nil, fmt.Errorf("semantic: layer %d wire ID %#08x differs from reviewed legacy definition %q",
				layer, method.definition.Definition.ID, method.name)
		}
		official.Definition.Namespace = append([]string(nil), method.namespace...)
		official.Definition.Name = method.local
		schema.Definitions[wireIndex] = official
		matched[i] = true
	}
	return matched, nil
}

func (m legacyMethod) promoted() tl.SchemaDefinition {
	result := cloneSchemaDefinition(m.definition)
	result.Definition.Namespace = append([]string(nil), m.namespace...)
	result.Definition.Name = m.local
	return result
}

func splitQualifiedName(name string) ([]string, string) {
	index := strings.LastIndexByte(name, '.')
	if index < 0 {
		return nil, name
	}
	return strings.Split(name[:index], "."), name[index+1:]
}

func equalNamespace(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func validTLIdentifier(value string) bool {
	if value == "" {
		return false
	}
	for i, r := range value {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z':
		case i > 0 && (r >= '0' && r <= '9' || r == '_'):
		default:
			return false
		}
	}
	return true
}
