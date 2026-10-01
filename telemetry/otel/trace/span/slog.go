package span

import (
	"log/slog"
	"math"
	"strconv"
	"strings"

	"go.opentelemetry.io/otel/attribute"
)

// AppendSlogAttrs converts attrs to OpenTelemetry attributes and appends them to dst.
// Each key is prefixed with prefix, which should be empty or end with ".".
//
// Values are resolved first, so slog.LogValuer values are recorded by their resolved value.
// Conversions follow slog conventions:
//   - Bool, Int64, Float64 and String keep their type.
//   - Uint64 is recorded as an int64 when it fits, otherwise as its decimal string.
//   - Duration is recorded as int64 nanoseconds and "_ns" is appended to the key unless present.
//   - Time is recorded as int64 Unix nanoseconds and "_unix_ns" is appended to the key unless present.
//   - Group keys become a dotted namespace, such as "group.key". Groups with an empty key are inlined.
//
// Attributes with an empty key and values of kind slog.KindAny are skipped.
func AppendSlogAttrs(dst []attribute.KeyValue, prefix string, attrs ...slog.Attr) []attribute.KeyValue {
	for _, a := range attrs {
		v := a.Value.Resolve()
		if v.Kind() == slog.KindGroup {
			p := prefix
			if a.Key != "" {
				p = prefix + a.Key + "."
			}
			dst = AppendSlogAttrs(dst, p, v.Group()...)
			continue
		}
		if a.Key == "" {
			continue
		}
		key := prefix + a.Key
		switch v.Kind() {
		case slog.KindString:
			dst = append(dst, attribute.String(key, v.String()))
		case slog.KindBool:
			dst = append(dst, attribute.Bool(key, v.Bool()))
		case slog.KindInt64:
			dst = append(dst, attribute.Int64(key, v.Int64()))
		case slog.KindFloat64:
			dst = append(dst, attribute.Float64(key, v.Float64()))
		case slog.KindUint64:
			if u := v.Uint64(); u <= math.MaxInt64 {
				dst = append(dst, attribute.Int64(key, int64(u)))
			} else {
				dst = append(dst, attribute.String(key, strconv.FormatUint(u, 10)))
			}
		case slog.KindDuration:
			if !strings.HasSuffix(key, "_ns") {
				key += "_ns"
			}
			dst = append(dst, attribute.Int64(key, int64(v.Duration())))
		case slog.KindTime:
			if !strings.HasSuffix(key, "_unix_ns") {
				key += "_unix_ns"
			}
			dst = append(dst, attribute.Int64(key, v.Time().UnixNano()))
		}
	}
	return dst
}
