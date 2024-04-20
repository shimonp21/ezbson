package ezbson

import (
	bytelib "bytes"
	binlib "encoding/binary"
	"fmt"
	"reflect"
	timelib "time"
)

const (
	kEtypeSize   = 1
	kSubtypeSize = 1

	kInt8Size    = 1
	kInt32Size   = 4
	kInt64Size   = 8
	kFloat64Size = 8
)

// Unmarshal deserializes a BSON document into a struct or map[string]...
//
// See the examples at the package documentation for example usage, and https://bsonspec.org for more info on the BSON format.
//
// ptr should point to where you would like the data to be serialized to.
//
// The BSON spec does not allow arrays (slices) as top-level documents, but they are supported when nested in a map[string]... or a struct.
//
// Unamarshal uses reflection-information from ptr to decide what golang-type to use for each BSON type.
// for instance, deserializing `{"A": [1, 2, 3], "B": [4, 5, 6]}` can be deserialized
// into either:
//   - map[string]any (each element will be of type []any).
//   - map[string][]int64 (each element will be of type []int64).
//   - map[string][]int32 (each element will be of type []int32).
//   - map[string][]any (each element will be of type []any)
//   - struct { A []int64; B []any }
//
// Below are the supported types that Unmarshal can convert:
//
//	// +----------------------+---------------------------+
//	// | bson type            | golang type               |
//	// +----------------------+---------------------------+
//	// | double (1)           | float64                   |
//	// | string (2)           | string                    |
//	// | document (3)         | struct, or map[string]... |
//	// | array (4)            | []...                     |
//	// | binary (5)           | []byte                    |
//	// | deprecated (6)       | <NOT IMPLEMENTED>         |
//	// | objectid (7)         | <NOT IMPLEMENTED>         |
//	// | boolean (8)          | bool                      |
//	// | UTC datetime (9)     | time.Time                 |
//	// | null (10)            | <NOT IMPLEMENTED>         |
//	// | regex (11)           | <NOT IMPLEMENTED>         |
//	// | deprecated (12)      | <NOT IMPLEMENTED>         |
//	// | javascript code (13) | <NOT IMPLEMENTED>         |
//	// | symbol (14)          | <NOT IMPLEMENTED>         |
//	// | deprecated (15)      | <NOT IMPLEMENTED>         |
//	// | int32 (16)           | int32                     |
//	// | mongo timestamp (17) | <NOT IMPLEMENTED>         |
//	// | int64 (18)           | int64 or int              |
//	// | decimal128 (19)      | <NOT IMPLEMENTED>         |
//	// | min_key (-1)         | <NOT IMPLEMENTED>         |
//	// | max_key (-1)         | <NOT IMPLEMENTED>         |
//	// +----------------------+---------------------------+
//
// Limitations:
//   - due to the way reflect works, all structs that are being marshalled must only contain exported (uppercase) fields.
//   - as of right now, only 64 bit architectures are supported.
func Unmarshal(marshalled []byte, ptr any) error {
	if err := validate64bit(); err != nil {
		return fmt.Errorf("ezbson.Unmarshal: %w", err)
	}

	buffer := bytelib.NewBuffer(marshalled)

	if reflect.TypeOf(ptr).Kind() != reflect.Ptr {
		return fmt.Errorf("ezbson.Unmarshal: ptr must be a pointer")
	}
	valRtype := reflect.TypeOf(ptr).Elem()
	valRkind := valRtype.Kind()

	var numread int
	var err error

	switch valRkind {
	case reflect.Struct:
		numread, err = readStruct(buffer, ptr)
	case reflect.Map:
		numread, err = readMap(buffer, ptr)
	default:
		return fmt.Errorf("ezbson.Unmarshal: only structs or maps are supported at the top level")
	}

	if err != nil {
		return fmt.Errorf("ezbson.Unmarshal: %w", err)
	}
	if numread != len(marshalled) {
		return fmt.Errorf(
			"ezbson.Unmarshal: did not consume all bytes (%v) and not (%v)",
			numread, len(marshalled))
	}

	return nil
}

