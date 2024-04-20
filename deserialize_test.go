package ezbson

import (
	"testing"
	timelib "time"

	"github.com/go-test/deep"
	"github.com/stretchr/testify/assert"
)

func TestDeserializeEmptyStruct(t *testing.T) {
	marshalled := []byte{
		0x05, 0x00, 0x00, 0x00, // Size
		0x00, // Terminator
	}

	expectedStruct := EmptyStruct{}
	st := EmptyStruct{}

	err := Unmarshal(marshalled, &st)
	if !assert.Nil(t, err) {
		return
	}

	if !assert.Equal(t, expectedStruct, st) {
		return
	}
}

func TestDeserializeHelloStruct(t *testing.T) {
	var err error

	marshalled := []byte{
		0x16, 0x00, 0x00, 0x00, // total document length
		0x02,                          // etype (string)
		'H', 'e', 'l', 'l', 'o', 0x00, // ename
		0x06, 0x00, 0x00, 0x00, // string-length (incl' nullterminator)
		'w', 'o', 'r', 'l', 'd', 0x00, // string-value
		0x00, // done
	}

	expectedStruct := HelloStruct{
		Hello: "world",
	}

	st := HelloStruct{}

	err = Unmarshal(marshalled, &st)
	if !assert.Nil(t, err) {
		return
	}

	if !assert.Equal(t, expectedStruct, st) {
		return
	}
}

func TestDeserializeVariousStruct(t *testing.T) {
	var err error

	expectedTime, err := timelib.Parse(timelib.RFC3339, "2006-01-02T15:04:05Z")
	if err != nil {
		t.Error(err)
		return
	}

	kMarshalled := []byte{
		0xa4, 0x00, 0x00, 0x00, // total document length

		0x05, // etype (binary)
		'B', 'i', 'n', 0x00,
		0x05, 0x00, 0x00, 0x00, // buffer-length
		0x00, // subtype
		'w', 'o', 'r', 'l', 'd',

		0x01, // etype-double
		'D', 'o', 'u', 'b', 'l', 'e', 0x00,
		0x33, 0x33, 0x33, 0x33, 0x33, 0x33, 0x14, 0x40,

		0x08, // etype-boolean
		'F', 'a', 'l', 's', 'e', 0x00,
		0x00,

		0x12,
		'I', 'n', 't', 0x00,
		0xef, 0xbe, 0xad, 0xde, 0xde, 0xc0, 0xad, 0x0b,

		0x10, // etype-int32
		'I', 'n', 't', '3', '2', 0x00,
		0xbe, 0xba, 0xad, 0x0b,

		0x12, //etype-int64
		'I', 'n', 't', '6', '4', 0x00,
		0xef, 0xbe, 0xad, 0xde, 0xde, 0xc0, 0xad, 0x0b,

		0x12,
		'M', 'i', 'n', 'u', 's', 0x00,
		0xfb, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,

		0x10, // etype-int32
		'M', 'i', 'n', 'u', 's', '3', '2', 0x00,
		0xfb, 0xff, 0xff, 0xff,

		0x12, // etype-int64
		'M', 'i', 'n', 'u', 's', '6', '4', 0x00,
		0xfb, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,

		0x02, // etype string
		'S', 't', 'r', 0x00,
		0x06, 0x00, 0x00, 0x00, // string-length + 1
		'w', 'o', 'r', 'l', 'd', 0x00,

		0x09, // etype-time
		'T', 'i', 'm', 'e', 0x00,
		0x88, 0x7e, 0xa5, 0x8b, 0x08, 0x01, 0x00, 0x00,

		0x08, //etype-bool
		'T', 'r', 'u', 'e', 0x00,
		0x01,

		0x00, // done
	}

	expectedStruct := VariousStruct{
		Bin:     []byte("world"),
		Int:     int(0x0badc0dedeadbeef),
		Int32:   int32(0x0badbabe),
		Int64:   int64(0x0badc0dedeadbeef),
		Minus:   int(-5),
		Minus32: int32(-5),
		Minus64: int64(-5),
		Double:  float64(5.05),
		Str:     "world",
		Time:    expectedTime,
		True:    true,
		False:   false,
	}

	st := VariousStruct{}

	err = Unmarshal(kMarshalled, &st)
	if !assert.Nil(t, err) {
		return
	}

	if !assert.Equal(t, expectedStruct, st) {
		return
	}
}

