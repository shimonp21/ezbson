// Package ezbson is a minimalistic reflection-based implementation of [BSON] encoding and docoding.
// Usage is similar to [encoding/json]'s Marshal/Unmarshal.
//
// [BSON]: https://bsonspec.org
// [encoding/json]: https://pkg.go.dev/encoding/json
package ezbson

import (
	"bytes"
	binlib "encoding/binary"
	"fmt"
	"math"
	"reflect"
	"sort"
	"strconv"
	"time"
)

// Serialized documents always have their keys sorted.
// Everything is little-endian.

type etype byte

const (
	kEtypeDone           etype = 0x00
	kEtypeDouble         etype = 0x01
	kEtypeString         etype = 0x02
	kEtypeDocument       etype = 0x03
	kEtypeArray          etype = 0x04
	kEtypeBinary         etype = 0x05
	kEtypeDeprecated6    etype = 0x06
	kEtypeObjectId       etype = 0x07
	kEtypeBoolean        etype = 0x08
	kEtypeUtcDatetime    etype = 0x09
	kEtypeNull           etype = 0x0a
	kEtypeRegex          etype = 0x0b
	kEtypeDeprecated12   etype = 0x0c
	kEtypeJavascriptCode etype = 0x0d
	kEtypeDeprecated14   etype = 0x0e
	kEtypeDeprecated15   etype = 0x0f
	kEtypeInt32          etype = 0x10
	kEtypeMongoTimestamp etype = 0x11
	kEtypeInt64          etype = 0x12
	kEtypeDecimal128     etype = 0x13
	kEtypeMinKey         etype = 0xff
	kEtypeMaxKey         etype = 0x7f
)

const (
	kBinarySubtype  byte = 0
	kNullTerminator byte = 0
)

func validate64bit() error {
	var i int
	if reflect.TypeOf(i).Size() != kInt64Size {
		return fmt.Errorf("ezbson only supports 64bit architecture")
	}

	return nil
}

func getEtype(val any) (etype, error) {
	rtype := reflect.TypeOf(val)
	rkind := rtype.Kind()

	if rkind == reflect.Pointer {
		return getEtype(reflect.ValueOf(val).Elem().Interface())
	}

	switch val.(type) {
	case float64:
		return kEtypeDouble, nil
	case string:
		return kEtypeString, nil
	case []byte:
		return kEtypeBinary, nil
	case bool:
		return kEtypeBoolean, nil
	case time.Time:
		return kEtypeUtcDatetime, nil
	case int32:
		return kEtypeInt32, nil
	case int:
		return kEtypeInt64, nil
	case int64:
		return kEtypeInt64, nil
	default:
		break
	}

	switch rkind {
	case reflect.Map:
		return kEtypeDocument, nil
	case reflect.Struct:
		return kEtypeDocument, nil
	case reflect.Slice:
		return kEtypeArray, nil
	default:
		return 0, fmt.Errorf("unsupported type %T", val)
	}
}

func appendAny(buffer []byte, val_any any) ([]byte, error) {
	var err error

	valRtype := reflect.TypeOf(val_any)
	valRkind := valRtype.Kind()

	if valRkind == reflect.Pointer {
		return appendAny(buffer, reflect.ValueOf(val_any).Elem().Interface()) // .Interface() copies
	}

	switch val := val_any.(type) {
	case []byte:
		if len(val) > math.MaxInt32 {
			return buffer, fmt.Errorf("byte slice too big (%v)", len(val))
		}
		buffer, err = appendInt32(buffer, int32(len(val)))
		if err != nil {
			return buffer, err
		}
		buffer = append(buffer, kBinarySubtype)
		buffer = append(buffer, val...)
	case string:
		if len(val)+1 > math.MaxInt32 {
			return buffer, fmt.Errorf("string too long (%v)", len(val))
		}
		buffer, err = appendInt32(buffer, int32(len(val)+1))
		if err != nil {
			return buffer, err
		}
		buffer = append(buffer, []byte(val)...)
		buffer = append(buffer, kNullTerminator)

	case float64:
		buffer, err = appendFloat64(buffer, val)
	case bool:
		buffer, err = appendBoolean(buffer, val)
	case time.Time:
		val_int64 := val.UTC().UnixMilli()
		buffer, err = appendInt64(buffer, val_int64)
	case int32:
		buffer, err = appendInt32(buffer, val)
	case int:
		buffer, err = appendInt64(buffer, int64(val))
	case int64:
		buffer, err = appendInt64(buffer, val)
	default:
		buffer, err = appendOther(buffer, val)
	}

	if err != nil {
		return buffer, err
	}

	return buffer, nil
}

