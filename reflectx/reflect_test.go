package reflectx

import (
	"math"
	"net/url"
	"reflect"
	"strconv"
	"testing"
	"time"
	"unsafe"

	"github.com/samuelsih/golib/assert"
)

// level is a custom primitive type that parses itself with encoding.TextUnmarshaler.
type level int

func (l *level) UnmarshalText(text []byte) error {
	n, err := strconv.Atoi(string(text))
	if err != nil {
		return err
	}

	*l = level(n)

	return nil
}

func TestIsEmptyValue(t *testing.T) {
	var (
		nilPtr      *int
		nilMap      map[string]int
		nilSlice    []int
		nilChan     chan int
		nilFunc     func()
		nilIface    any
		filledIface any = "kudaponi"
	)

	tests := []struct {
		name  string
		value reflect.Value
		want  bool
	}{
		{name: "invalid value", value: reflect.Value{}, want: true},

		{name: "empty array", value: reflect.ValueOf([0]int{}), want: true},
		{name: "zero-valued array", value: reflect.ValueOf([2]int{}), want: false},

		{name: "nil map", value: reflect.ValueOf(nilMap), want: true},
		{name: "empty map", value: reflect.ValueOf(map[string]int{}), want: true},
		{name: "non-empty map", value: reflect.ValueOf(map[string]int{"kudaponi": 1}), want: false},

		{name: "nil slice", value: reflect.ValueOf(nilSlice), want: true},
		{name: "empty slice", value: reflect.ValueOf([]int{}), want: true},
		{name: "non-empty slice", value: reflect.ValueOf([]int{1}), want: false},

		{name: "empty string", value: reflect.ValueOf(""), want: true},
		{name: "non-empty string", value: reflect.ValueOf("kudaponi"), want: false},

		{name: "false bool", value: reflect.ValueOf(false), want: true},
		{name: "true bool", value: reflect.ValueOf(true), want: false},

		{name: "zero int", value: reflect.ValueOf(int(0)), want: true},
		{name: "non-zero int", value: reflect.ValueOf(int(42)), want: false},
		{name: "negative int", value: reflect.ValueOf(int(-1)), want: false},
		{name: "zero int32", value: reflect.ValueOf(int32(0)), want: true},
		{name: "non-zero int32", value: reflect.ValueOf(int32(1)), want: false},

		{name: "zero uint", value: reflect.ValueOf(uint(0)), want: true},
		{name: "non-zero uint", value: reflect.ValueOf(uint(1)), want: false},
		{name: "zero uint64", value: reflect.ValueOf(uint64(0)), want: true},
		{name: "zero uintptr", value: reflect.ValueOf(uintptr(0)), want: true},
		{name: "non-zero uintptr", value: reflect.ValueOf(uintptr(1)), want: false},

		{name: "zero float32", value: reflect.ValueOf(float32(0)), want: true},
		{name: "zero float64", value: reflect.ValueOf(float64(0)), want: true},
		{name: "negative zero float64", value: reflect.ValueOf(math.Copysign(0, -1)), want: true},
		{name: "non-zero float64", value: reflect.ValueOf(1.5), want: false},

		{name: "nil interface", value: reflect.ValueOf(&nilIface).Elem(), want: true},
		{name: "non-nil interface", value: reflect.ValueOf(&filledIface).Elem(), want: false},

		{name: "nil pointer", value: reflect.ValueOf(nilPtr), want: true},
		{name: "non-nil pointer", value: reflect.ValueOf(new(int)), want: false},

		{name: "nil chan", value: reflect.ValueOf(nilChan), want: true},
		{name: "non-nil chan", value: reflect.ValueOf(make(chan int)), want: false},

		{name: "nil func", value: reflect.ValueOf(nilFunc), want: true},
		{name: "non-nil func", value: reflect.ValueOf(func() {}), want: false},

		{name: "zero struct", value: reflect.ValueOf(struct{}{}), want: true},
		{name: "non-zero struct", value: reflect.ValueOf(struct{ Name string }{Name: "mario"}), want: false},
		{name: "zero complex", value: reflect.ValueOf(complex(0, 0)), want: true},
		{name: "non-zero complex", value: reflect.ValueOf(complex(1, 0)), want: false},

		{name: "nil unsafe pointer", value: reflect.ValueOf(unsafe.Pointer(nil)), want: true},
		{name: "non-nil unsafe pointer", value: reflect.ValueOf(unsafe.Pointer(new(int))), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, IsEmptyValue(tt.value), tt.want)
		})
	}
}

