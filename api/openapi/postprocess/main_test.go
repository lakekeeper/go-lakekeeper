package main

import (
	"strings"
	"testing"
)

func TestStripDisallowUnknownFields_RemovesModelCall(t *testing.T) {
	src := `func (o *X) UnmarshalJSON(data []byte) (err error) {
	varX := _X{}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	err = decoder.Decode(&varX)
	*o = X(varX)
	return err
}
`
	out, n := stripDisallowUnknownFields(src)
	if n != 1 {
		t.Fatalf("removed: got %d, want 1", n)
	}
	if strings.Contains(out, "decoder.DisallowUnknownFields()") {
		t.Error("strict-decode call still present")
	}
	// The decoder is still declared and used, so the result stays valid Go.
	if !strings.Contains(out, "decoder := json.NewDecoder") {
		t.Error("decoder declaration was wrongly removed")
	}
	if !strings.Contains(out, "err = decoder.Decode(&varX)") {
		t.Error("decoder.Decode call was wrongly removed")
	}
}

func TestStripDisallowUnknownFields_PreservesNewStrictDecoder(t *testing.T) {
	// utils.go's newStrictDecoder uses a `dec` receiver and must survive: it
	// only decodes the discriminator lookup into a map, where the call is inert.
	src := `func newStrictDecoder(data []byte) *json.Decoder {
	dec := json.NewDecoder(bytes.NewBuffer(data))
	dec.DisallowUnknownFields()
	return dec
}
`
	out, n := stripDisallowUnknownFields(src)
	if n != 0 {
		t.Fatalf("removed: got %d, want 0 (newStrictDecoder must be untouched)", n)
	}
	if out != src {
		t.Error("newStrictDecoder source was modified")
	}
}

func TestTolerateUnknownEnumValues_StoresUnknown(t *testing.T) {
	src := `func (v *WarehouseAction) UnmarshalJSON(src []byte) error {
	var value string
	err := json.Unmarshal(src, &value)
	if err != nil {
		return err
	}
	enumTypeValue := WarehouseAction(value)
	for _, existing := range AllowedWarehouseActionEnumValues {
		if existing == enumTypeValue {
			*v = enumTypeValue
			return nil
		}
	}

	return fmt.Errorf("%+v is not a valid WarehouseAction", value)
}
`
	out, n := tolerateUnknownEnumValues(src)
	if n != 1 {
		t.Fatalf("relaxed: got %d, want 1", n)
	}
	if strings.Contains(out, "is not a valid") {
		t.Error("strict enum error still present")
	}
	if !strings.Contains(out, "*v = enumTypeValue\n\treturn nil") {
		t.Errorf("expected store-and-return replacement, got:\n%s", out)
	}
}

func TestTolerateUnknownEnumValues_NoMatchUnchanged(t *testing.T) {
	src := "func f() error {\n\treturn fmt.Errorf(\"something else\", x)\n}\n"
	out, n := tolerateUnknownEnumValues(src)
	if n != 0 || out != src {
		t.Errorf("non-enum error was modified: n=%d", n)
	}
}

func TestStripDisallowUnknownFields_Idempotent(t *testing.T) {
	src := `	decoder.DisallowUnknownFields()
	x := 1
`
	once, _ := stripDisallowUnknownFields(src)
	twice, n := stripDisallowUnknownFields(once)
	if n != 0 {
		t.Fatalf("second pass removed %d, want 0", n)
	}
	if once != twice {
		t.Error("not idempotent")
	}
}