func TestDeserializeStructToStruct(t *testing.T) {
	kMarshalled := []byte{
		0x52, 0x00, 0x00, 0x00, // total document size

		0x02, // etype-string
		'A', 0x00,
		0x04, 0x00, 0x00, 0x00,
		'1', '2', '3', 0x00,

		0x03, // etype-doc
		'B', 0x00,
		0x1f, 0x00, 0x00, 0x00, // b's document size
		0x02, // etype-string
		'X', 0x00,
		0x06, 0x00, 0x00, 0x00,
		'h', 'e', 'l', 'l', 'o', 0x00,
		0x05, // etype-binary
		'Y', 0x00,
		0x05, 0x00, 0x00, 0x00,
		0x00,
		'w', 'o', 'r', 'l', 'd',
		0x00, // doc-term

		0x03, // etype-doc
		'C', 0x00,
		0x1d, 0x00, 0x00, 0x00,
		0x12, // etype int64
		'T', '1', 0x00,
		0xef, 0xbe, 0xad, 0xde, 0xde, 0xc0, 0xad, 0x0b,
		0x12, // type int64
		'T', '2', 0x00,
		0xbe, 0xba, 0xad, 0x0b, 0x00, 0x00, 0x00, 0x00,
		0x00, //doc-term

		0x00,
	}

	expected := EmbeddedDocStruct{
		A: "123",
		B: struct {
			X string
			Y []byte
		}{
			X: "hello",
			Y: []byte("world"),
		},
		C: struct {
			T1 int64
			T2 int64
		}{
			T1: 0x0badc0dedeadbeef,
			T2: 0x0badbabe,
		},
	}

	actual := EmbeddedDocStruct{}

	if err := Unmarshal(kMarshalled, &actual); !assert.Nil(t, err) {
		return
	}

	if !assert.Nil(t, deep.Equal(expected, actual)) {
		return
	}
}

type MapInStruct struct {
	A string
	B map[string]any
	C map[string]int64
}

func TestDeserializeStructToMap(t *testing.T) {
	kMarshalled := []byte{
		0x52, 0x00, 0x00, 0x00, // total document size

		0x02, // etype-string
		'A', 0x00,
		0x04, 0x00, 0x00, 0x00,
		'1', '2', '3', 0x00,

		0x03, // etype-doc
		'B', 0x00,
		0x1f, 0x00, 0x00, 0x00, // b's document size
		0x02, // etype-string
		'X', 0x00,
		0x06, 0x00, 0x00, 0x00,
		'h', 'e', 'l', 'l', 'o', 0x00,
		0x05, // etype-binary
		'Y', 0x00,
		0x05, 0x00, 0x00, 0x00,
		0x00,
		'w', 'o', 'r', 'l', 'd',
		0x00, // doc-term

		0x03, // etype-doc
		'C', 0x00,
		0x1d, 0x00, 0x00, 0x00,
		0x12, // etype int64
		'T', '1', 0x00,
		0xef, 0xbe, 0xad, 0xde, 0xde, 0xc0, 0xad, 0x0b,
		0x12, // type int64
		'T', '2', 0x00,
		0xbe, 0xba, 0xad, 0x0b, 0x00, 0x00, 0x00, 0x00,
		0x00, //doc-term

		0x00,
	}

	expected := MapInStruct{
		A: "123",
		B: map[string]any{
			"X": "hello",
			"Y": []byte("world"),
		},
		C: map[string]int64{
			"T1": 0x0badc0dedeadbeef,
			"T2": 0x0badbabe,
		},
	}

	actual := MapInStruct{}

	if err := Unmarshal(kMarshalled, &actual); !assert.Nil(t, err) {
		return
	}

	if !assert.Nil(t, deep.Equal(expected, actual)) {
		return
	}
}

