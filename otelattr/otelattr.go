// Package otelattr edits the attributes an instrumentation produces.
//
// Instrumentation libraries decide their own attributes, and most expose no
// hook to change them: otelhttp puts the full request URL on every client
// span, which is where credentials in a query string and high-cardinality
// paths end up. An [Editor] applied to a wrapped [go.opentelemetry.io/otel/trace.TracerProvider]
// runs on every span the instrumentation creates, so the call site decides what
// leaves the process without patching the instrumentation.
//
// Metrics need no wrapper: go.opentelemetry.io/otel/sdk/metric.Stream carries
// an AttributeFilter, so attributes are dropped by a view at provider setup.
package otelattr

import (
	"net/url"

	"go.opentelemetry.io/otel/attribute"
)

// Redacted replaces a value an [Editor] hides.
const Redacted = "[redacted]"

// Editor rewrites a set of attributes. It must not modify or retain attrs:
// the slice can belong to the caller of the instrumented code.
//
// A nil Editor means no editing, and wrapping with one is a no-op.
type Editor func(attrs []attribute.KeyValue) []attribute.KeyValue

func (e Editor) apply(attrs []attribute.KeyValue) []attribute.KeyValue {
	if e == nil {
		return attrs
	}
	return e(attrs)
}

// Join returns an Editor applying each in order. Nil entries are skipped, and
// a Join of nothing is nil.
func Join(editors ...Editor) Editor {
	set := make([]Editor, 0, len(editors))
	for _, e := range editors {
		if e != nil {
			set = append(set, e)
		}
	}
	switch len(set) {
	case 0:
		return nil
	case 1:
		return set[0]
	}
	return func(attrs []attribute.KeyValue) []attribute.KeyValue {
		for _, e := range set {
			attrs = e(attrs)
		}
		return attrs
	}
}

// Drop removes the named attributes.
func Drop(keys ...attribute.Key) Editor {
	drop := keySet(keys)
	if len(drop) == 0 {
		return nil
	}
	return func(attrs []attribute.KeyValue) []attribute.KeyValue {
		return filter(attrs, func(kv attribute.KeyValue) bool {
			_, ok := drop[kv.Key]
			return !ok
		})
	}
}

// Keep removes every attribute but the named ones.
func Keep(keys ...attribute.Key) Editor {
	keep := keySet(keys)
	return func(attrs []attribute.KeyValue) []attribute.KeyValue {
		return filter(attrs, func(kv attribute.KeyValue) bool {
			_, ok := keep[kv.Key]
			return ok
		})
	}
}

// Add appends attributes. It does not deduplicate: the OpenTelemetry SDK keeps
// the last value for a repeated key, so an Add of a key already present wins.
func Add(kvs ...attribute.KeyValue) Editor {
	if len(kvs) == 0 {
		return nil
	}
	return func(attrs []attribute.KeyValue) []attribute.KeyValue {
		out := make([]attribute.KeyValue, 0, len(attrs)+len(kvs))
		return append(append(out, attrs...), kvs...)
	}
}

// Redact replaces the value of the named attributes with [Redacted], keeping
// the attribute itself so a query can still tell it was set.
func Redact(keys ...attribute.Key) Editor {
	return RedactFunc(keys, func(attribute.Value) attribute.Value {
		return attribute.StringValue(Redacted)
	})
}

// RedactFunc replaces the value of the named attributes with fn's result.
func RedactFunc(keys []attribute.Key, fn func(attribute.Value) attribute.Value) Editor {
	redact := keySet(keys)
	if len(redact) == 0 || fn == nil {
		return nil
	}
	return func(attrs []attribute.KeyValue) []attribute.KeyValue {
		var out []attribute.KeyValue
		for i, kv := range attrs {
			if _, ok := redact[kv.Key]; !ok {
				continue
			}
			if out == nil {
				out = make([]attribute.KeyValue, len(attrs))
				copy(out, attrs)
			}
			out[i].Value = fn(kv.Value)
		}
		if out == nil {
			return attrs
		}
		return out
	}
}

// URLKeys are the attributes carrying a full URL: url.full is the current
// convention, http.url the one it replaced. Instrumentation emits both while
// the semantic conventions migrate.
var URLKeys = []attribute.Key{"url.full", "http.url"}

// RedactURL rewrites URL-valued attributes to scheme://host, dropping the
// userinfo, path, query and fragment. With no keys it edits [URLKeys].
//
// This is the http client case: the destination stays queryable while
// credentials in a query string, presigned signatures, and per-object paths
// never reach the trace backend.
func RedactURL(keys ...attribute.Key) Editor {
	if len(keys) == 0 {
		keys = URLKeys
	}
	return RedactFunc(keys, func(v attribute.Value) attribute.Value {
		if v.Type() != attribute.STRING {
			return v
		}
		u, err := url.Parse(v.AsString())
		if err != nil {
			return attribute.StringValue(Redacted)
		}
		return attribute.StringValue((&url.URL{Scheme: u.Scheme, Host: u.Host}).String())
	})
}

func keySet(keys []attribute.Key) map[attribute.Key]struct{} {
	set := make(map[attribute.Key]struct{}, len(keys))
	for _, k := range keys {
		set[k] = struct{}{}
	}
	return set
}

func filter(attrs []attribute.KeyValue, keep func(attribute.KeyValue) bool) []attribute.KeyValue {
	out := make([]attribute.KeyValue, 0, len(attrs))
	for _, kv := range attrs {
		if keep(kv) {
			out = append(out, kv)
		}
	}
	return out
}