// Returns the amount of bytes read (only valid if error is nil)
func readStruct(buffer *bytelib.Buffer, structptr any) (numread int, err error) {
	var expectedSize int32
	var actualSize int

	if numread, err = readInt32(buffer, &expectedSize); err != nil {
		return 0, err
	}
	actualSize += numread

	struct_rvalue := reflect.Indirect(reflect.ValueOf(structptr))

	for {
		var et etype
		if numread, err = readEtype(buffer, &et); err != nil {
			return 0, err
		}
		actualSize += numread

		if et == kEtypeDone {
			if actualSize != int(expectedSize) {
				return 0, fmt.Errorf("expected size (%v) does not match actual size (%v)", expectedSize, actualSize)
			}
			return actualSize, nil
		}

		var ename string
		if numread, err = readEname(buffer, &ename); err != nil {
			return 0, err
		}
		actualSize += numread

		field_rvalue := struct_rvalue.FieldByName(ename)
		if field_rvalue == (reflect.Value{}) {
			return 0, fmt.Errorf("field {%v} not found", ename)
		}

		field_rtype := field_rvalue.Type()
		if err = validateEtypeCanBeDeserializeToRtype(et, field_rtype); err != nil {
			return 0, fmt.Errorf("field {%v}: %w", ename, err)
		}

		fieldptr_rvalue := field_rvalue.Addr()
		fieldptr_any := fieldptr_rvalue.Interface()

		if numread, err = readEvalue(buffer, fieldptr_any, et); err != nil {
			return 0, fmt.Errorf("field {%v}: %w", ename, err)
		}
		actualSize += numread
	}
}

// https://stackoverflow.com/a/18316266
func emptyInterfaceRtype() reflect.Type {
	var s = make([]any, 0)
	return reflect.TypeOf(s).Elem()
}

func validateEtypeCanBeDeserializeToRtype(et etype, rtype reflect.Type) error {
	var rkind = rtype.Kind()

	if rtype == emptyInterfaceRtype() { // We can always deserialize into 'any'
		return nil
	}

	switch et {
	case kEtypeDouble:
		if rkind != reflect.Float64 {
			return fmt.Errorf("cannot convert double (etype %v) to %v", et, rtype)
		}
	case kEtypeString:
		if rkind != reflect.String {
			return fmt.Errorf("cannot convert string (etype %v) to %v", et, rtype)
		}
	case kEtypeBinary:
		if rtype != reflect.TypeOf(make([]byte, 0)) {
			return fmt.Errorf("cannot convert binary (etype %v) to %v", et, rtype)
		}
	case kEtypeBoolean:
		if rkind != reflect.Bool {
			return fmt.Errorf("cannot convert boolean (etype %v) to %v", et, rtype)
		}
	case kEtypeUtcDatetime:
		if rtype != reflect.TypeOf(timelib.Time{}) {
			return fmt.Errorf("cannot convert UtcDatetime (etype %v) to %v", et, rtype)
		}
	case kEtypeInt32:
		if rkind != reflect.Int32 {
			return fmt.Errorf("cannot convert int32 (etype %v) to %v", et, rtype)
		}
	case kEtypeInt64:
		if rkind != reflect.Int64 && rkind != reflect.Int {
			return fmt.Errorf("cannot convert int64 (etype %v) to %v", et, rtype)
		}
	case kEtypeArray:
		if rkind != reflect.Slice {
			return fmt.Errorf("cannot convert Array (etype %v) to %v", et, rtype)
		}
	case kEtypeDocument:
		if rkind != reflect.Struct && rkind != reflect.Map {
			return fmt.Errorf("cannot convert Document (etype %v) to %v", et, rtype)
		}
	}

	return nil
}