func TestDeserializeStructToSlice_SliceInt64(t *testing.T) {
	kMarshalled := []byte{
		0x26, 0x00, 0x00, 0x00, // total document size

		0x04, // etype-array
		'B', 'S', 'O', 'N', 0x00,
		0x1b, 0x00, 0x00, 0x00,

		0x12, // etype int64
		'0', 0x00,
		0xde, 0xc0, 0xad, 0x0b, 0xde, 0xc0, 0xad, 0x0b,

		0x12, // etype int64
		'1', 0x00,
		0xbe, 0xba, 0xad, 0xde, 0x00, 0x00, 0x00, 0x00,

		0x00, // end slice

		0x00, // end doc
	}

	expected := EmbeddedArrayStructInt64{
		BSON: []int64{0x0badc0de0badc0de, 0xdeadbabe},
	}

	actual := EmbeddedArrayStructInt64{}

	if err := Unmarshal(kMarshalled, &actual); !assert.Nil(t, err) {
		return
	}

	if !assert.Nil(t, deep.Equal(expected, actual)) {
		return
	}
}

func TestDeserializeStructToSlice_SliceAny(t *testing.T) {
	kMarshalled := []byte{
		0x35, 0x00, 0x00, 0x00, // total document size

		0x04, // etype-array
		'B', 'S', 'O', 'N', 0x00,
		0x2a, 0x00, 0x00, 0x00,

		0x02, // etype-string
		'0', 0x00,
		0x08, 0x00, 0x00, 0x00,
		'a', 'w', 'e', 's', 'o', 'm', 'e', 0x00,

		0x01, // etype-double
		'1', 0x00,
		0x33, 0x33, 0x33, 0x33, 0x33, 0x33, 0x14, 0x40,

		0x12, // etype int64
		'2', 0x00,
		0xc2, 0x07, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,

		0x00, // end slice

		0x00, // end doc
	}

	expected := EmbeddedArrayStruct{
		BSON: []any{"awesome", float64(5.05), int64(1986)},
	}

	actual := EmbeddedArrayStruct{}

	if err := Unmarshal(kMarshalled, &actual); !assert.Nil(t, err) {
		return
	}

	if !assert.Nil(t, deep.Equal(expected, actual)) {
		return
	}
}

type TwoIntStruct struct {
	T1 int64
	T2 int64
}

type StringByteStruct struct {
	X string
	Y []byte
}

func TestDeserializeMapToMap_MapStringAny(t *testing.T) {
	kMarshalled := []byte{
		0x52, 0x00, 0x00, 0x00, // total document size

		0x02, // etype-string
		'A', 0x00,
		0x04, 0x00, 0x00, 0x00,
		'1', '2', '3', 0x00,

		0x03, // etype-doc
		'B', 0x00,
		0x1f, 0x00, 0x00, 0x00, // b's document size
		0x02, // etype-string
		'X', 0x00,
		0x06, 0x00, 0x00, 0x00,
		'h', 'e', 'l', 'l', 'o', 0x00,
		0x05, // etype-binary
		'Y', 0x00,
		0x05, 0x00, 0x00, 0x00,
		0x00,
		'w', 'o', 'r', 'l', 'd',
		0x00, // doc-term

		0x03, // etype-doc
		'C', 0x00,
		0x1d, 0x00, 0x00, 0x00,
		0x12, // etype int64
		'T', '1', 0x00,
		0xef, 0xbe, 0xad, 0xde, 0xde, 0xc0, 0xad, 0x0b,
		0x12, // type int64
		'T', '2', 0x00,
		0xbe, 0xba, 0xad, 0x0b, 0x00, 0x00, 0x00, 0x00,
		0x00, //doc-term

		0x00,
	}

	expected := map[string]any{
		"A": "123",
		"B": map[string]any{
			"X": "hello",
			"Y": []byte("world"),
		},
		"C": map[string]any{
			"T1": int64(0x0badc0dedeadbeef),
			"T2": int64(0x0badbabe),
		},
	}

	actual := make(map[string]any)

	if err := Unmarshal(kMarshalled, &actual); !assert.Nil(t, err) {
		return
	}

	if diff := deep.Equal(expected, actual); !assert.Nil(t, diff) {
		return
	}
}

