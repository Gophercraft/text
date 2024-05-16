package text

import (
	"bufio"
	"fmt"
	"io"
	"reflect"
	"strconv"

	"github.com/davecgh/go-spew/spew"
)

// Word describes custom data types that use one string.
// Useful for custom integral types
type Word interface {
	EncodeWord() (string, error)
	DecodeWord(data string) error
}

var (
	word_type = reflect.TypeFor[Word]()
)

// A Decoder reads and decodes text values from an input stream.
type Decoder struct {
	input           *bufio.Reader
	coding_is_known bool
	tabular         bool
	line, column    int
	peeked_tokens   []*token
	columns         []string
}

// NewDecoder returns a new decoder that reads from r.
// The decoder introduces its own buffering and may read data from r beyond the text values requested.
func NewDecoder(in io.Reader) *Decoder {
	return &Decoder{
		input:  bufio.NewReader(in),
		line:   1,
		column: 1,
	}
}

// Get the size in bits of a numeric type kind
func bit_size(t reflect.Kind) int {
	switch t {
	case reflect.Uint, reflect.Uint64, reflect.Int, reflect.Int64:
		return 64
	case reflect.Uint32, reflect.Int32:
		return 32
	case reflect.Uint16, reflect.Int16:
		return 16
	case reflect.Uint8, reflect.Int8:
		return 8
	case reflect.Float32:
		return 32
	case reflect.Float64:
		return 64
	default:
		panic(t)
	}
}

// Initializes the value of types that need to be before they are used
func initialize_value(value reflect.Value) {
	switch value.Kind() {
	case reflect.Map:
		value.Set(reflect.MakeMap(value.Type()))
	}
}

func (decoder *Decoder) decode_int(value reflect.Value) (err error) {
	var (
		int_token *token
		i         int64
	)
	int_token, err = decoder.next_word()
	if err != nil {
		return
	}

	i, err = strconv.ParseInt(int_token.Data, 0, bit_size(value.Kind()))
	if err != nil {
		return
	}

	value.SetInt(i)
	return
}

func (decoder *Decoder) decode_uint(value reflect.Value) (err error) {
	var (
		uint_token *token
		u          uint64
	)
	uint_token, err = decoder.next_word()
	if err != nil {
		return
	}

	u, err = strconv.ParseUint(uint_token.Data, 0, bit_size(value.Kind()))
	if err != nil {
		return
	}

	value.SetUint(u)
	return
}

func (decoder *Decoder) decode_float(value reflect.Value) (err error) {
	var (
		float_token *token
		f           float64
	)
	float_token, err = decoder.next_word()
	if err != nil {
		return
	}
	f, err = strconv.ParseFloat(float_token.Data, bit_size(value.Kind()))
	if err != nil {
		return err
	}
	value.SetFloat(f)
	return
}

func (decoder *Decoder) decode_bool(value reflect.Value) (err error) {
	var (
		boolean_token *token
		b             bool
	)
	boolean_token, err = decoder.next_word()
	if err != nil {
		err = fmt.Errorf("error getting boolean word token: %w", err)
		return
	}
	b, err = strconv.ParseBool(boolean_token.Data)
	if err != nil {
		return
	}
	value.SetBool(b)
	return
}

func (decoder *Decoder) decode_string(value reflect.Value) (err error) {
	var (
		string_token *token
	)
	string_token, err = decoder.next_word()
	if err != nil {
		return err
	}
	value.SetString(string_token.Data)
	return nil
}

func (decoder *Decoder) decode_array(value reflect.Value) (err error) {
	var (
		open_token  *token
		next_token  *token
		close_token *token
	)
	open_token, err = decoder.next_token()
	if err != nil {
		return
	}

	if open_token.Type != token_open {
		return fmt.Errorf("invalid token at start of array: %d", open_token.Type)
	}

	for i := range value.Len() {
		next_token, err = decoder.peek_token()
		if err != nil {
			return
		}
		if next_token.Type == token_close {
			break
		}
		err = decoder.decode_value(value.Index(i))
		if err != nil {
			return err
		}
	}

	close_token, err = decoder.next_token()
	if err != nil {
		return
	}

	if close_token.Type != token_close {
		err = fmt.Errorf("array in text appears to exceed bounds")
	}

	return
}

func (decoder *Decoder) decode_slice(value reflect.Value) (err error) {
	var (
		open_token    *token
		next_token    *token
		close_token   *token
		slice_element reflect.Value
	)
	open_token, err = decoder.next_token()
	if err != nil {
		return
	}

	if open_token.Type != token_open {
		err = fmt.Errorf("invalid token at start of slice: %d", open_token.Type)
		return
	}

	for {
		next_token, err = decoder.peek_token()
		if err != nil {
			return err
		}
		if next_token.Type == token_close {
			break
		}
		// element must be allocated
		slice_element = reflect.New(value.Type().Elem()).Elem()
		err = decoder.decode_value(slice_element)
		if err != nil {
			return
		}
		value.Set(reflect.Append(value, slice_element))
	}

	close_token, err = decoder.next_token()
	if err != nil {
		return
	}

	if close_token.Type != token_close {
		err = fmt.Errorf("slice does not end with close token?")
	}

	return
}

