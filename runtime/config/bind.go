package config

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// durationType is time.Duration, handled specially so that duration strings such
// as "15s" bind correctly despite the underlying int64 kind.
var durationType = reflect.TypeOf(time.Duration(0))

// Bind populates target (a pointer to a struct) from src, reading each field
// under the given dotted prefix. Field keys come from the `config:"..."` tag or
// the lowercased field name; missing values fall back to the `default:"..."`
// tag; a field tagged `required:"true"` with neither a value nor a default is an
// error (§28.5). Nested structs are bound recursively under an extended prefix.
func Bind(prefix string, src Source, target any) error {
	v := reflect.ValueOf(target)
	if v.Kind() != reflect.Pointer || v.IsNil() || v.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("config: bind target must be a non-nil pointer to a struct")
	}
	return bindStruct(prefix, src, v.Elem())
}

func bindStruct(prefix string, src Source, s reflect.Value) error {
	t := s.Type()
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if !field.IsExported() {
			continue
		}
		key := field.Tag.Get("config")
		if key == "" {
			key = strings.ToLower(field.Name)
		}
		full := joinKey(prefix, key)
		fv := s.Field(i)

		// Recurse into nested structs (but not special-cased types like
		// time.Duration or time.Time).
		if fv.Kind() == reflect.Struct && fv.Type() != durationType && !isTime(fv.Type()) {
			if err := bindStruct(full, src, fv); err != nil {
				return err
			}
			continue
		}

		raw, ok := src.Get(full)
		if !ok {
			raw, ok = field.Tag.Lookup("default")
		}
		if !ok || raw == "" {
			if field.Tag.Get("required") == "true" && raw == "" {
				return fmt.Errorf("config: required property %q is not set", full)
			}
			continue
		}
		if err := setField(fv, raw); err != nil {
			return fmt.Errorf("config: property %q: %w", full, err)
		}
	}
	return nil
}

// setField parses a raw string into a struct field.
func setField(fv reflect.Value, raw string) error {
	if fv.Type() == durationType {
		d, err := time.ParseDuration(raw)
		if err != nil {
			return fmt.Errorf("invalid duration %q", raw)
		}
		fv.SetInt(int64(d))
		return nil
	}

	switch fv.Kind() {
	case reflect.String:
		fv.SetString(raw)
	case reflect.Bool:
		b, err := strconv.ParseBool(raw)
		if err != nil {
			return fmt.Errorf("invalid boolean %q", raw)
		}
		fv.SetBool(b)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		n, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid integer %q", raw)
		}
		fv.SetInt(n)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		n, err := strconv.ParseUint(raw, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid unsigned integer %q", raw)
		}
		fv.SetUint(n)
	case reflect.Float32, reflect.Float64:
		f, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return fmt.Errorf("invalid number %q", raw)
		}
		fv.SetFloat(f)
	case reflect.Slice:
		if fv.Type().Elem().Kind() != reflect.String {
			return fmt.Errorf("unsupported slice element %s", fv.Type().Elem().Kind())
		}
		fv.Set(reflect.ValueOf(splitList(raw)))
	default:
		return fmt.Errorf("unsupported config target %s", fv.Kind())
	}
	return nil
}

// splitList parses a comma-separated list into a trimmed, non-empty slice.
func splitList(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			out = append(out, trimmed)
		}
	}
	return out
}

// isTime reports whether t is time.Time.
func isTime(t reflect.Type) bool {
	return t.PkgPath() == "time" && t.Name() == "Time"
}