func TestDeserializeMapToStruct(t *testing.T) {
	kMarshalled := []byte{
		0x45, 0x00, 0x00, 0x00, // total document size

		0x03, // etype-doc
		'A', 0x00,
		0x1d, 0x00, 0x00, 0x00,
		0x12, // etype int64
		'T', '1', 0x00,
		0xef, 0xcd, 0xab, 0x90, 0x78, 0x56, 0x34, 0x12,
		0x12, // type int64
		'T', '2', 0x00,
		0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, //doc-term

		0x03, // etype-doc
		'B', 0x00,
		0x1d, 0x00, 0x00, 0x00,
		0x12, // etype int64
		'T', '1', 0x00,
		0xde, 0xc0, 0xad, 0x0b, 0xde, 0xc0, 0xad, 0x0b,
		0x12, // type int64
		'T', '2', 0x00,
		0xbe, 0xba, 0xad, 0x0b, 0xbe, 0xba, 0xad, 0x0b,
		0x00, //doc-term

		0x00,
	}

	expected := map[string]TwoIntStruct{
		"A": {
			T1: 0x1234567890abcdef,
			T2: 0x0000000000000001,
		},
		"B": {
			T1: 0x0badc0de0badc0de,
			T2: 0x0badbabe0badbabe,
		},
	}

	actual := make(map[string]TwoIntStruct)

	if err := Unmarshal(kMarshalled, &actual); !assert.Nil(t, err) {
		return
	}

	if !assert.Nil(t, deep.Equal(expected, actual)) {
		return
	}
}

func TestDeserializeMapToMap_MapStringMapStringInt64(t *testing.T) {
	kMarshalled := []byte{
		0x45, 0x00, 0x00, 0x00, // total document size

		0x03, // etype-doc
		'A', 0x00,
		0x1d, 0x00, 0x00, 0x00,
		0x12, // etype int64
		'T', '1', 0x00,
		0xef, 0xcd, 0xab, 0x90, 0x78, 0x56, 0x34, 0x12,
		0x12, // type int64
		'T', '2', 0x00,
		0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, //doc-term

		0x03, // etype-doc
		'B', 0x00,
		0x1d, 0x00, 0x00, 0x00,
		0x12, // etype int64
		'T', '1', 0x00,
		0xde, 0xc0, 0xad, 0x0b, 0xde, 0xc0, 0xad, 0x0b,
		0x12, // type int64
		'T', '2', 0x00,
		0xbe, 0xba, 0xad, 0x0b, 0xbe, 0xba, 0xad, 0x0b,
		0x00, //doc-term

		0x00,
	}

	expected := map[string](map[string]int64){
		"A": {
			"T1": 0x1234567890abcdef,
			"T2": 0x0000000000000001,
		},
		"B": {
			"T1": 0x0badc0de0badc0de,
			"T2": 0x0badbabe0badbabe,
		},
	}

	actual := make(map[string](map[string]int64))

	if err := Unmarshal(kMarshalled, &actual); !assert.Nil(t, err) {
		return
	}

	if !assert.Nil(t, deep.Equal(expected, actual)) {
		return
	}
}

func TestDeserializeEmptyMap_MapStrAny(t *testing.T) {
	marshalled := []byte{
		0x05, 0x00, 0x00, 0x00, // Size
		0x00, // Terminator
	}

	expected := make(map[string]any)
	actual := make(map[string]any)

	if err := Unmarshal(marshalled, &actual); !assert.Nil(t, err) {
		return
	}

	if !assert.Equal(t, expected, actual) {
		return
	}
}

func TestDeserializeEmptyMap_MapStrInt(t *testing.T) {
	marshalled := []byte{
		0x05, 0x00, 0x00, 0x00, // Size
		0x00, // Terminator
	}

	expected := make(map[string]int)
	actual := make(map[string]int)

	if err := Unmarshal(marshalled, &actual); !assert.Nil(t, err) {
		return
	}

	if !assert.Equal(t, expected, actual) {
		return
	}
}

func TestDeserializeEmptyMap_MapStrAny_Nil(t *testing.T) {
	marshalled := []byte{
		0x05, 0x00, 0x00, 0x00, // Size
		0x00, // Terminator
	}

	expected := make(map[string]any)
	var actual map[string]any = nil

	if err := Unmarshal(marshalled, &actual); !assert.Nil(t, err) {
		return
	}

	if !assert.Equal(t, expected, actual) {
		return
	}
}

