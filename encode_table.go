package text

import (
	"fmt"
	"reflect"
	"strconv"
)

func (encoder *Encoder) encode_column(value reflect.Value) (err error) {
	if can_encode_word(value) {
		return encoder.encode_word(value)
	}

	switch value.Kind() {
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		if _, err := encoder.out.Write([]byte(strconv.FormatUint(value.Uint(), 10))); err != nil {
			return err
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		if _, err := encoder.out.Write([]byte(strconv.FormatInt(value.Int(), 10))); err != nil {
			return err
		}
	case reflect.Float32, reflect.Float64:
		if _, err := encoder.out.Write([]byte(strconv.FormatFloat(value.Float(), 'f', -1, bit_size(value.Kind())))); err != nil {
			return err
		}
	case reflect.Bool:
		if _, err := encoder.out.Write([]byte(strconv.FormatBool(value.Bool()))); err != nil {
			return err
		}
	case reflect.String:
		if err := encoder.encode_string(value.String()); err != nil {
			return err
		}
	case reflect.Slice, reflect.Array:
		if value.Len() == 0 {
			encoder.out.Write([]byte("{}"))
		} else {
			encoder.out.Write([]byte("{ "))
			for x := 0; x < value.Len(); x++ {
				if err := encoder.encode_column(value.Index(x)); err != nil {
					return err
				}
				encoder.out.Write([]byte(" "))
			}
			encoder.out.Write([]byte("}"))
		}
	case reflect.Struct:
		if value.IsZero() {
			encoder.out.Write([]byte("{}"))
		} else {
			encoder.out.Write([]byte("{ "))

			for x := 0; x < value.NumField(); x++ {
				field := value.Field(x)

				if err := encoder.encode_column(field); err != nil {
					return err
				}

				encoder.out.Write([]byte(" "))
			}
			encoder.out.Write([]byte("}"))
		}
	case reflect.Map:
		if value.IsZero() {
			encoder.out.Write([]byte("{}"))
		} else {
			encoder.out.Write([]byte("{ "))

			map_keys := value.MapKeys()
			sort_values(map_keys)

			for _, key := range map_keys {
				if err := encoder.encode_column(key); err != nil {
					return err
				}

				field := value.MapIndex(key)

				encoder.out.Write([]byte(" "))
				if err := encoder.encode_column(field); err != nil {
					return err
				}

				encoder.out.Write([]byte(" "))
			}

			encoder.out.Write([]byte("}"))
		}
	default:
		return fmt.Errorf("unknown kind %s", value.Kind())
	}

	return nil
}

func (encoder *Encoder) encode_row(value reflect.Value) (err error) {
	if value.Kind() != reflect.Struct {
		err = fmt.Errorf("to use tabular encoding, a row must be a struct")
		return
	}

	num_field := value.NumField()

	if value.IsZero() {
		_, err = encoder.out.Write([]byte("{}"))
		return
	}

	_, err = encoder.out.Write([]byte("{ "))
	if err != nil {
		return
	}

	for i := range num_field {
		if err = encoder.encode_column(value.Field(i)); err != nil {
			return
		}

		if _, err = encoder.out.Write([]byte(encoder.Indent)); err != nil {
			return
		}
	}

	_, err = encoder.out.Write([]byte("}\n"))
	return
}
