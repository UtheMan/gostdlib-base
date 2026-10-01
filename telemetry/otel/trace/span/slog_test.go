package span

import (
	"log/slog"
	"math"
	"testing"
	"time"

	"github.com/kylelemons/godebug/pretty"
	"go.opentelemetry.io/otel/attribute"
)

type testLogValuer struct {
	v slog.Value
}

func (t testLogValuer) LogValue() slog.Value {
	return t.v
}

func TestAppendSlogAttrs(t *testing.T) {
	t.Parallel()

	testTime := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)

	tests := []struct {
		name   string
		dst    []attribute.KeyValue
		prefix string
		attrs  []slog.Attr
		want   []attribute.KeyValue
	}{
		{
			name:   "Success: Bool",
			prefix: "ns.",
			attrs:  []slog.Attr{slog.Bool("enabled", true)},
			want:   []attribute.KeyValue{attribute.Bool("ns.enabled", true)},
		},
		{
			name:   "Success: Int64",
			prefix: "ns.",
			attrs:  []slog.Attr{slog.Int64("count", 42)},
			want:   []attribute.KeyValue{attribute.Int64("ns.count", 42)},
		},
		{
			name:   "Success: Uint64 within int64 range",
			prefix: "ns.",
			attrs:  []slog.Attr{slog.Uint64("count", 100)},
			want:   []attribute.KeyValue{attribute.Int64("ns.count", 100)},
		},
		{
			name:   "Success: Uint64 equal to MaxInt64 is an int64",
			prefix: "ns.",
			attrs:  []slog.Attr{slog.Uint64("count", math.MaxInt64)},
			want:   []attribute.KeyValue{attribute.Int64("ns.count", math.MaxInt64)},
		},
		{
			name:   "Success: Uint64 exceeding int64 range is a string",
			prefix: "ns.",
			attrs:  []slog.Attr{slog.Uint64("big", math.MaxUint64)},
			want:   []attribute.KeyValue{attribute.String("ns.big", "18446744073709551615")},
		},
		{
			name:   "Success: Float64",
			prefix: "ns.",
			attrs:  []slog.Attr{slog.Float64("rate", 3.14)},
			want:   []attribute.KeyValue{attribute.Float64("ns.rate", 3.14)},
		},
		{
			name:   "Success: String",
			prefix: "ns.",
			attrs:  []slog.Attr{slog.String("name", "test")},
			want:   []attribute.KeyValue{attribute.String("ns.name", "test")},
		},
		{
			name:   "Success: Duration adds _ns suffix",
			prefix: "ns.",
			attrs:  []slog.Attr{slog.Duration("elapsed", 5*time.Second)},
			want:   []attribute.KeyValue{attribute.Int64("ns.elapsed_ns", int64(5*time.Second))},
		},
		{
			name:   "Success: Duration key already has _ns suffix",
			prefix: "ns.",
			attrs:  []slog.Attr{slog.Duration("elapsed_ns", 5*time.Second)},
			want:   []attribute.KeyValue{attribute.Int64("ns.elapsed_ns", int64(5*time.Second))},
		},
		{
			name:   "Success: Duration key ending in ns without underscore gets suffix",
			prefix: "ns.",
			attrs:  []slog.Attr{slog.Duration("runs", time.Second)},
			want:   []attribute.KeyValue{attribute.Int64("ns.runs_ns", int64(time.Second))},
		},
		{
			name:   "Success: Time adds _unix_ns suffix",
			prefix: "ns.",
			attrs:  []slog.Attr{slog.Time("created", testTime)},
			want:   []attribute.KeyValue{attribute.Int64("ns.created_unix_ns", testTime.UnixNano())},
		},
		{
			name:   "Success: Time key already has _unix_ns suffix",
			prefix: "ns.",
			attrs:  []slog.Attr{slog.Time("created_unix_ns", testTime)},
			want:   []attribute.KeyValue{attribute.Int64("ns.created_unix_ns", testTime.UnixNano())},
		},
		{
			name:   "Success: Group uses dotted keys",
			prefix: "ns.",
			attrs: []slog.Attr{
				slog.Group("user", slog.String("name", "john"), slog.Group("id", slog.Int64("num", 30))),
			},
			want: []attribute.KeyValue{
				attribute.String("ns.user.name", "john"),
				attribute.Int64("ns.user.id.num", 30),
			},
		},
		{
			name:   "Success: Group with empty key is inlined",
			prefix: "ns.",
			attrs:  []slog.Attr{slog.Group("", slog.String("a", "1"))},
			want:   []attribute.KeyValue{attribute.String("ns.a", "1")},
		},
		{
			name:  "Success: Empty key is skipped",
			attrs: []slog.Attr{slog.String("", "x"), slog.String("k", "v")},
			want:  []attribute.KeyValue{attribute.String("k", "v")},
		},
		{
			name:  "Success: LogValuer is resolved",
			attrs: []slog.Attr{slog.Any("lv", testLogValuer{slog.StringValue("resolved")})},
			want:  []attribute.KeyValue{attribute.String("lv", "resolved")},
		},
		{
			name:  "Success: LogValuer inside a group is resolved",
			attrs: []slog.Attr{slog.Group("g", slog.Any("lv", testLogValuer{slog.Int64Value(7)}))},
			want:  []attribute.KeyValue{attribute.Int64("g.lv", 7)},
		},
		{
			name: "Success: LogValuer resolving to a group uses dotted keys",
			attrs: []slog.Attr{
				slog.Any("target", testLogValuer{slog.GroupValue(slog.String("node", "node-1"))}),
			},
			want: []attribute.KeyValue{attribute.String("target.node", "node-1")},
		},
		{
			name:  "Success: KindAny is skipped",
			attrs: []slog.Attr{slog.Any("data", struct{ X int }{X: 1})},
			want:  nil,
		},
		{
			name:   "Success: Appends to existing dst",
			dst:    []attribute.KeyValue{attribute.String("existing", "attr")},
			prefix: "ns.",
			attrs:  []slog.Attr{slog.String("a", "1"), slog.Int64("b", 2), slog.Bool("c", true)},
			want: []attribute.KeyValue{
				attribute.String("existing", "attr"),
				attribute.String("ns.a", "1"),
				attribute.Int64("ns.b", 2),
				attribute.Bool("ns.c", true),
			},
		},
		{
			name:  "Success: Empty prefix",
			attrs: []slog.Attr{slog.String("key", "value")},
			want:  []attribute.KeyValue{attribute.String("key", "value")},
		},
		{
			name:   "Success: No attrs returns dst unchanged",
			prefix: "ns.",
			want:   nil,
		},
	}

	for _, test := range tests {
		got := AppendSlogAttrs(test.dst, test.prefix, test.attrs...)
		if diff := pretty.Compare(test.want, got); diff != "" {
			t.Errorf("TestAppendSlogAttrs(%s): -want/+got:\n%s", test.name, diff)
		}
	}
}

func BenchmarkAppendSlogAttrs(b *testing.B) {
	attrs := []slog.Attr{
		slog.String("name", "test"),
		slog.Int64("count", 42),
		slog.Duration("elapsed", time.Second),
		slog.Group("user", slog.String("name", "john"), slog.Int64("age", 30)),
	}
	dst := make([]attribute.KeyValue, 0, 8)

	b.ReportAllocs()
	for b.Loop() {
		dst = AppendSlogAttrs(dst[:0], "ns.", attrs...)
	}
}
