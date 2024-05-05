# ezbson

Package ezbson is a minimalistic reflection-based implementation of [BSON](https://bsonspec.org) encoding and decoding.

While similar to [JSON](https://json.org), BSON can be more fitting in use-cases where binary data needs to be serialized (where usually, JSON would have to use base64/urlencoding/backslash-escaping/...)

The API is meant to be similar to [encoding/json](https://pkg.go.dev/encoding/json)'s  Marshal and Unmarshal.

## Installing

```bash
go get github.com/shimonp21/ezbson
```

## Usage

```go
type ExampleStruct struct {
	BSON []any
}

func main() {
	// This is the example from https://bsonspec.org/faq.html
	example := ExampleStruct{
		BSON: []any{"awesome", 5.05, int32(1986)},
	}

	bson, err := ezbson.Marshal(example)
	if err != nil {
		log.Fatal(err)
	}

	// The marshalled document is:
	// \x31\x00\x00\x00
	// \x04BSON\x00
	// \x26\x00\x00\x00
	// \x02\x30\x00\x08\x00\x00\x00awesome\x00
	// \x01\x31\x00\x33\x33\x33\x33\x33\x33\x14\x40
	// \x10\x32\x00\xc2\x07\x00\x00
	// \x00
	// \x00

	unmarshalled := ExampleStruct{}
	err = ezbson.Unmarshal(bson, &unmarshalled)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(unmarshalled) // {[awesome 5.05 1986]}

	asMap := make(map[string]any)
	err = ezbson.Unmarshal(bson, &asMap)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(asMap) // map[BSON:[awesome 5.05 1986]]
}
```

## Limitations
- Serializing or deserializing structs with unexported-fields will cause panics, due to the way reflect works.
- Currently only supports 64 bit architecture (but this can be fixed).

## Contributing

Contributions are welcome! Please feel free to submit bugs, feature requests and pull requests.

## License

ezbson is licensed under the [MIT license](/LICENSE).