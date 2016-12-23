package gdvariant

import (
	"bytes"
	"io"
	"reflect"
)

const (
	NullType         uint32 = iota
	BoolType                = 1
	IntegerType             = 2
	FloatType               = 3
	StringType              = 4
	Vector2Type             = 5
	Rect2Type               = 6
	Vector3Type             = 7
	Matrix32Type            = 8
	PlaneType               = 9
	QuaternionType          = 10
	AabbType                = 11 //(rect3)
	Matrix3x3Type           = 12
	TransformType           = 13 // (matrix 4x3)
	ColorType               = 14
	ImageType               = 15
	NodePathType            = 16 // path
	RidType                 = 17 // (unsupported)
	ObjectType              = 18 // (unsupported)
	InputEventType          = 19
	DictionaryType          = 20
	ArrayType               = 21
	ByteArrayType           = 22
	IntegerArrayType        = 23
	FloatArrayType          = 24
	StringArrayType         = 25
	Vector2ArrayType        = 26
	Vector3ArrayType        = 27
	ColorArrayType          = 28
)

type VariantMarshaler interface {
	MarshalVariant() ([]byte, error)
}

var marshalerType = reflect.TypeOf((*VariantMarshaler)(nil)).Elem()

type VariantUnmarshaler interface {
	UnmarshalVariant([]byte) error
}

var unmarshalerType = reflect.TypeOf((*VariantUnmarshaler)(nil)).Elem()

type Integer int32

var integerType = reflect.TypeOf(Integer(0))

func (i Integer) MarshalVariant() ([]byte, error) {
	var buf bytes.Buffer
	if err := WriteHeader(&buf, IntegerType); err != nil {
		return nil, err
	}
	if err := WriteInt32(&buf, int32(i)); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (i *Integer) UnmarshalVariant(data []byte) error {
	if len(data) < 4 {
		return io.ErrUnexpectedEOF
	}
	*i = Integer(Int32FromBytes(data[0:4]))
	return nil
}

type Float float32

var floatType = reflect.TypeOf(Float(0))

func (f Float) MarshalVariant() ([]byte, error) {
	var buf bytes.Buffer
	if err := WriteHeader(&buf, FloatType); err != nil {
		return nil, err
	}
	if err := WriteFloat32(&buf, float32(f)); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (f *Float) UnmarshalVariant(data []byte) error {
	if len(data) < 4 {
		return io.ErrUnexpectedEOF
	}
	*f = Float(Float32FromBytes(data[0:4]))
	return nil
}

type Vector3 struct {
	X, Y, Z float32
}

func (v Vector3) MarshalVariant() ([]byte, error) {
	var buf bytes.Buffer
	if err := WriteHeader(&buf, Vector3Type); err != nil {
		return nil, err
	}
	if err := WriteFloat32(&buf, v.X); err != nil {
		return nil, err
	}
	if err := WriteFloat32(&buf, v.Y); err != nil {
		return nil, err
	}
	if err := WriteFloat32(&buf, v.Z); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (v *Vector3) UnmarshalVariant(data []byte) error {
	if len(data) < 12 {
		return io.ErrUnexpectedEOF
	}
	v.X = Float32FromBytes(data[0:4])
	v.Y = Float32FromBytes(data[4:8])
	v.Z = Float32FromBytes(data[8:12])
	return nil
}