func TestDeserializeHello_MapStrAny(t *testing.T) {
	var err error

	marshalled := []byte{
		0x16, 0x00, 0x00, 0x00, // total document length
		0x02,                          // etype (string)
		'h', 'e', 'l', 'l', 'o', 0x00, // ename
		0x06, 0x00, 0x00, 0x00, // string-length (incl' nullterminator)
		'w', 'o', 'r', 'l', 'd', 0x00, // string-value
		0x00, // done
	}

	expected := map[string]any{
		"hello": "world",
	}

	actual := make(map[string]any)

	err = Unmarshal(marshalled, &actual)
	if !assert.Nil(t, err) {
		return
	}

	if !assert.Equal(t, expected, actual) {
		return
	}
}

func TestDeserializeHello_MapStrAny_Nil(t *testing.T) {
	var err error

	marshalled := []byte{
		0x16, 0x00, 0x00, 0x00, // total document length
		0x02,                          // etype (string)
		'h', 'e', 'l', 'l', 'o', 0x00, // ename
		0x06, 0x00, 0x00, 0x00, // string-length (incl' nullterminator)
		'w', 'o', 'r', 'l', 'd', 0x00, // string-value
		0x00, // done
	}

	expected := map[string]any{
		"hello": "world",
	}

	var actual map[string]any = nil

	err = Unmarshal(marshalled, &actual)
	if !assert.Nil(t, err) {
		return
	}

	if !assert.Equal(t, expected, actual) {
		return
	}
}

func TestDeserializeHello_MapStrStr(t *testing.T) {
	var err error

	marshalled := []byte{
		0x16, 0x00, 0x00, 0x00, // total document length
		0x02,                          // etype (string)
		'h', 'e', 'l', 'l', 'o', 0x00, // ename
		0x06, 0x00, 0x00, 0x00, // string-length (incl' nullterminator)
		'w', 'o', 'r', 'l', 'd', 0x00, // string-value
		0x00, // done
	}

	expected := map[string]string{
		"hello": "world",
	}

	actual := make(map[string]string)

	err = Unmarshal(marshalled, &actual)
	if !assert.Nil(t, err) {
		return
	}

	if !assert.Equal(t, expected, actual) {
		return
	}
}

func TestDeserializeVarious_MapStrAny(t *testing.T) {
	expectedTime, err := timelib.Parse(timelib.RFC3339, "2006-01-02T15:04:05Z")
	if err != nil {
		t.Error(err)
		return
	}

	kMarshalled := []byte{
		0xa4, 0x00, 0x00, 0x00, // total document length

		0x05, // etype (binary)
		'b', 'i', 'n', 0x00,
		0x05, 0x00, 0x00, 0x00, // buffer-length
		0x00, // subtype
		'w', 'o', 'r', 'l', 'd',

		0x01, // etype-double
		'd', 'o', 'u', 'b', 'l', 'e', 0x00,
		0x33, 0x33, 0x33, 0x33, 0x33, 0x33, 0x14, 0x40,

		0x08, // etype-boolean
		'f', 'a', 'l', 's', 'e', 0x00,
		0x00,

		0x12,
		'i', 'n', 't', 0x00,
		0xef, 0xbe, 0xad, 0xde, 0xde, 0xc0, 0xad, 0x0b,

		0x10, // etype-int32
		'i', 'n', 't', '3', '2', 0x00,
		0xbe, 0xba, 0xad, 0x0b,

		0x12, //etype-int64
		'i', 'n', 't', '6', '4', 0x00,
		0xef, 0xbe, 0xad, 0xde, 0xde, 0xc0, 0xad, 0x0b,

		0x12,
		'm', 'i', 'n', 'u', 's', 0x00,
		0xfb, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,

		0x10, // etype-int32
		'm', 'i', 'n', 'u', 's', '3', '2', 0x00,
		0xfb, 0xff, 0xff, 0xff,

		0x12, // etype-int64
		'm', 'i', 'n', 'u', 's', '6', '4', 0x00,
		0xfb, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff,

		0x02, // etype string
		's', 't', 'r', 0x00,
		0x06, 0x00, 0x00, 0x00, // string-length + 1
		'w', 'o', 'r', 'l', 'd', 0x00,

		0x09, // etype-time
		't', 'i', 'm', 'e', 0x00,
		0x88, 0x7e, 0xa5, 0x8b, 0x08, 0x01, 0x00, 0x00,

		0x08, //etype-bool
		't', 'r', 'u', 'e', 0x00,
		0x01,

		0x00, // done
	}

	expected := map[string]any{
		"bin":     []byte("world"),
		"double":  float64(5.05),
		"false":   false,
		"int":     int64(0x0badc0dedeadbeef),
		"int32":   int32(0x0badbabe),
		"int64":   int64(0x0badc0dedeadbeef),
		"minus":   int64(-5),
		"minus32": int32(-5),
		"minus64": int64(-5),
		"str":     "world",
		"time":    expectedTime,
		"true":    true,
	}

	actual := make(map[string]any)

	err = Unmarshal(kMarshalled, &actual)
	if !assert.Nil(t, err) {
		return
	}

	if !assert.Nil(t, deep.Equal(expected, actual)) {
		return
	}
}

