package ezbson_test

import (
	"fmt"
	"log"

	"github.com/shimonp21/ezbson"
)

func Example_marshalUnmarshalMap() {
	// This is the example from https://bsonspec.org/faq.html
	example := map[string][]any{
		"BSON": {"awesome", 5.05, int32(1986)},
	}

	marshalled, err := ezbson.Marshal(example)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("%x\n", marshalled)

	unmarshalled := make(map[string][]any)

	err = ezbson.Unmarshal(marshalled, &unmarshalled)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Println(unmarshalled)

	// Output:
	// 310000000442534f4e002600000002300008000000617765736f6d65000131003333333333331440103200c20700000000
	// map[BSON:[awesome 5.05 1986]]
}