func TestGetTagValueAs(t *testing.T) {
	type tagged struct {
		Int      int            `env:"42"`
		Uint     uint           `env:"7"`
		Float    float64        `env:"1.5"`
		Complex  complex128     `env:"1+2i"`
		Bool     bool           `env:"true"`
		Str      string         `env:"kudaponi"`
		Level    level          `env:"3"`
		BadInt   int            `env:"not-a-number"`
		BadBool  bool           `env:"not-a-bool"`
		Slice    []string       `env:"a,b"`
		Time     time.Time      `env:"2024-01-02T03:04:05Z"`
		BadTime  time.Time      `env:"not-a-time"`
		Duration time.Duration  `env:"5s"`
		Location *time.Location `env:"Asia/Jakarta"`
		URL      *url.URL       `env:"https://example.com/path?q=1"`
		NoTag    int
		Empty    int `env:""`
	}

	field := func(t *testing.T, name string) reflect.StructField {
		t.Helper()

		sf, ok := reflect.TypeFor[tagged]().FieldByName(name)
		if !ok {
			t.Fatalf("field %q not found", name)
		}

		return sf
	}

	t.Run("string", func(t *testing.T) {
		assert.Equal(t, GetTagValueAs[string](field(t, "Str"), "env"), "kudaponi")
	})

	t.Run("int", func(t *testing.T) {
		assert.Equal(t, GetTagValueAs[int](field(t, "Int"), "env"), 42)
	})

	t.Run("uint", func(t *testing.T) {
		assert.Equal(t, GetTagValueAs[uint](field(t, "Uint"), "env"), uint(7))
	})

	t.Run("float", func(t *testing.T) {
		assert.Equal(t, GetTagValueAs[float64](field(t, "Float"), "env"), 1.5)
	})

	t.Run("complex", func(t *testing.T) {
		assert.Equal(t, GetTagValueAs[complex128](field(t, "Complex"), "env"), complex(1, 2))
	})

	t.Run("bool", func(t *testing.T) {
		assert.Equal(t, GetTagValueAs[bool](field(t, "Bool"), "env"), true)
	})

	t.Run("custom type via text unmarshaler", func(t *testing.T) {
		assert.Equal(t, GetTagValueAs[level](field(t, "Level"), "env"), level(3))
	})

	t.Run("time.Duration", func(t *testing.T) {
		assert.Equal(t, GetTagValueAs[time.Duration](field(t, "Duration"), "env"), 5*time.Second)
	})

	t.Run("time.Location", func(t *testing.T) {
		want, err := time.LoadLocation("Asia/Jakarta")
		if err != nil {
			t.Fatalf("load location: %v", err)
		}

		got := GetTagValueAs[*time.Location](field(t, "Location"), "env")
		if got == nil {
			t.Fatal("got nil location")
		}

		assert.Equal(t, got.String(), want.String())
	})

	t.Run("url.URL", func(t *testing.T) {
		want, err := url.Parse("https://example.com/path?q=1")
		if err != nil {
			t.Fatalf("parse url: %v", err)
		}

		assert.Equal(t, GetTagValueAs[*url.URL](field(t, "URL"), "env"), want)
	})

	t.Run("text unmarshaler", func(t *testing.T) {
		want := time.Date(2024, time.January, 2, 3, 4, 5, 0, time.UTC)
		assert.Equal(t, GetTagValueAs[time.Time](field(t, "Time"), "env"), want)
	})

	t.Run("unparseable text unmarshaler", func(t *testing.T) {
		assert.Equal(t, GetTagValueAs[time.Time](field(t, "BadTime"), "env"), time.Time{})
	})

	t.Run("missing key", func(t *testing.T) {
		assert.Equal(t, GetTagValueAs[int](field(t, "Int"), "json"), 0)
	})

	t.Run("unparseable int", func(t *testing.T) {
		assert.Equal(t, GetTagValueAs[int](field(t, "BadInt"), "env"), 0)
	})

	t.Run("unparseable bool", func(t *testing.T) {
		assert.Equal(t, GetTagValueAs[bool](field(t, "BadBool"), "env"), false)
	})

	t.Run("unsupported kind", func(t *testing.T) {
		assert.Equal(t, GetTagValueAs[[]string](field(t, "Slice"), "env"), nil)
	})

	t.Run("empty tag value", func(t *testing.T) {
		assert.Equal(t, GetTagValueAs[int](field(t, "Empty"), "env"), 0)
	})

	t.Run("field without tags", func(t *testing.T) {
		assert.Equal(t, GetTagValueAs[string](field(t, "NoTag"), "env"), "")
	})
}

func TestWalkStruct(t *testing.T) {
	collect := func(typ reflect.Type) []string {
		var got []string
		WalkStruct(typ, func(sf reflect.StructField) {
			got = append(got, sf.Name)
		})
		return got
	}

	type (
		inner struct{ X int }
		outer struct {
			Inner inner
			Name  string
		}
		leaf struct{ L int }
		pair struct {
			First  leaf
			Second leaf
		}
		node struct {
			Value int
			Next  *node
		}
	)

	tests := []struct {
		name string
		typ  reflect.Type
		want []string
	}{
		{name: "flat", typ: reflect.TypeFor[struct {
			B string
			C int
		}](), want: []string{"B", "C"}},
		{name: "pointer to struct", typ: reflect.TypeFor[*struct {
			B string
			C int
		}](), want: []string{"B", "C"}},
		{name: "nested", typ: reflect.TypeFor[outer](), want: []string{"Inner", "X", "Name"}},
		{name: "same type twice", typ: reflect.TypeFor[pair](), want: []string{"First", "L", "Second", "L"}},
		{name: "recursive", typ: reflect.TypeFor[node](), want: []string{"Value", "Next"}},
		{name: "non-struct", typ: reflect.TypeFor[int](), want: nil},
		{name: "pointer to non-struct", typ: reflect.TypeFor[*int](), want: nil},
		{name: "nil type", typ: nil, want: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, collect(tt.typ), tt.want)
		})
	}

	WalkStruct(reflect.TypeFor[struct{ B string }](), nil)
	WalkStruct(nil, nil)
}
