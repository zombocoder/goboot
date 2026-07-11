package runtime

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strconv"
)

// Binder populates a request struct from an HTTP request (§19.2). The default
// implementation uses reflection, which is acceptable here because binding is
// runtime data transformation, not dependency discovery (§19.2). A future
// version may generate type-specific binders (§19.3).
type Binder interface {
	Bind(ctx context.Context, r *http.Request, target any) error
}

// DefaultBinder binds struct fields from path, query, header, and cookie values
// (via struct tags) and decodes a JSON body into the remaining fields.
type DefaultBinder struct {
	// MaxBodyBytes bounds the JSON request body. Zero selects defaultMaxBody.
	MaxBodyBytes int64
}

const defaultMaxBody = 1 << 20 // 1 MiB

// errBadRequest is the status/code used for malformed requests.
func bindError(format string, args ...any) error {
	return Errorf(http.StatusBadRequest, "bad_request", format, args...)
}

// Bind implements Binder.
func (b DefaultBinder) Bind(_ context.Context, r *http.Request, target any) error {
	v := reflect.ValueOf(target)
	if v.Kind() != reflect.Pointer || v.IsNil() || v.Elem().Kind() != reflect.Struct {
		return errors.New("runtime: bind target must be a non-nil pointer to a struct")
	}

	if err := b.decodeBody(r, target); err != nil {
		return err
	}
	return bindFields(r, v.Elem())
}

// decodeBody decodes a JSON request body into target when one is present.
func (b DefaultBinder) decodeBody(r *http.Request, target any) error {
	if r.Body == nil || r.Method == http.MethodGet || r.Method == http.MethodDelete {
		return nil
	}
	limit := b.MaxBodyBytes
	if limit == 0 {
		limit = defaultMaxBody
	}
	dec := json.NewDecoder(io.LimitReader(r.Body, limit))
	if err := dec.Decode(target); err != nil {
		if errors.Is(err, io.EOF) {
			return nil // empty body is not an error
		}
		return bindError("invalid JSON body: %v", err)
	}
	return nil
}

// bindFields sets fields tagged path/query/header/cookie from the request.
func bindFields(r *http.Request, s reflect.Value) error {
	t := s.Type()
	for i := 0; i < t.NumField(); i++ {
		field := t.Field(i)
		if !field.IsExported() {
			continue
		}
		fv := s.Field(i)
		if name, ok := field.Tag.Lookup("path"); ok {
			if err := setValue(fv, []string{r.PathValue(name)}, field.Name); err != nil {
				return err
			}
		}
		if name, ok := field.Tag.Lookup("query"); ok {
			if vals, present := r.URL.Query()[name]; present {
				if err := setValue(fv, vals, field.Name); err != nil {
					return err
				}
			}
		}
		if name, ok := field.Tag.Lookup("header"); ok {
			if v := r.Header.Get(name); v != "" {
				if err := setValue(fv, []string{v}, field.Name); err != nil {
					return err
				}
			}
		}
		if name, ok := field.Tag.Lookup("cookie"); ok {
			if c, err := r.Cookie(name); err == nil {
				if err := setValue(fv, []string{c.Value}, field.Name); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// setValue converts string request values into a struct field.
func setValue(fv reflect.Value, values []string, fieldName string) error {
	if len(values) == 0 {
		return nil
	}
	if fv.Kind() == reflect.Slice && fv.Type().Elem().Kind() == reflect.String {
		fv.Set(reflect.ValueOf(append([]string(nil), values...)))
		return nil
	}
	return setScalar(fv, values[0], fieldName)
}

// setScalar converts a single string into a scalar field.
func setScalar(fv reflect.Value, raw, fieldName string) error {
	if raw == "" {
		return nil
	}
	switch fv.Kind() {
	case reflect.String:
		fv.SetString(raw)
	case reflect.Bool:
		v, err := strconv.ParseBool(raw)
		if err != nil {
			return bindError("field %s: invalid boolean %q", fieldName, raw)
		}
		fv.SetBool(v)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		v, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return bindError("field %s: invalid integer %q", fieldName, raw)
		}
		fv.SetInt(v)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		v, err := strconv.ParseUint(raw, 10, 64)
		if err != nil {
			return bindError("field %s: invalid unsigned integer %q", fieldName, raw)
		}
		fv.SetUint(v)
	case reflect.Float32, reflect.Float64:
		v, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			return bindError("field %s: invalid number %q", fieldName, raw)
		}
		fv.SetFloat(v)
	default:
		return bindError("field %s: unsupported bind target %s", fieldName, fv.Kind())
	}
	return nil
}
