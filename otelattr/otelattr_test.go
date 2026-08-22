package otelattr_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/otel/attribute"

	"github.com/go-faster/sdk/otelattr"
)

func TestEditors(t *testing.T) {
	attrs := []attribute.KeyValue{
		attribute.String("url.full", "https://user:pass@example.com:8443/blob/key?sig=deadbeef#frag"),
		attribute.String("server.address", "example.com"),
		attribute.Int("http.response.status_code", 200),
	}

	tests := []struct {
		name string
		edit otelattr.Editor
		want []attribute.KeyValue
	}{
		{
			name: "nil is identity",
			edit: nil,
			want: attrs,
		},
		{
			name: "drop",
			edit: otelattr.Drop("url.full"),
			want: attrs[1:],
		},
		{
			name: "keep",
			edit: otelattr.Keep("server.address"),
			want: attrs[1:2],
		},
		{
			name: "redact",
			edit: otelattr.Redact("url.full"),
			want: []attribute.KeyValue{
				attribute.String("url.full", otelattr.Redacted),
				attrs[1], attrs[2],
			},
		},
		{
			name: "redact url",
			edit: otelattr.RedactURL(),
			want: []attribute.KeyValue{
				attribute.String("url.full", "https://example.com:8443"),
				attrs[1], attrs[2],
			},
		},
		{
			name: "redact url leaves other keys alone",
			edit: otelattr.RedactURL("http.url"),
			want: attrs,
		},
		{
			name: "add",
			edit: otelattr.Add(attribute.String("peer.service", "gitlab")),
			want: append(append([]attribute.KeyValue{}, attrs...), attribute.String("peer.service", "gitlab")),
		},
		{
			name: "join applies in order",
			edit: otelattr.Join(nil, otelattr.RedactURL(), otelattr.Drop("http.response.status_code")),
			want: []attribute.KeyValue{
				attribute.String("url.full", "https://example.com:8443"),
				attrs[1],
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := append([]attribute.KeyValue{}, attrs...)

			var got []attribute.KeyValue
			if tt.edit == nil {
				got = in
			} else {
				got = tt.edit(in)
			}

			require.Equal(t, tt.want, got)
			require.Equal(t, attrs, in, "editor modified the caller's slice")
		})
	}
}

func TestRedactURLUnparseable(t *testing.T) {
	got := otelattr.RedactURL()([]attribute.KeyValue{
		attribute.String("url.full", "://not a url"),
		attribute.Int("url.full.but.int", 1),
	})
	require.Equal(t, otelattr.Redacted, got[0].Value.AsString())
}

func TestJoinEmpty(t *testing.T) {
	require.Nil(t, otelattr.Join())
	require.Nil(t, otelattr.Join(nil, nil))
	require.Nil(t, otelattr.Drop())
	require.Nil(t, otelattr.Add())
}
