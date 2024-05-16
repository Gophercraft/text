package text_test

import (
	"bytes"
	"testing"

	"github.com/Gophercraft/text"
)

type record1 struct {
	ID      uint64
	Key     string
	Strings []string
}

type test_case1 struct {
	Record1      []record1
	EncodedTable string
}

var cases1 = []test_case1{
	{
		Record1: []record1{
			{
				ID:  1,
				Key: "ABCDEFGHIJKLMNOP",
				Strings: []string{
					"00",
					"01",
					"02",
					"03",
				},
			},
		},

		EncodedTable: `[ ID Key Strings ]
{ 1 ABCDEFGHIJKLMNOP { 00 01 02 03 } }
`,
	},
}

func TestEncodeTable(t *testing.T) {

	for _, case1 := range cases1 {
		var buf bytes.Buffer
		encoder := text.NewEncoder(&buf)
		encoder.Indent = " "
		encoder.Tabular = true

		for _, record := range case1.Record1 {
			if err := encoder.Encode(&record); err != nil {
				t.Fatal(err)
			}
		}

		str := buf.String()
		expected := case1.EncodedTable
		if str != expected {
			t.Fatal(str, "should have been equal to", expected)
		}
	}

}
