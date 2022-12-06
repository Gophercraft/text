package text

import (
	"fmt"
	"strconv"
	"testing"
)

type testMapEncodable map[testEncodable]bool

type testEncodable uint8

func (t *testEncodable) DecodeWord(str string) error {
	u, err := strconv.ParseUint(str, 0, 8)
	if err != nil {
		return err
	}

	*t = testEncodable(u)
	return nil
}

func (t *testEncodable) EncodeWord() (string, error) {
	return fmt.Sprintf("%d", *t), nil
}

type testDoc struct {
	V testMapEncodable
}

type MapNest map[string]MapNest

func TestEncode(t *testing.T) {

	doc := &testDoc{
		V: testMapEncodable{
			3: false,
		},
	}

	data, err := Marshal(doc)
	if err != nil {
		panic(err)
	}

	fmt.Println(string(data))

	mn := make(MapNest)

	mn["level1"] = MapNest{
		"level2": MapNest{
			"level3": MapNest{
				"level4": MapNest{},
			},
		},
	}

	data, err = Marshal(mn)
	if err != nil {
		panic(err)
	}

	fmt.Println(string(data))

}