func readMap(buffer *bytelib.Buffer, mapptr any) (numread int, err error) {
	var expectedSize int32
	var actualSize int

	mapRtype := reflect.TypeOf(mapptr).Elem()
	mapKeyRtype := mapRtype.Key()
	mapKeyRkind := mapKeyRtype.Kind()
	mapElemRtype := mapRtype.Elem()
	mapElemRkind := mapElemRtype.Kind()

	if mapKeyRkind != reflect.String {
		return 0, fmt.Errorf("only map[string]... is supported")
	}

	if numread, err = readInt32(buffer, &expectedSize); err != nil {
		return 0, err
	}
	actualSize += numread

	mapRvalue := reflect.ValueOf(mapptr).Elem()
	mapRvalue.Set(reflect.MakeMap(mapRtype)) // This changes a nil-map to an empty map (important for 'SetMapIndex' later).

	for {
		var et etype
		if numread, err = readEtype(buffer, &et); err != nil {
			return 0, err
		}
		actualSize += numread

		if et == kEtypeDone {
			if actualSize != int(expectedSize) {
				return 0, fmt.Errorf("expected size (%v) does not match actual size (%v)", expectedSize, actualSize)
			}
			return actualSize, nil
		}

		var ename string
		if numread, err = readEname(buffer, &ename); err != nil {
			return 0, err
		}
		actualSize += numread

		if err = validateEtypeCanBeDeserializeToRtype(et, mapElemRtype); err != nil {
			return 0, fmt.Errorf("field {%v}: %w", ename, err)
		}

		// map values aren't addressable in golang, so we need to read into a temporary variable.
		// tmpptr is a pointer to a concrete-type (stored in an 'any' interface)

		var tmpptr any
		switch et {
		case kEtypeDouble:
			var tmp float64
			tmpptr = &tmp
		case kEtypeString:
			var tmp string
			tmpptr = &tmp
		case kEtypeBinary:
			var tmp []byte
			tmpptr = &tmp
		case kEtypeBoolean:
			var tmp bool
			tmpptr = &tmp
		case kEtypeUtcDatetime:
			var tmp timelib.Time
			tmpptr = &tmp
		case kEtypeInt32:
			var tmp int32
			tmpptr = &tmp
		case kEtypeInt64:
			if mapElemRkind == reflect.Int {
				var tmp int
				tmpptr = &tmp
			} else {
				var tmp int64
				tmpptr = &tmp
			}
		case kEtypeDocument:
			switch mapElemRkind {
			case reflect.Struct:
				var tmp = reflect.New(mapElemRtype)
				tmpptr = tmp.Interface()
			case reflect.Map:
				tmpmap := reflect.MakeMap(mapElemRtype) // https://stackoverflow.com/a/25386460
				tmpptr_rvalue := reflect.New(mapElemRtype)
				tmpptr_rvalue.Elem().Set(tmpmap)
				tmpptr = tmpptr_rvalue.Interface()

			default:
				_, ok := mapptr.(*map[string]any)
				if !ok {
					return 0, fmt.Errorf("field %v: cannot deserialize a document into {%v}", ename, mapElemRtype)
				}

				// Deserialize the element into a map[string]any
				var tmp = make(map[string]any)
				tmpptr = &tmp
			}
		case kEtypeArray:
			switch mapElemRkind {
			case reflect.Slice:
				tmpslice := reflect.MakeSlice(mapElemRtype, 0, 0) // https://stackoverflow.com/a/25386460
				tmpptr_rvalue := reflect.New(mapElemRtype)
				tmpptr_rvalue.Elem().Set(tmpslice)
				tmpptr = tmpptr_rvalue.Interface()
			default:
				_, ok := mapptr.(*map[string]any)
				if !ok {
					return 0, fmt.Errorf("field %v: cannot deserialize a slice into {%v}", ename, mapElemRtype)
				}

				var tmp = make([]any, 0)
				tmpptr = &tmp
			}
		default:
			return 0, fmt.Errorf("field %v: unsupported etype %v", ename, et)
		}

		if numread, err = readEvalue(buffer, tmpptr, et); err != nil {
			return 0, fmt.Errorf("field {%v}: %w", ename, err)
		}
		actualSize += numread

		tmpptr_rvalue := reflect.ValueOf(tmpptr)

		mapRvalue.SetMapIndex(reflect.ValueOf(ename), tmpptr_rvalue.Elem())
	}
}

