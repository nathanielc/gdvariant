package gdvariant_test

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/nathanielc/gdvariant"
)

func TestVariant_EncodeDecode(t *testing.T) {
	type subobject struct {
		FieldB  string
		Vector  gdvariant.Vector3
		Options map[string]string
		List    []uint32
	}
	type object struct {
		FieldA   string
		Strength gdvariant.Float
		Mass     float32
		Radius   float64
		Count    int32
		Index    gdvariant.Integer
		Sub      subobject
	}

	exp := object{
		FieldA:   "field A",
		Strength: -5,
		Mass:     4,
		Radius:   6 * 9,
		Count:    9,
		Index:    -3,
		Sub: subobject{
			FieldB: "field B",
			Vector: gdvariant.Vector3{
				X: 42.0,
				Y: 4.2,
				Z: 0.42,
			},
			Options: map[string]string{
				"o1": "option 1",
				"o2": "option 2",
			},
			List: []uint32{43, 215, 16},
		},
	}

	var buf bytes.Buffer
	enc := gdvariant.NewEncoder(&buf)
	if err := enc.Encode(exp); err != nil {
		t.Fatal(err)
	}
	var got object
	if err := gdvariant.NewDecoder(&buf).Decode(&got); err != nil {
		t.Fatal(err)
	}

	if !reflect.DeepEqual(got, exp) {
		t.Errorf("unexpected object:\ngot\n%+v\nexp\n%+v\n", got, exp)
	}
}
