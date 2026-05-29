package obs

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/ethanvc/evo/logjson"
	"github.com/stretchr/testify/require"
)

func ptr(s string) *string { return &s }

func ptrBytes(b []byte) *[]byte { return &b }

func TestPreferJson_MarshalJSON(t *testing.T) {
	tests := []struct {
		name string
		in   PreferJson
		want string
	}{
		{
			name: "valid object string",
			in:   PreferJSON(`{"a":1}`),
			want: `{"a":1}`,
		},
		{
			name: "valid array bytes",
			in:   PreferJSON([]byte("[1,2]")),
			want: `[1,2]`,
		},
		{
			name: "valid json string literal",
			in:   PreferJSON(`"hello"`),
			want: `"hello"`,
		},
		{
			name: "valid json number",
			in:   PreferJSON(`123`),
			want: `123`,
		},
		{
			name: "valid json bool",
			in:   PreferJSON(`true`),
			want: `true`,
		},
		{
			name: "valid json null",
			in:   PreferJSON(`null`),
			want: `null`,
		},
		{
			name: "invalid text falls back to string",
			in:   PreferJSON(`not-json`),
			want: `"not-json"`,
		},
		{
			name: "valid object string pointer",
			in:   PreferJSON(ptr(`{"a":1}`)),
			want: `{"a":1}`,
		},
		{
			name: "valid array bytes pointer",
			in:   PreferJSON(ptrBytes([]byte("[1,2]"))),
			want: `[1,2]`,
		},
		{
			name: "invalid string pointer falls back to string",
			in:   PreferJSON(ptr("not-json")),
			want: `"not-json"`,
		},
		{
			name: "nil string pointer uses generic marshal",
			in:   PreferJSON((*string)(nil)),
			want: `null`,
		},
		{
			name: "nil bytes pointer uses generic marshal",
			in:   PreferJSON((*[]byte)(nil)),
			want: `null`,
		},
		{
			name: "other type uses generic marshal",
			in:   PreferJSON(42),
			want: `42`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := logjson.Marshal(tt.in)
			require.NoError(t, err)
			require.Equal(t, tt.want, string(got))
		})
	}
}

func TestPreferJson_MarshalJSONTo(t *testing.T) {
	var buf bytes.Buffer
	enc := logjson.NewEncoderOf(&buf)
	require.NoError(t, enc.WriteToken(logjson.BeginObject))
	require.NoError(t, enc.WriteToken(logjson.TokenString("payload")))
	p := PreferJSON(`{"k":"v"}`)
	require.NoError(t, p.MarshalJSONTo(enc))
	require.NoError(t, enc.WriteToken(logjson.EndObject))
	require.Equal(t, `{"payload":{"k":"v"}}`, strings.TrimSuffix(buf.String(), "\n"))
}

func TestJsonHandler_PreferJson(t *testing.T) {
	var writer nopWriteCloser
	handler := NewJsonHandler(&writer)
	ctx, _ := WithObsContext(context.Background(), &ObsConfig{Handler: handler})

	Info(ctx, "Test", Any("payload", PreferJSON(`{"k":"v"}`)))
	require.Contains(t, writer.String(), `|Test|{"payload":{"k":"v"}}`)
}
