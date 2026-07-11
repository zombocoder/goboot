package config

import (
	"fmt"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// FlattenYAML parses a YAML document and flattens it into dotted, lowercase keys
// suitable for a MapSource. Nested mappings become dotted paths
// (server.read-timeout), and scalar values are rendered as strings. Sequences
// are joined with commas so that []string fields bind naturally.
func FlattenYAML(data []byte) (MapSource, error) {
	var root map[string]any
	if err := yaml.Unmarshal(data, &root); err != nil {
		return nil, fmt.Errorf("config: parsing YAML: %w", err)
	}
	out := MapSource{}
	flatten("", root, out)
	return out, nil
}

// flatten walks a decoded YAML value, writing scalar leaves into dst.
func flatten(prefix string, value any, dst MapSource) {
	switch v := value.(type) {
	case map[string]any:
		for key, child := range v {
			flatten(joinKey(prefix, strings.ToLower(key)), child, dst)
		}
	case []any:
		parts := make([]string, len(v))
		for i, e := range v {
			parts[i] = scalarString(e)
		}
		dst[prefix] = strings.Join(parts, ",")
	default:
		if prefix != "" {
			dst[prefix] = scalarString(v)
		}
	}
}

// joinKey joins a prefix and a key with a dot.
func joinKey(prefix, key string) string {
	if prefix == "" {
		return key
	}
	return prefix + "." + key
}

// scalarString renders a scalar YAML value as a string.
func scalarString(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case bool:
		return strconv.FormatBool(t)
	case int:
		return strconv.FormatInt(int64(t), 10)
	case int64:
		return strconv.FormatInt(t, 10)
	case float64:
		return strconv.FormatFloat(t, 'g', -1, 64)
	case nil:
		return ""
	default:
		return fmt.Sprintf("%v", t)
	}
}
