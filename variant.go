package gdvariant

import (
	"encoding/binary"
	"fmt"
	"io"
	"math"
	"reflect"

	"github.com/mitchellh/mapstructure"
)

func readHeader(r io.Reader) (header uint32, err error) {
	err = binary.Read(r, binary.LittleEndian, &header)
	return
}

type Decoder struct {
	r io.Reader
}

func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{
		r: r,
	}
}

func (d *Decoder) Decode(i interface{}) error {
	o, err := decodeObj(d.r)
	if err != nil {
		return err
	}
	return mapstructure.Decode(o, i)
}

func decodeObj(r io.Reader) (o interface{}, err error) {
	typ, err := readHeader(r)
	if err != nil {
		return nil, err
	}
	switch typ {
	case StringType:
		o, err = decodeStr(r)
	case IntegerType:
		i := new(Integer)
		buf := make([]byte, 4)
		if _, err := io.ReadAtLeast(r, buf, 4); err != nil {
			return nil, err
		}
		if err := i.UnmarshalVariant(buf); err != nil {
			return nil, err
		}
		o = *i
	case FloatType:
		f := new(Float)
		buf := make([]byte, 4)
		if _, err := io.ReadAtLeast(r, buf, 4); err != nil {
			return nil, err
		}
		if err := f.UnmarshalVariant(buf); err != nil {
			return nil, err
		}
		o = *f
	case DictionaryType:
		o, err = decodeDict(r)
	case Vector3Type:
		v := new(Vector3)
		buf := make([]byte, 12)
		if _, err := io.ReadAtLeast(r, buf, 12); err != nil {
			return nil, err
		}
		if err := v.UnmarshalVariant(buf); err != nil {
			return nil, err
		}
		o = *v
	case IntegerArrayType:
		o, err = decodeIntegerArray(r)
	case FloatArrayType:
		o, err = decodeFloatArray(r)
	case ArrayType:
		o, err = decodeGenericArray(r)
	default:
		b := make([]byte, 4)
		binary.LittleEndian.PutUint32(b, typ)
		err = fmt.Errorf("unsupported decode type %d %v", typ, b)
	}
	return
}

func decodeDict(r io.Reader) (map[string]interface{}, error) {
	d := make(map[string]interface{})
	header, err := readHeader(r)
	if err != nil {
		return nil, err
	}
	elements := (header & 0x7FFFFFFF)
	for i := uint32(0); i < elements; i++ {
		o, err := decodeObj(r)
		if err != nil {
			return nil, err
		}
		k, err := makeString(o)
		if err != nil {
			return nil, err
		}
		v, err := decodeObj(r)
		if err != nil {
			return nil, err
		}
		d[k] = v
	}
	return d, nil
}

func decodeIntegerArray(r io.Reader) ([]int32, error) {
	header, err := readHeader(r)
	if err != nil {
		return nil, err
	}
	size := int(header)
	a := make([]int32, size)
	for i := range a {
		v, err := ReadInt32(r)
		if err != nil {
			return nil, err
		}
		a[i] = v
	}
	return a, nil
}

func decodeFloatArray(r io.Reader) ([]float32, error) {
	header, err := readHeader(r)
	if err != nil {
		return nil, err
	}
	size := int(header)
	a := make([]float32, size)
	for i := range a {
		v, err := ReadFloat32(r)
		if err != nil {
			return nil, err
		}
		a[i] = v
	}
	return a, nil
}

func decodeGenericArray(r io.Reader) ([]interface{}, error) {
	header, err := readHeader(r)
	if err != nil {
		return nil, err
	}
	size := int(header & 0x7FFFFFFF)
	a := make([]interface{}, size)
	for i := range a {
		v, err := decodeObj(r)
		if err != nil {
			return nil, err
		}
		a[i] = v
	}
	return a, nil
}

func makeString(o interface{}) (string, error) {
	if s, ok := o.(string); ok {
		return s, nil
	}
	if s, ok := o.(fmt.Stringer); ok {
		return s.String(), nil
	}
	return "", fmt.Errorf("cannot convert %T to string", o)
}

var discardPaddingBuf = make([]byte, 3)

func discardPadding(r io.Reader, size int) error {
	padding := 4 - (size % 4)
	if padding > 0 && padding < 4 {
		_, err := io.ReadAtLeast(r, discardPaddingBuf[:padding], padding)
		return err
	}
	return nil
}

func decodeStr(r io.Reader) (string, error) {
	header, err := readHeader(r)
	if err != nil {
		return "", err
	}
	size := int(header)
	str := make([]byte, size)
	_, err = io.ReadAtLeast(r, str, size)
	if err != nil {
		return "", err
	}
	discardPadding(r, size)
	return string(str), nil
}

type Encoder struct {
	w io.Writer
}

func NewEncoder(w io.Writer) *Encoder {
	return &Encoder{w: w}
}

func (e *Encoder) Encode(i interface{}) error {
	v := reflect.ValueOf(i)
	return e.encodeObj(v)
}

func (e *Encoder) encodeObj(v reflect.Value) error {

	if v.Type().Implements(marshalerType) {
		m := v.Interface().(VariantMarshaler)
		data, err := m.MarshalVariant()
		if err != nil {
			return err
		}
		return e.writePadded(data)
	}

	switch k := v.Kind(); k {
	case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		i := v.Convert(integerType)
		return e.encodeObj(i)
	case reflect.Float32, reflect.Float64:
		f := v.Convert(floatType)
		return e.encodeObj(f)
	case reflect.String:
		return e.encodeStr(v.String())
	case reflect.Slice:
		return e.encodeSlice(v)
	case reflect.Struct:
		return e.encodeStruct(v)
	case reflect.Map:
		return e.encodeMap(v)
	default:
		return fmt.Errorf("unsupported kind %s", k)
	}
}

