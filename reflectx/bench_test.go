package reflectx

import (
	"net/url"
	"reflect"
	"testing"
	"time"
)

type benchStruct struct {
	Int      int            `tag:"42"`
	Uint     uint           `tag:"7"`
	Float64  float64        `tag:"1.5"`
	Complex  complex128     `tag:"1+2i"`
	Bool     bool           `tag:"true"`
	String   string         `tag:"hello"`
	Duration time.Duration  `tag:"5s"`
	Time     time.Time      `tag:"2024-01-02T03:04:05Z"`
	Location *time.Location `tag:"Asia/Jakarta"`
	URL      *url.URL       `tag:"https://example.com/path?q=1"`
	Slice    []string       `tag:"x"`
	Missing  int
}

func benchField(b *testing.B, name string) reflect.StructField {
	b.Helper()

	sf, ok := reflect.TypeFor[benchStruct]().FieldByName(name)
	if !ok {
		b.Fatalf("field %q not found", name)
	}

	return sf
}

func BenchmarkGetTagValueAs(b *testing.B) {
	b.ReportAllocs()

	b.Run("string", func(b *testing.B) {
		sf := benchField(b, "String")

		for b.Loop() {
			_ = GetTagValueAs[string](sf, "tag")
		}
	})

	b.Run("bool", func(b *testing.B) {
		sf := benchField(b, "Bool")

		for b.Loop() {
			_ = GetTagValueAs[bool](sf, "tag")
		}
	})

	b.Run("int", func(b *testing.B) {
		sf := benchField(b, "Int")

		for b.Loop() {
			_ = GetTagValueAs[int](sf, "tag")
		}
	})

	b.Run("float64", func(b *testing.B) {
		sf := benchField(b, "Float64")

		for b.Loop() {
			_ = GetTagValueAs[float64](sf, "tag")
		}
	})

	b.Run("complex128", func(b *testing.B) {
		sf := benchField(b, "Complex")

		for b.Loop() {
			_ = GetTagValueAs[complex128](sf, "tag")
		}
	})

	b.Run("duration", func(b *testing.B) {
		sf := benchField(b, "Duration")

		for b.Loop() {
			_ = GetTagValueAs[time.Duration](sf, "tag")
		}
	})

	b.Run("text_unmarshaler", func(b *testing.B) {
		sf := benchField(b, "Time")

		for b.Loop() {
			_ = GetTagValueAs[time.Time](sf, "tag")
		}
	})

	b.Run("location", func(b *testing.B) {
		sf := benchField(b, "Location")

		for b.Loop() {
			_ = GetTagValueAs[*time.Location](sf, "tag")
		}
	})

	b.Run("url", func(b *testing.B) {
		sf := benchField(b, "URL")

		for b.Loop() {
			_ = GetTagValueAs[*url.URL](sf, "tag")
		}
	})

	b.Run("missing_key", func(b *testing.B) {
		sf := benchField(b, "Missing")

		for b.Loop() {
			_ = GetTagValueAs[string](sf, "tag")
		}
	})

	b.Run("unsupported", func(b *testing.B) {
		sf := benchField(b, "Slice")

		for b.Loop() {
			_ = GetTagValueAs[[]string](sf, "tag")
		}
	})
}
