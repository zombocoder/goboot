package runtime

import (
	"net/http"
	"strings"
)

// NegotiateContent enforces a route's @Consumes / @Produces media-type
// constraints (§19). It is called before binding so an unacceptable request
// fails fast:
//
//   - consumes: when non-empty and the request carries a Content-Type, the
//     media type must be listed, else a 415 error is returned.
//   - produces: when non-empty, the request's Accept header must match one of
//     the offered types (an empty or */* Accept matches anything), else 406.
//
// Empty constraint lists disable the respective check. A nil return means the
// request is acceptable.
func NegotiateContent(r *http.Request, consumes, produces []string) error {
	if len(consumes) > 0 {
		if ct := mediaType(r.Header.Get("Content-Type")); ct != "" && !matchesMedia(consumes, ct) {
			return Errorf(http.StatusUnsupportedMediaType, "unsupported_media_type",
				"content type %q is not supported", ct)
		}
	}
	if len(produces) > 0 && !acceptsAny(r.Header.Get("Accept"), produces) {
		return NewError(http.StatusNotAcceptable, "not_acceptable",
			"no acceptable representation for the Accept header")
	}
	return nil
}

// mediaType extracts the lowercased media type from a header value, dropping any
// parameters (e.g. "application/json; charset=utf-8" → "application/json").
func mediaType(header string) string {
	if i := strings.IndexByte(header, ';'); i >= 0 {
		header = header[:i]
	}
	return strings.ToLower(strings.TrimSpace(header))
}

// matchesMedia reports whether ct is covered by any entry in list. A list entry
// of "*/*" matches anything and "type/*" matches that type.
func matchesMedia(list []string, ct string) bool {
	for _, item := range list {
		if mediaRangeMatches(strings.ToLower(strings.TrimSpace(item)), ct) {
			return true
		}
	}
	return false
}

// acceptsAny reports whether an Accept header admits any of the offered types.
// An empty Accept accepts anything.
func acceptsAny(accept string, produces []string) bool {
	if strings.TrimSpace(accept) == "" {
		return true
	}
	for _, part := range strings.Split(accept, ",") {
		rng := mediaType(part) // also strips the q-value parameter
		if rng == "" {
			continue
		}
		for _, p := range produces {
			if mediaRangeMatches(rng, strings.ToLower(strings.TrimSpace(p))) {
				return true
			}
		}
	}
	return false
}

// mediaRangeMatches reports whether a media range (which may be */* or type/*)
// covers a concrete media type.
func mediaRangeMatches(rng, concrete string) bool {
	switch {
	case rng == "*/*":
		return true
	case strings.HasSuffix(rng, "/*"):
		return strings.HasPrefix(concrete, strings.TrimSuffix(rng, "*"))
	default:
		return rng == concrete
	}
}