func TestDeserialize_MapStrInt(t *testing.T) {

	kMarshalled := []byte{
		0x1b, 0x00, 0x00, 0x00,

		0x12, // type-int64
		'A', 0x00,
		0xef, 0xcd, 0xab, 0x90, 0x78, 0x56, 0x34, 0x12,

		0x12, // etype-int64
		'B', 0x00,
		0xef, 0xbe, 0xad, 0xde, 0xde, 0xc0, 0xad, 0x0b,

		0x00,
	}

	expected := map[string]int{
		"A": 0x1234567890abcdef,
		"B": 0x0badc0dedeadbeef,
	}

	var actual map[string]int

	err := Unmarshal(kMarshalled, &actual)
	if !assert.Nil(t, err) {
		return
	}

	if !assert.Nil(t, deep.Equal(expected, actual)) {
		return
	}
}

type EmbeddedArrayStructInt64 struct {
	BSON []int64
}

func TestDeserializeMapToSlice_MapStringAny(t *testing.T) {
	kMarshalled := []byte{
		0x35, 0x00, 0x00, 0x00, // total document size

		0x04, // etype-array
		'B', 'S', 'O', 'N', 0x00,
		0x2a, 0x00, 0x00, 0x00,

		0x02, // etype-string
		'0', 0x00,
		0x08, 0x00, 0x00, 0x00,
		'a', 'w', 'e', 's', 'o', 'm', 'e', 0x00,

		0x01, // etype-double
		'1', 0x00,
		0x33, 0x33, 0x33, 0x33, 0x33, 0x33, 0x14, 0x40,

		0x12, // etype int64
		'2', 0x00,
		0xc2, 0x07, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,

		0x00, // end slice

		0x00, // end doc
	}

	expected := map[string]any{
		"BSON": []any{"awesome", float64(5.05), int64(1986)},
	}

	actual := make(map[string]any)

	if err := Unmarshal(kMarshalled, &actual); !assert.Nil(t, err) {
		return
	}

	if !assert.Nil(t, deep.Equal(expected, actual)) {
		return
	}
}

func TestDeserializeMapToSlice_SliceInt64(t *testing.T) {
	kMarshalled := []byte{
		0x31, 0x00, 0x00, 0x00, // total document size

		0x04, // etype-array
		'B', 'S', 'O', 'N', 0x00,
		0x26, 0x00, 0x00, 0x00,

		0x12, // etype-string
		'0', 0x00,
		0xc2, 0x07, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,

		0x12, // etype-double
		'1', 0x00,
		0xef, 0xbe, 0xad, 0xde, 0xde, 0xc0, 0xad, 0x0b,

		0x12, // etype int64
		'2', 0x00,
		0xbe, 0xba, 0xad, 0x0b, 0x00, 0x00, 0x00, 0x00,

		0x00, // end slice

		0x00, // end doc
	}

	expected := map[string][]int64{
		"BSON": {1986, 0x0badc0dedeadbeef, 0x0badbabe},
	}

	actual := make(map[string][]int64)

	if err := Unmarshal(kMarshalled, &actual); !assert.Nil(t, err) {
		return
	}

	if !assert.Nil(t, deep.Equal(expected, actual)) {
		return
	}
}

