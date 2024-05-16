package text

import (
	"fmt"
	"reflect"

	"github.com/davecgh/go-spew/spew"
)

func (decoder *Decoder) decode_map_column(value reflect.Value) (err error) {
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

		if err := decoder.decode_column(map_value); err != nil {
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

func (decoder *Decoder) decode_array_column(value reflect.Value) (err error) {
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
		err = decoder.decode_column(value.Index(i))
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

func (decoder *Decoder) decode_slice_column(value reflect.Value) (err error) {
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
		err = decoder.decode_column(slice_element)
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

func (decoder *Decoder) read_table_header() (err error) {
	var (
		t *token
	)
	t, err = decoder.next_token()
	if err != nil {
		return
	}

	if t.Type != token_open_table_header {
		err = fmt.Errorf("table header is invalid")
		return
	}

	for {
		t, err = decoder.next_token()
		if err != nil {
			return
		}

		switch t.Type {
		case token_word:
			decoder.columns = append(decoder.columns, t.Data)
		case token_close_table_header:
			return
		default:
			err = fmt.Errorf("invalid token %d in table header", t.Type)
			return
		}
	}
}

func (decoder *Decoder) decode_column(value reflect.Value) (err error) {
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
		return decoder.decode_array_column(value)
	case reflect.Slice:
		return decoder.decode_slice_column(value)
	case reflect.Struct:
		return decoder.decode_unkeyed_struct(value)
	case reflect.Map:
		return decoder.decode_map_column(value)
	default:
	}

	return fmt.Errorf("unknown kind for %s", value.Kind())
}

func (decoder *Decoder) decode_unkeyed_struct(value reflect.Value) (err error) {
	var (
		open_token  *token
		close_token *token
		next_token  *token
	)

	open_token, err = decoder.next_token()
	if err != nil {
		err = fmt.Errorf("error in getting open token: %w", err)
		return
	}

	if open_token.Type != token_open {
		err = fmt.Errorf("struct needs open tag (tag is %d, '%s')", open_token.Type, open_token.Data)
		return
	}

	for i := 0; ; i++ {
		next_token, err = decoder.peek_token()
		if err != nil {
			return
		}

		if next_token.Type == token_close {
			break
		}

		if i >= value.Type().NumField() {
			err = fmt.Errorf("value [#%d] in row at line %d column %d exceeds the number of columns in table header", i, decoder.line, decoder.column)
			return
		}

		field := value.Field(i)

		if !field.IsValid() {
			return fmt.Errorf("no field in %s at the index %d", value.Type(), i)
		}

		err = decoder.decode_value(field)
		if err != nil {
			err = fmt.Errorf("error in decode_value: %w", err)
			return
		}
	}

	close_token, err = decoder.next_token()
	if err != nil {
		err = fmt.Errorf("error in getting open token: %w", err)
		return
	}

	if close_token.Type != token_close {
		err = fmt.Errorf("struct needs close tag")
		return
	}

	return
}

func (decoder *Decoder) decode_row(value reflect.Value) (err error) {
	var (
		open_token  *token
		next_token  *token
		close_token *token
	)

	open_token, err = decoder.next_token()
	if err != nil {
		err = fmt.Errorf("error in getting open token: %w", err)
		return
	}

	if open_token.Type != token_open {
		err = fmt.Errorf("struct needs open tag (tag type is %d, '%s')", open_token.Type, open_token.Data)
		return
	}

	for i := 0; ; i++ {
		next_token, err = decoder.peek_token()
		if err != nil {
			return
		}

		if next_token.Type == token_close {
			break
		}

		if i >= len(decoder.columns) {
			err = fmt.Errorf("value [#%d] in row at line %d exceeds the number of columns in table header", i, decoder.line)
			return
		}

		field_name := decoder.columns[i]

		field := value.FieldByName(field_name)

		if !field.IsValid() {
			return fmt.Errorf("no field by the name of %s", spew.Sdump(field_name))
		}

		err = decoder.decode_value(field)
		if err != nil {
			err = fmt.Errorf("error in decode_value: %w", err)
			return
		}
	}

	close_token, err = decoder.next_token()
	if err != nil {
		err = fmt.Errorf("error in getting open token: %w", err)
		return
	}

	if close_token.Type != token_close {
		err = fmt.Errorf("struct needs close tag")
		return
	}

	return
}