// a struct in bson is a sequence of [etype ename evalue].
// This function receives a generic pointer and an etype, and reads the evalue into it.
func readEvalue(buffer *bytelib.Buffer, ptr_any any, et etype) (numread int, err error) {
	switch et {
	case kEtypeDouble:
		ptr := ptr_any.(*float64)

		if numread, err = readFloat64(buffer, ptr); err != nil {
			return 0, err
		}

	case kEtypeString:
		ptr := ptr_any.(*string)

		if numread, err = readEstring(buffer, ptr); err != nil {
			return 0, err
		}

	case kEtypeBinary:
		ptr := ptr_any.(*[]byte)

		if numread, err = readEbinary(buffer, ptr); err != nil {
			return 0, err
		}

	case kEtypeBoolean:
		ptr := ptr_any.(*bool)
		if numread, err = readBoolean(buffer, ptr); err != nil {
			return 0, err
		}

	case kEtypeUtcDatetime:
		ptr := ptr_any.(*timelib.Time)

		var millisecFromEpoch int64
		if numread, err = readInt64(buffer, &millisecFromEpoch); err != nil {
			return 0, err
		}

		*ptr = timelib.Unix(
			millisecFromEpoch/1e3, (millisecFromEpoch%1e3)*1e6).UTC()

	case kEtypeInt32:
		ptr := ptr_any.(*int32)

		if numread, err = readInt32(buffer, ptr); err != nil {
			return 0, err
		}

	case kEtypeInt64:
		switch ptr := ptr_any.(type) {
		case *int64:
			if numread, err = readInt64(buffer, ptr); err != nil {
				return 0, err
			}
		case *int:
			if numread, err = readInt(buffer, ptr); err != nil {
				return 0, err
			}
		default:
			return 0, fmt.Errorf("cannot convert etype int64 to %t", ptr_any)

		}

	case kEtypeDocument:
		valRtype := reflect.TypeOf(ptr_any).Elem()
		valRkind := valRtype.Kind()

		switch valRkind {
		case reflect.Struct:
			numread, err = readStruct(buffer, ptr_any)
		case reflect.Map:
			numread, err = readMap(buffer, ptr_any)
		default:
			return 0, fmt.Errorf("unsupported type %v", valRtype)
		}
		return numread, err

	case kEtypeArray:
		numread, err = readArray(buffer, ptr_any)
	default:
		return 0, fmt.Errorf("unsupported etype %v", et)
	}

	return numread, err
}

// Mostly a copy of readMap
func readArray(buffer *bytelib.Buffer, arrptr any) (numread int, err error) {
	var expectedSize int32
	var actualSize int

	arrRtype := reflect.TypeOf(arrptr).Elem()
	arrElemRtype := arrRtype.Elem()
	arrElemRkind := arrElemRtype.Kind()

	if numread, err = readInt32(buffer, &expectedSize); err != nil {
		return 0, err
	}
	actualSize += numread

	arrRvalue := reflect.ValueOf(arrptr).Elem()
	arrRvalue.Set(reflect.MakeSlice(arrRtype, 0, 0)) // This changes a nil-slice to an empty slice (important for 'SetMapIndex' later).

	for {
		var et etype
		if numread, err = readEtype(buffer, &et); err != nil {
			return 0, err
		}
		actualSize += numread

		if et == kEtypeDone {
			if actualSize != int(expectedSize) {
				return 0, fmt.Errorf("expected size (%v) does not match actual size (%v)", expectedSize, actualSize)
			}
			return actualSize, nil
		}

		var ename string
		if numread, err = readEname(buffer, &ename); err != nil {
			return 0, err
		}
		actualSize += numread

		if err = validateEtypeCanBeDeserializeToRtype(et, arrElemRtype); err != nil {
			return 0, fmt.Errorf("field {%v}: %w", ename, err)
		}

		var tmpptr any
		switch et {
		case kEtypeDouble:
			var tmp float64
			tmpptr = &tmp
		case kEtypeString:
			var tmp string
			tmpptr = &tmp
		case kEtypeBinary:
			var tmp []byte
			tmpptr = &tmp
		case kEtypeBoolean:
			var tmp bool
			tmpptr = &tmp
		case kEtypeUtcDatetime:
			var tmp timelib.Time
			tmpptr = &tmp
		case kEtypeInt32:
			var tmp int32
			tmpptr = &tmp
		case kEtypeInt64:
			if arrElemRkind == reflect.Int {
				var tmp int
				tmpptr = &tmp
			} else {
				var tmp int64
				tmpptr = &tmp
			}
		case kEtypeDocument:
			switch arrElemRkind {
			case reflect.Struct:
				var tmp = reflect.New(arrElemRtype)
				tmpptr = tmp.Interface()
			case reflect.Map:
				tmpmap := reflect.MakeMap(arrElemRtype) // https://stackoverflow.com/a/25386460
				tmpptr_rvalue := reflect.New(arrElemRtype)
				tmpptr_rvalue.Elem().Set(tmpmap)
				tmpptr = tmpptr_rvalue.Interface()

			default:
				_, ok := arrptr.(*[]any)
				if !ok {
					return 0, fmt.Errorf("field %v: cannot deserialize a document into {%v}", ename, arrElemRtype)
				}

				// Deserialize the element into a map[string]any
				var tmp = make(map[string]any)
				tmpptr = &tmp
			}
		case kEtypeArray:
			switch arrElemRkind {
			case reflect.Slice:
				tmpslice := reflect.MakeSlice(arrElemRtype, 0, 0) // https://stackoverflow.com/a/25386460
				tmpptr_rvalue := reflect.New(arrElemRtype)
				tmpptr_rvalue.Elem().Set(tmpslice)
				tmpptr = tmpptr_rvalue.Interface()
			default:
				_, ok := arrptr.(*[]any)
				if !ok {
					return 0, fmt.Errorf("field {%v}: cannot deserialize a slice into {%v}", ename, arrElemRtype)
				}

				// Deserialize the element into a []any
				var tmp = make([]any, 0)
				tmpptr = &tmp
			}
		default:
			return 0, fmt.Errorf("unsupported etype %v", et)
		}

		if numread, err = readEvalue(buffer, tmpptr, et); err != nil {
			return 0, fmt.Errorf("field {%v}: %w", ename, err)
		}
		actualSize += numread

		tmpptr_rvalue := reflect.ValueOf(tmpptr)
		arrRvalue.Set(reflect.Append(arrRvalue, tmpptr_rvalue.Elem()))
	}
}

