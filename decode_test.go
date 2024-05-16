package text_test

import (
	"errors"
	"io"
	"reflect"
	"strings"
	"testing"

	"github.com/Gophercraft/text"
)

func TestDecodeTable(t *testing.T) {
	for _, case1 := range cases1 {
		decoder := text.NewDecoder(strings.NewReader(case1.EncodedTable))

		var records []record1

		for {
			var record record1
			if err := decoder.Decode(&record); errors.Is(err, io.EOF) {
				break
			} else if err != nil {
				t.Fatal(err)
			}

			records = append(records, record)
		}

		if !reflect.DeepEqual(case1.Record1, records) {
			t.Fatal("got back incorrect records")
		}
	}

}