func TestDeserializeMapToSlice_MapStringSliceAny(t *testing.T) {
	kMarshalled := []byte{
		0x35, 0x00, 0x00, 0x00, // total document size

		0x04, // etype-array
		'B', 'S', 'O', 'N', 0x00,
		0x2a, 0x00, 0x00, 0x00,

		0x02, // etype-string
		'0', 0x00,
		0x08, 0x00, 0x00, 0x00,
		'a', 'w', 'e', 's', 'o', 'm', 'e', 0x00,

		0x01, // etype-double
		'1', 0x00,
		0x33, 0x33, 0x33, 0x33, 0x33, 0x33, 0x14, 0x40,

		0x12, // etype int64
		'2', 0x00,
		0xc2, 0x07, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,

		0x00, // end slice

		0x00, // end doc
	}

	expected := map[string][]any{
		"BSON": {"awesome", float64(5.05), int64(1986)},
	}

	actual := make(map[string][]any)

	if err := Unmarshal(kMarshalled, &actual); !assert.Nil(t, err) {
		return
	}

	if !assert.Nil(t, deep.Equal(expected, actual)) {
		return
	}
}

func TestDeserializeSliceToStruct(t *testing.T) {
	kMarshalled := []byte{
		0x41, 0x00, 0x00, 0x00, // total doc size

		0x04, // etype-array
		'S', 0x00,
		0x39, 0x00, 0x00, 0x00, // slice size

		0x03, // etype-doc
		'0', 0x00,
		0x16, 0x00, 0x00, 0x00, // doc-size
		0x02, // etype-string
		'H', 'e', 'l', 'l', 'o', 0x00,
		0x06, 0x00, 0x00, 0x00,
		'w', 'o', 'r', 'l', 'd', 0x00,
		0x00, // doc-term

		0x03, // etype-doc
		'1', 0x00,
		0x18, 0x00, 0x00, 0x00, // doc-size
		0x02,
		'H', 'e', 'l', 'l', 'o', 0x00,
		0x08, 0x00, 0x00, 0x00,
		'm', 'y', 'k', 'o', 'n', 'o', 's', 0x00,
		0x00, // doc-term

		0x00, // end-slice

		0x00, // doc-end
	}

	expected := struct {
		S []HelloStruct
	}{
		[]HelloStruct{
			{
				"world",
			},
			{
				"mykonos",
			},
		},
	}

	actual := struct {
		S []HelloStruct
	}{}

	if err := Unmarshal(kMarshalled, &actual); !assert.Nil(t, err) {
		return
	}

	if !assert.Nil(t, deep.Equal(expected, actual)) {
		return
	}
}

func TestDeserializeSliceToMap(t *testing.T) {
	kMarshalled := []byte{
		0x41, 0x00, 0x00, 0x00, // total doc size

		0x04, // etype-array
		'S', 0x00,
		0x39, 0x00, 0x00, 0x00, // slice size

		0x03, // etype-doc
		'0', 0x00,
		0x16, 0x00, 0x00, 0x00, // doc-size
		0x02, // etype-string
		'h', 'e', 'l', 'l', 'o', 0x00,
		0x06, 0x00, 0x00, 0x00,
		'w', 'o', 'r', 'l', 'd', 0x00,
		0x00, // doc-term

		0x03, // etype-doc
		'1', 0x00,
		0x18, 0x00, 0x00, 0x00, // doc-size
		0x02,
		'h', 'e', 'l', 'l', 'o', 0x00,
		0x08, 0x00, 0x00, 0x00,
		'm', 'y', 'k', 'o', 'n', 'o', 's', 0x00,
		0x00, // doc-term

		0x00, // end-slice

		0x00, // doc-end
	}

	expected := struct {
		S []map[string]any
	}{
		[]map[string]any{
			{
				"hello": "world",
			},
			{
				"hello": "mykonos",
			},
		},
	}

	actual := struct {
		S []map[string]any
	}{}

	if err := Unmarshal(kMarshalled, &actual); !assert.Nil(t, err) {
		return
	}

	if !assert.Nil(t, deep.Equal(expected, actual)) {
		return
	}
}