func (e *Encoder) writeHeader(header uint32) error {
	return WriteHeader(e.w, header)
}

var paddingBuf = make([]byte, 3)

func (e *Encoder) writePadded(data []byte) error {
	if _, err := e.w.Write(data); err != nil {
		return err
	}
	padding := 4 - (len(data) % 4)
	if padding > 0 && padding < 4 {
		if _, err := e.w.Write(paddingBuf[:padding]); err != nil {
			return err
		}
	}
	return nil
}

func (e *Encoder) encodeStr(s string) error {
	if err := e.writeHeader(StringType); err != nil {
		return err
	}
	size := uint32(len(s))
	if err := e.writeHeader(size); err != nil {
		return err
	}
	return e.writePadded([]byte(s))
}

func (e *Encoder) writeDictHeader(size int) error {
	if err := e.writeHeader(DictionaryType); err != nil {
		return err
	}
	elements := uint32(size)
	var header uint32
	// Set shared bit
	header |= 0x80000000
	// Set element count
	header |= 0x7FFFFFFF & elements
	return e.writeHeader(header)
}

func (e *Encoder) encodeSlice(v reflect.Value) error {
	et := v.Type().Elem()
	switch et.Kind() {
	case reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return e.encodeUIntegerArray(v)
	case reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return e.encodeIntegerArray(v)
	case reflect.Float32, reflect.Float64:
		return e.encodeFloatArray(v)
	default:
		return e.encodeGenericArray(v)
	}
}

func (e *Encoder) encodeUIntegerArray(v reflect.Value) error {
	if err := e.writeHeader(IntegerArrayType); err != nil {
		return err
	}
	n := v.Len()
	if err := e.writeHeader(uint32(n)); err != nil {
		return err
	}
	for i := 0; i < n; i++ {
		intValue := int32(v.Index(i).Uint())
		if err := WriteInt32(e.w, intValue); err != nil {
			return err
		}
	}
	return nil
}

func (e *Encoder) encodeIntegerArray(v reflect.Value) error {
	if err := e.writeHeader(IntegerArrayType); err != nil {
		return err
	}
	n := v.Len()
	if err := e.writeHeader(uint32(n)); err != nil {
		return err
	}
	for i := 0; i < n; i++ {
		intValue := int32(v.Index(i).Int())
		if err := WriteInt32(e.w, intValue); err != nil {
			return err
		}
	}
	return nil
}

func (e *Encoder) encodeFloatArray(v reflect.Value) error {
	if err := e.writeHeader(FloatArrayType); err != nil {
		return err
	}
	n := v.Len()
	if err := e.writeHeader(uint32(n)); err != nil {
		return err
	}
	for i := 0; i < n; i++ {
		floatValue := float32(v.Index(i).Float())
		if err := WriteFloat32(e.w, floatValue); err != nil {
			return err
		}
	}
	return nil
}

func (e *Encoder) encodeGenericArray(v reflect.Value) error {
	if err := e.writeHeader(ArrayType); err != nil {
		return err
	}
	n := v.Len()
	var header uint32
	header |= 0x80000000
	header |= uint32(n) & 0x7FFFFFFF
	if err := e.writeHeader(header); err != nil {
		return err
	}
	for i := 0; i < n; i++ {
		if err := e.encodeObj(v.Index(i)); err != nil {
			return err
		}
	}
	return nil
}

func (e *Encoder) encodeStruct(v reflect.Value) error {
	n := v.NumField()
	if err := e.writeDictHeader(n); err != nil {
		return nil
	}

	t := v.Type()
	for i := 0; i < n; i++ {
		f := t.Field(i)
		if err := e.encodeStr(f.Name); err != nil {
			return err
		}
		value := v.Field(i)
		if err := e.encodeObj(value); err != nil {
			return err
		}
	}
	return nil
}

func (e *Encoder) encodeMap(v reflect.Value) error {
	keys := v.MapKeys()
	n := len(keys)
	if err := e.writeDictHeader(n); err != nil {
		return nil
	}
	for _, k := range keys {
		if err := e.encodeObj(k); err != nil {
			return err
		}
		value := v.MapIndex(k)
		if err := e.encodeObj(value); err != nil {
			return err
		}
	}
	return nil
}

func Float32FromBytes(bytes []byte) float32 {
	bits := binary.LittleEndian.Uint32(bytes)
	float := math.Float32frombits(bits)
	return float
}

func Int32FromBytes(bytes []byte) int32 {
	return int32(binary.LittleEndian.Uint32(bytes))
}

func Float32ToBytes(float float32) []byte {
	bits := math.Float32bits(float)
	bytes := make([]byte, 4)
	binary.LittleEndian.PutUint32(bytes, bits)
	return bytes
}

func ReadFloat32(r io.Reader) (float32, error) {
	var bits uint32
	if err := binary.Read(r, binary.LittleEndian, &bits); err != nil {
		return 0, err
	}
	float := math.Float32frombits(bits)
	return float, nil
}

func ReadInt32(r io.Reader) (i int32, err error) {
	err = binary.Read(r, binary.LittleEndian, &i)
	return
}

func WriteFloat32(w io.Writer, float float32) error {
	bits := math.Float32bits(float)
	return binary.Write(w, binary.LittleEndian, bits)
}

func WriteInt32(w io.Writer, i int32) error {
	return binary.Write(w, binary.LittleEndian, i)
}
func WriteUint32(w io.Writer, i uint32) error {
	return binary.Write(w, binary.LittleEndian, i)
}

func WriteHeader(w io.Writer, header uint32) error {
	return binary.Write(w, binary.LittleEndian, header)
}

func ReadHeaderFromBytes(bytes []byte) uint32 {
	return binary.LittleEndian.Uint32(bytes)
}
