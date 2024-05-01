package text

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"testing"

	"github.com/davecgh/go-spew/spew"
)

type Something struct {
	Data string
}

func (some *Something) DecodeWord(data string) error {
	some.Data = "somethin:" + data
	return nil
}

func (some *Something) EncodeWord() (string, error) {
	return "something:" + some.Data, nil
}

type Dict map[string]string

func (d Dict) DecodeWord(data string) error {
	wz := strings.Split(data, ",")
	for _, w := range wz {
		kv := strings.SplitN(w, ":", 2)
		d[kv[0]] = kv[1]
	}
	return nil
}

func (d Dict) EncodeWord() (string, error) {
	var keys []string
	for k := range d {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	kvs := []string{}
	for _, k := range keys {
		v := d[k]
		kvs = append(kvs, k+":"+v)
	}
	return strings.Join(kvs, ","), nil
}

func TestDecode(t *testing.T) {
	type testStruct struct {
		Strfield   string
		IntField   uint32
		FloatField float64
		SliceField []float32
		Fielded    struct {
			Data string
		}
		MapThing  map[string]string
		Dict      Dict
		Some      Something
		FloatDict map[float32]struct {
			Test string
		}
	}

	code := `{
		Strfield 1
		FloatField 3.5
	}
	{ 
		Strfield "hello\t\nworld" // Multiline strings are allowed, however you can use escape sequences
		FloatField 19.17
		SliceField
		{
			123
			596
			2313
			3414.4
		}
	}{
		Strfield "hey friends
			how's it going
		"
	}
	// Fields can also be omitted entirely
	{ // quotes in weird places
	}
	// Structs can be one-liners if you want them to
	{ Strfield hi FloatField "3.6"  }
	{
		Fielded {
			Data "Test"
		}
	}
	/*
	!!! 23ldaladalxzcz
	dfaleww {}ASD}WE
	*/
	// Map
	{
		MapThing
		{
			"Something" "else"
			EvenHave.Periods too
		}

		/*
		* !!! 23ldaladalxzcz
		* dfaleww {}ASD}WE
		* 
		*
		*/
		Dict johnny:got,his:gun

		Some huh

		FloatDict
		{
			300.500 {
				Test "test"
			}
			29345934544 {
				Test words50wordswords
			}
		}
	}
	`

	var test testStruct

	reader := strings.NewReader(code)
	decoder := NewDecoder(reader)

	writer := bytes.NewBuffer(nil)
	out := NewEncoder(writer)
	out.Indent = "  "

	for {
		err := decoder.Decode(&test)
		if errors.Is(err, io.EOF) {
			break
		}

		if err != nil {
			t.Fatal(err)
		}

		fmt.Println("Strfield", test.Strfield)

		fmt.Println(spew.Sdump(test))

		if err := out.Encode(&test); err != nil {
			t.Fatal(err)
		}

		fmt.Println(spew.Sdump(test.Dict))
	}

	fmt.Println(writer.String())

	bools := "true"
	var b bool
	dec := NewDecoder(strings.NewReader(bools))

	if err := dec.Decode(&b); err != nil {
		t.Fatal(err)
	}

	if b != true {
		panic(b)
	}
}