func readInt32(buffer *bytelib.Buffer, val *int32) (numread int, err error) {
	return kInt32Size, binlib.Read(buffer, binlib.LittleEndian, val)
}

func readEtype(buffer *bytelib.Buffer, val *etype) (numread int, err error) {
	b, err := buffer.ReadByte()
	if err != nil {
		return 0, err
	}

	*val = etype(b)
	return kEtypeSize, nil
}

func readEname(buffer *bytelib.Buffer, val *string) (numread int, err error) {
	ename := make([]byte, 0)

	for {
		b, err := buffer.ReadByte()
		if err != nil {
			return 0, err
		}

		if b == 0 {
			*val = string(ename)
			return len(ename) + 1, nil
		}

		ename = append(ename, b)
	}
}

// reads an int64 into val.
func readInt(buffer *bytelib.Buffer, val *int) (numread int, err error) {
	var tmp int64
	numread, err = readInt64(buffer, &tmp)
	if err != nil {
		return numread, err
	}

	*val = int(tmp)
	return numread, nil
}

func readInt64(buffer *bytelib.Buffer, val *int64) (numread int, err error) {
	return kInt64Size, binlib.Read(buffer, binlib.LittleEndian, val)
}

func readFloat64(buffer *bytelib.Buffer, val *float64) (numread int, err error) {
	return kFloat64Size, binlib.Read(buffer, binlib.LittleEndian, val)
}

func readEstring(buffer *bytelib.Buffer, val *string) (numread int, err error) {
	var sizeWithNullterm int32
	if _, err := readInt32(buffer, &sizeWithNullterm); err != nil {
		return 0, err
	}

	str := make([]byte, sizeWithNullterm-1)
	if numread, err = buffer.Read(str); err != nil {
		return 0, err
	}
	if numread != len(str) {
		return 0, fmt.Errorf("expected to read %v bytes, but read %v", len(str), numread)
	}

	nullterm, err := buffer.ReadByte()
	if err != nil {
		return 0, err
	}
	if nullterm != 0 {
		return 0, fmt.Errorf("expected null terminator")
	}

	*val = string(str)
	return int(sizeWithNullterm) + kInt32Size, nil
}

func readEbinary(buffer *bytelib.Buffer, val *[]byte) (numread int, err error) {
	var size int32

	if _, err := readInt32(buffer, &size); err != nil {
		return 0, err
	}

	_, err = buffer.ReadByte() // Consume subtype
	if err != nil {
		return 0, err
	}

	bin := make([]byte, size)
	if numread, err = buffer.Read(bin); err != nil {
		return 0, err
	}
	if numread != len(bin) {
		return 0, fmt.Errorf("expected to read %v bytes, but read %v", len(bin), numread)
	}

	*val = bin
	return int(size) + kInt32Size + kSubtypeSize, nil
}

func readBoolean(buffer *bytelib.Buffer, val *bool) (numread int, err error) {
	b, err := buffer.ReadByte()
	if err != nil {
		return 0, fmt.Errorf("readBoolean: %w", err)
	}

	if b == 0 {
		*val = false
		return kInt8Size, nil
	} else if b == 1 {
		*val = true
		return kInt8Size, nil
	}

	return kInt8Size, fmt.Errorf("readBoolean: unexpected value read (%v)", b)
}
