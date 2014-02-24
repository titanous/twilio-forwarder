package underscore

import (
	"net/url"
	"reflect"
	"testing"
)

func TestStructToMap(t *testing.T) {
	type Foo struct {
		Int       int
		Uint      uint
		Float32   float32
		Float64   float64
		BoolTrue  bool
		BoolFalse bool
		String    string
		Slice     []string
	}

	s := Foo{
		Int:       -8,
		Uint:      8,
		Float32:   0.16,
		Float64:   1024.99,
		BoolTrue:  true,
		BoolFalse: false,
		String:    "hello",
		Slice:     []string{"go", "lang"},
	}

	v := StructToMap(&s)

	want := url.Values{
		"Int":       {"-8"},
		"Uint":      {"8"},
		"Float32":   {"0.1600"},
		"Float64":   {"1024.9900"},
		"BoolTrue":  {"true"},
		"BoolFalse": {"false"},
		"String":    {"hello"},
		"Slice":     {"go", "lang"},
	}

	if !reflect.DeepEqual(v, want) {
		t.Errorf("StructToMap returned %#v, want %#v", v, want)
	}
}

func TestStructToMap_customBoolFlag(t *testing.T) {
	// Use 0/1 instead of false/true
	BoolFlag = []string{"0", "1"}

	type Foo struct {
		BoolTrue  bool
		BoolFalse bool
	}

	s := Foo{BoolTrue: true, BoolFalse: false}
	v := StructToMap(&s)

	want := url.Values{
		"BoolTrue":  {"1"},
		"BoolFalse": {"0"},
	}

	if !reflect.DeepEqual(v, want) {
		t.Errorf("StructToMap returned %#v, want %#v", v, want)
	}
}