func (decoder *Decoder) decode_map(value reflect.Value) (err error) {
	var (
		open_token  *token
		next_token  *token
		close_token *token
	)

	open_token, err = decoder.next_token()
	if err != nil {
		return
	}

	if open_token.Type != token_open {
		err = fmt.Errorf("invalid token at start of Map: %d: %s", open_token.Type, open_token.Data)
		return
	}

	value.Set(reflect.MakeMap(value.Type()))

	for {
		next_token, err = decoder.peek_token()
		if err != nil {
			return err
		}
		if next_token.Type == token_close {
			break
		}

		key_value := reflect.New(value.Type().Key()).Elem()

		if err := decoder.decode_value(key_value); err != nil {
			return err
		}

		map_value := reflect.New(value.Type().Elem()).Elem()

		if err := decoder.decode_value(map_value); err != nil {
			return err
		}

		value.SetMapIndex(key_value, map_value)
	}

	close_token, err = decoder.next_token()
	if err != nil {
		return
	}

	if close_token.Type != token_close {
		err = fmt.Errorf("map does not end with close token?")
	}

	return
}

func (decoder *Decoder) decode_struct(value reflect.Value) (err error) {
	var (
		open_token *token
		next_token *token
	)

	set_fields := make(map[string]bool, value.Type().NumField())

	open_token, err = decoder.next_token()
	if err != nil {
		err = fmt.Errorf("error in getting open token: %w", err)
		return
	}

	if open_token.Type != token_open {
		err = fmt.Errorf("struct needs open tag")
		return
	}

	for {
		next_token, err = decoder.next_token()
		if err != nil {
			err = fmt.Errorf("error in next_token: %w", err)
			return
		}

		if next_token.Type == token_close {
			break
		}

		if next_token.Type != token_word {
			err = fmt.Errorf("non-word token as struct key %d", next_token.Type)
			return
		}

		field_name := next_token.Data

		if field_name == "" {
			err = fmt.Errorf("empty keyword in struct")
			return
		}

		field := value.FieldByName(field_name)

		if !field.IsValid() {
			return fmt.Errorf("no field by the name of %s", spew.Sdump(field_name))
		}

		err = decoder.decode_value(field)
		if err != nil {
			err = fmt.Errorf("error in decode_value: %w", err)
			return
		}

		if set_fields[field_name] {
			return fmt.Errorf("field %s already set", field_name)
		}

		set_fields[field_name] = true
	}
	return
}

// Consumes a text value from the buffered input stream and decodes it into value
func (decoder *Decoder) decode_value(value reflect.Value) (err error) {
	if can_encode_word(value) {
		var (
			word       Word
			word_token *token
		)
		if reflect.PointerTo(value.Type()).Implements(word_type) {
			word = value.Addr().Interface().(Word)
		} else if value.Type().Implements(word_type) {
			word = value.Interface().(Word)
		}
		initialize_value(value)
		word_token, err = decoder.next_word()
		if err != nil {
			return
		}

		return word.DecodeWord(word_token.Data)
	}

	switch value.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return decoder.decode_int(value)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return decoder.decode_uint(value)
	case reflect.Float32, reflect.Float64:
		return decoder.decode_float(value)
	case reflect.Bool:
		return decoder.decode_bool(value)
	case reflect.String:
		return decoder.decode_string(value)
	case reflect.Array:
		return decoder.decode_array(value)
	case reflect.Slice:
		return decoder.decode_slice(value)
	case reflect.Struct:
		return decoder.decode_struct(value)
	case reflect.Map:
		return decoder.decode_map(value)
	default:
	}

	return fmt.Errorf("unknown kind for %s", value.Kind())
}

func (decoder *Decoder) Decode(value any) (err error) {
	if !decoder.coding_is_known {
		var first_token *token
		first_token, err = decoder.peek_token()
		if err == nil {
			if first_token.Type == token_open_table_header {
				decoder.tabular = true
				err = decoder.read_table_header()
				if err != nil {
					return
				}
			}
		}
		decoder.coding_is_known = true
	}

	v := reflect.ValueOf(value)
	if v.Kind() == reflect.Pointer {
		v = v.Elem()
	}
	v.Set(reflect.New(v.Type()).Elem())
	if decoder.tabular {
		return decoder.decode_row(v)
	} else {
		return decoder.decode_value(v)
	}
}
