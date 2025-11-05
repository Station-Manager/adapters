package adapters

import "testing"

// Simple source and destination structs to verify JSON round-trip conversion works.
type src struct {
	A string `json:"a"`
	B int    `json:"b"`
}

type dst struct {
	A string `json:"a"`
	B int    `json:"b"`
}

func Test_convert_roundtrip(t *testing.T) {
	in := src{A: "hello", B: 42}
	var out dst
	if err := convert(in, &out); err != nil {
		t.Fatalf("convert failed: %v", err)
	}
	if out.A != in.A || out.B != in.B {
		t.Fatalf("unexpected result: got %+v, want %+v", out, in)
	}
}