func appendMap(buffer []byte, doc map[string]any) ([]byte, error) {
	var kSizePlaceholder int32

	startPos := len(buffer)
	buffer, err := appendInt32(buffer, kSizePlaceholder)
	if err != nil {
		return buffer, err
	}

	for _, key := range sortedKeys(doc) {
		if err = validateEname(key); err != nil {
			return buffer, err
		}

		val := doc[key]
		et, err := getEtype(val)
		if err != nil {
			return buffer, fmt.Errorf("key %v: %w", key, err)
		}

		buffer = append(buffer, byte(et))

		buffer = append(buffer, []byte(key)...)
		buffer = append(buffer, kNullTerminator)

		buffer, err = appendAny(buffer, val)
		if err != nil {
			return buffer, fmt.Errorf("key %v: %w", key, err)
		}
	}
	buffer = append(buffer, byte(kEtypeDone))

	endPos := len(buffer)
	totalSize := endPos - startPos

	if totalSize < 0 || totalSize > math.MaxInt32 {
		return nil, fmt.Errorf("size of marshalled buffer too big (%v)", totalSize)
	}

	totalSize_bin, err := convertInt32ToBytes(int32(totalSize))
	if err != nil {
		return buffer, err
	}
	copy(buffer[startPos:], totalSize_bin)

	return buffer, nil
}

// handles maps, slices, and structs (the types that require reflection)
func appendOther(buffer []byte, val_any any) ([]byte, error) {
	valType := reflect.TypeOf(val_any)
	valKind := valType.Kind()

	var err error
	switch valKind {
	case reflect.Map:
		mapKeyType := valType.Key()
		mapKeyKind := mapKeyType.Kind()

		if mapKeyKind != reflect.String {
			return buffer, fmt.Errorf("only map[string]... is supported")
		}
		doc := convertReflectMapToMapStringAny(reflect.ValueOf(val_any))

		if buffer, err = appendMap(buffer, doc); err != nil {
			return buffer, err
		}

	case reflect.Slice:
		doc := convertReflectSliceToMapStringAny(reflect.ValueOf(val_any))

		if buffer, err = appendMap(buffer, doc); err != nil {
			return buffer, err
		}

	case reflect.Struct:
		doc := convertReflectStructToMapStringAny(reflect.ValueOf(val_any))

		if buffer, err = appendMap(buffer, doc); err != nil {
			return buffer, err
		}

	default:
		return buffer, fmt.Errorf("unable to serialize %T", val_any)
	}

	return buffer, nil
}

func convertReflectStructToMapStringAny(v reflect.Value) map[string]any {
	result := make(map[string]any)

	for i := 0; i < v.NumField(); i++ {
		field := v.Field(i)
		fieldName := v.Type().Field(i).Name
		result[fieldName] = field.Interface()
	}

	return result
}