func TestDeserializeSliceToSlice(t *testing.T) {
	kMarshalled := []byte{
		0x49, 0x00, 0x00, 0x00, // total doc length

		0x04, // etype-array
		'S', 0x00,
		0x41, 0x00, 0x00, 0x00, // array-length

		0x04,
		'0', 0x00,
		0x1b, 0x00, 0x00, 0x00,
		0x12,
		'0', 0x00,
		0xef, 0xcd, 0xab, 0x90, 0x78, 0x56, 0x34, 0x12,
		0x12,
		'1', 0x00,
		0x22, 0x11, 0xff, 0xee, 0xdd, 0xcc, 0xbb, 0x11,
		0x00,

		0x04,
		'1', 0x00,
		0x1b, 0x00, 0x00, 0x00,
		0x12,
		'0', 0x00,
		0xef, 0xbe, 0xad, 0xde, 0xde, 0xc0, 0xad, 0x0b,
		0x12,
		'1', 0x00,
		0xbe, 0xba, 0xad, 0x0b, 0x00, 0x00, 0x00, 0x00,
		0x00,

		0x00, // end array

		0x00, // end doc
	}

	expected := struct {
		S [][]int64
	}{
		[][]int64{
			{
				0x1234567890abcdef,
				0x11bbccddeeff1122,
			},
			{
				0x0badc0dedeadbeef,
				0x0badbabe,
			},
		},
	}

	actual := struct {
		S [][]int64
	}{}

	if err := Unmarshal(kMarshalled, &actual); !assert.Nil(t, err) {
		return
	}

	if !assert.Nil(t, deep.Equal(expected, actual)) {
		return
	}
}

func TestDeserializeSliceToSlice_SliceAny(t *testing.T) {
	kMarshalled := []byte{
		0x49, 0x00, 0x00, 0x00, // total doc length

		0x04, // etype-array
		'S', 0x00,
		0x41, 0x00, 0x00, 0x00, // array-length

		0x04,
		'0', 0x00,
		0x1b, 0x00, 0x00, 0x00,
		0x12,
		'0', 0x00,
		0xef, 0xcd, 0xab, 0x90, 0x78, 0x56, 0x34, 0x12,
		0x12,
		'1', 0x00,
		0x22, 0x11, 0xff, 0xee, 0xdd, 0xcc, 0xbb, 0x11,
		0x00,

		0x04,
		'1', 0x00,
		0x1b, 0x00, 0x00, 0x00,
		0x12,
		'0', 0x00,
		0xef, 0xbe, 0xad, 0xde, 0xde, 0xc0, 0xad, 0x0b,
		0x12,
		'1', 0x00,
		0xbe, 0xba, 0xad, 0x0b, 0x00, 0x00, 0x00, 0x00,
		0x00,

		0x00, // end array

		0x00, // end doc
	}

	expected := struct {
		S []any
	}{
		[]any{
			[]any{
				int64(0x1234567890abcdef),
				int64(0x11bbccddeeff1122),
			},
			[]any{
				int64(0x0badc0dedeadbeef),
				int64(0x0badbabe),
			},
		},
	}

	actual := struct {
		S []any
	}{}

	if err := Unmarshal(kMarshalled, &actual); !assert.Nil(t, err) {
		return
	}

	if !assert.Nil(t, deep.Equal(expected, actual)) {
		return
	}
}

func TestDeserializeIntArray(t *testing.T) {
	kMarshalled := []byte{
		0x23, 0x00, 0x00, 0x00, // doc-size

		0x04, // etype-array
		'S', 0x00,
		0x1b, 0x00, 0x00, 0x00, // array-size

		0x12, // type-int64
		'0', 0x00,
		0xef, 0xcd, 0xab, 0x90, 0x78, 0x56, 0x34, 0x12,

		0x12, // etype-int64
		'1', 0x00,
		0xef, 0xbe, 0xad, 0xde, 0xde, 0xc0, 0xad, 0x0b,

		0x00, // end-array

		0x00, // end-doc
	}

	expected := struct {
		S []int
	}{
		[]int{0x1234567890abcdef, 0x0badc0dedeadbeef},
	}

	actual := struct {
		S []int
	}{}

	if err := Unmarshal(kMarshalled, &actual); !assert.Nil(t, err) {
		return
	}

	if !assert.Nil(t, deep.Equal(expected, actual)) {
		return
	}
}
