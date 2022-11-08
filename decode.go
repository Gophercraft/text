package text

import (
	"bufio"
	"fmt"
	"io"
	"reflect"
	"strconv"
	"strings"

	"github.com/davecgh/go-spew/spew"
)

// Word describes custom data types that use one string.
// Useful for custom integral types
type Word interface {
	EncodeWord() (string, error)
	DecodeWord(data string) error
}

var (
	wordType = reflect.TypeOf((*Word)(nil)).Elem()
)

type Decoder struct {
	input        *bufio.Reader
	line, column int
	tokens       []*Token
}

func NewDecoder(in io.Reader) *Decoder {
	return &Decoder{
		input:  bufio.NewReader(in),
		line:   1,
		column: 1,
	}
}

func bitSize(t reflect.Kind) int {
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

func initValue(value reflect.Value) {
	switch value.Kind() {
	case reflect.Map:
		value.Set(reflect.MakeMap(value.Type()))
	}
}

func (decoder *Decoder) decodeValue(value reflect.Value) error {
	if canEncodeWord(value) {
		var word Word
		if reflect.PtrTo(value.Type()).Implements(wordType) {
			word = value.Addr().Interface().(Word)
		} else if value.Type().Implements(wordType) {
			word = value.Interface().(Word)
		}
		initValue(value)
		tok, err := decoder.NextWord()
		if err != nil {
			return err
		}

		return word.DecodeWord(tok.Data)
	}

	switch value.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		tok, err := decoder.NextWord()
		if err != nil {
			return err
		}

		i, err := strconv.ParseInt(tok.Data, 0, bitSize(value.Kind()))
		if err != nil {
			return err
		}

		value.SetInt(i)
		return nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		tok, err := decoder.NextWord()
		if err != nil {
			return err
		}

		i, err := strconv.ParseUint(tok.Data, 0, bitSize(value.Kind()))
		if err != nil {
			return err
		}

		value.SetUint(i)
		return nil
	case reflect.Float32, reflect.Float64:
		tok, err := decoder.NextWord()
		if err != nil {
			return err
		}
		float, err := strconv.ParseFloat(tok.Data, bitSize(value.Kind()))
		if err != nil {
			return err
		}
		value.SetFloat(float)
		return nil
	case reflect.Bool:
		tok, err := decoder.NextWord()
		if err != nil {
			return err
		}
		boolean, err := strconv.ParseBool(tok.Data)
		if err != nil {
			return err
		}
		value.SetBool(boolean)
		return nil
	case reflect.String:
		tok, err := decoder.NextWord()
		if err != nil {
			return err
		}
		value.SetString(tok.Data)
		return nil
	case reflect.Array:
		tok, err := decoder.nextToken()
		if err != nil {
			return err
		}

		if tok.Type != TokOpen {
			return fmt.Errorf("invalid token at start of array: %d", tok.Type)
		}

		var lastToken *Token

		for x := 0; x < value.Len(); x++ {
			nextToken, err := decoder.nextToken()
			if err != nil {
				return err
			}
			if nextToken.Type == TokClose {
				break
			}
			decoder.tokens = []*Token{nextToken}
			err = decoder.decodeValue(value.Index(x))
			if err != nil {
				return err
			}
			lastToken = nextToken
		}

		if lastToken.Type != TokClose {
			return fmt.Errorf("no close token at end of array")
		}
		return nil
	case reflect.Slice:
		tok, err := decoder.nextToken()
		if err != nil {
			return err
		}

		if tok.Type != TokOpen {
			return fmt.Errorf("invalid token at start of slice: %d", tok.Type)
		}

		for {
			nextToken, err := decoder.nextToken()
			if err != nil {
				return err
			}
			if nextToken.Type == TokClose {
				break
			}
			decoder.tokens = []*Token{nextToken}
			newObject := reflect.New(value.Type().Elem())
			err = decoder.decodeValue(newObject.Elem())
			if err != nil {
				return err
			}
			value.Set(reflect.Append(value, newObject.Elem()))
		}

		return nil
	case reflect.Struct:
		fieldSet := map[string]bool{}

		openToken, err := decoder.nextToken()
		if err != nil {
			return err
		}

		if openToken.Type != TokOpen {
			return fmt.Errorf("struct needs open tag")
		}

		for {
			nextToken, err := decoder.nextToken()
			if err != nil {
				return err
			}

			if nextToken.Type == TokClose {
				break
			}

			if nextToken.Type != TokWord {
				return fmt.Errorf("non-word token as struct key %d", nextToken.Type)
			}

			fieldName := nextToken.Data

			if fieldName == "" {
				return fmt.Errorf("empty keyword in struct")
			}

			var field reflect.Value
			// Multi-field key
			if strings.Contains(fieldName, ".") {
				field = value
				fieldNames := strings.Split(fieldName, ".")
				fieldNamesIter := fieldNames

				for len(fieldNamesIter) > 0 {
					nextFieldKey := fieldNamesIter[0]
					fieldNamesIter = fieldNamesIter[1:]

					field = field.FieldByName(nextFieldKey)
					if !field.IsValid() {
						return fmt.Errorf("no field by the name of %s (%s)", strings.Join(fieldNames, "."), nextFieldKey)
					}
				}
			} else {
				field = value.FieldByName(fieldName)

				if !field.IsValid() {
					return fmt.Errorf("no field by the name of %s", spew.Sdump(fieldName))
				}
			}

			err = decoder.decodeValue(field)
			if err != nil {
				return err
			}

			if fieldSet[fieldName] {
				return fmt.Errorf("field %s already set", fieldName)
			}

			fieldSet[fieldName] = true
		}
		return nil
	case reflect.Map:
		openToken, err := decoder.nextToken()
		if err != nil {
			return err
		}

		if openToken.Type != TokOpen {
			return fmt.Errorf("invalid token at start of Map: %d: %s", openToken.Type, openToken.Data)
		}

		value.Set(reflect.MakeMap(value.Type()))

		for {
			nextToken, err := decoder.nextToken()
			if err != nil {
				return err
			}
			if nextToken.Type == TokClose {
				break
			} else {
				decoder.tokens = []*Token{nextToken}
			}

			keyValue := reflect.New(value.Type().Key()).Elem()
			if err := decoder.decodeValue(keyValue); err != nil {
				return err
			}

			mapValue := reflect.New(value.Type().Elem()).Elem()

			if err := decoder.decodeValue(mapValue); err != nil {
				return err
			}

			value.SetMapIndex(keyValue, mapValue)
		}
		return nil
	default:
		if !value.IsValid() {
			panic("invalid value")
		}
		return fmt.Errorf("unknown kind for %s", value.Kind())
	}
	panic("should be unreachable")
}

func (decoder *Decoder) Decode(i interface{}) error {
	value := reflect.ValueOf(i)
	if value.Kind() == reflect.Ptr {
		value = value.Elem()
	}
	value.Set(reflect.New(value.Type()).Elem())
	return decoder.decodeValue(value)
}