// Marhsal recursively marshals a golang map[string]... or a golang struct into BSON format.
//
// The BSON spec does not allow arrays (slices) as top-level documents, but they are supported when nested in a map[string]... or a struct.
//
// Marshal automatically dereferences pointers (so a *int64 will still be serialized into the BSON int64 type).
//
// See the examples at the package documentation for example usage, and https://bsonspec.org for more info on the BSON format.
//
// Below are the supported types that Marshal can convert.
//
//	// +----------------+------------------+
//	// | golang type    | bson type        |
//	// +----------------+------------------+
//	// | float64        | double (1)       |
//	// | string         | string (2)       |
//	// | map[string]... | document (3)     |
//	// | struct         | document (3)     |
//	// | []...          | array (4)        |
//	// | []byte         | binary (5)       |
//	// | bool           | boolean (8)      |
//	// | time.Time      | utc datetime (9) |
//	// | int32          | int32 (16)       |
//	// | int64          | int64 (18)       |
//	// | int            | int64 (18)       |
//	// +----------------+------------------+
//
// Limitations:
//   - due to the way reflect works, all structs that are being marshalled must only contain exported (uppercase) fields.
//   - as of right now, only 64 bit architectures are supported.
func Marshal(document any) ([]byte, error) {
	if err := validate64bit(); err != nil {
		return nil, fmt.Errorf("ezbson.Marshal: %w", err)
	}

	documentRtype := reflect.TypeOf(document)
	documentRkind := documentRtype.Kind()

	if documentRkind == reflect.Pointer {
		return Marshal(reflect.ValueOf(document).Elem().Interface()) // .Interface() copies
	}

	if documentRkind != reflect.Map && documentRkind != reflect.Struct {
		return nil, fmt.Errorf("ezbson.Marshal: at the top-level, only maps and structs are supported")
	}

	buffer := make([]byte, 0)
	buffer, err := appendAny(buffer, document)
	if err != nil {
		return nil, fmt.Errorf("ezbson.Marshal: %w", err)
	}
	return buffer, err
}

// Receives a map[string]...
// And returns a map[string]any
func convertReflectMapToMapStringAny(m reflect.Value) map[string]any {
	result := make(map[string]any)

	for _, k := range m.MapKeys() {
		result[k.String()] = m.MapIndex(k).Interface()
	}

	return result
}

// e.g. [100, "hello", 300] -> {"0": 100, "1": "hello", "2": 300}
func convertReflectSliceToMapStringAny(s reflect.Value) map[string]any {
	m := make(map[string]any)
	for i := 0; i < s.Len(); i++ {
		m[strconv.Itoa(i)] = s.Index(i).Interface()
	}

	return m
}

func validateEname(ename string) error {
	for i := 0; i < len(ename); i++ {
		if ename[i] == 0 {
			return fmt.Errorf("null bytes not allowed in enames (ename=%v)", ename)
		}
	}
	return nil
}

func appendInt32(buffer []byte, val int32) ([]byte, error) {
	val_bin, err := convertInt32ToBytes(val)
	if err != nil {
		return nil, err
	}

	buffer = append(buffer, val_bin...)
	return buffer, nil
}

func appendInt64(buffer []byte, val int64) ([]byte, error) {
	val_bin, err := convertInt64ToBytes(val)
	if err != nil {
		return nil, err
	}

	buffer = append(buffer, val_bin...)
	return buffer, nil
}

func appendFloat64(buffer []byte, val float64) ([]byte, error) {
	val_bin, err := convertFloat64ToBytes(val)
	if err != nil {
		return nil, err
	}

	buffer = append(buffer, val_bin...)
	return buffer, nil
}

func convertInt32ToBytes(val int32) ([]byte, error) {
	buffer := &bytes.Buffer{}

	if err := binlib.Write(buffer, binlib.LittleEndian, val); err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

func convertInt64ToBytes(val int64) ([]byte, error) {
	buffer := &bytes.Buffer{}

	if err := binlib.Write(buffer, binlib.LittleEndian, val); err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

func convertFloat64ToBytes(val float64) ([]byte, error) {
	buffer := &bytes.Buffer{}

	if err := binlib.Write(buffer, binlib.LittleEndian, val); err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

func sortedKeys(m map[string]any) []string {
	keys := make([]string, 0)
	for key := range m {
		keys = append(keys, key)
	}

	sort.Strings(keys)
	return keys
}

func appendBoolean(buffer []byte, val bool) ([]byte, error) {
	if val {
		buffer = append(buffer, 1)
	} else {
		buffer = append(buffer, 0)
	}

	return buffer, nil
}
