package text

import (
	"errors"
	"fmt"
	"io"
)

type token_type uint8

const (
	token_open token_type = iota
	token_close
	token_word
	token_open_table_header
	token_close_table_header
)

type token struct {
	Type token_type
	Data string
}

func (decoder *Decoder) read_quoted_word() (word *token, err error) {
	word = &token{token_word, ""}
	_, err = decoder.input.ReadByte()
	if err != nil {
		return
	}

	for {
		var next_char rune
		next_char, _, err = decoder.input.ReadRune()
		if err != nil {
			return
		}

		if next_char == '"' {
			return
		}

		if next_char == '\n' {
			decoder.line++
			decoder.column = 0
		}

		if next_char == '\\' {
			var escaped_char rune
			escaped_char, _, err = decoder.input.ReadRune()
			if err != nil {
				return
			}

			switch escaped_char {
			case 'n':
				next_char = '\n'
			case 'r':
				next_char = '\r'
			case 't':
				next_char = '\t'
			case '\\':
				next_char = '\\'
			case '"':
				next_char = '"'
			default:
				return nil, fmt.Errorf("unknown escape sequence: \\%c", next_char)
			}
		}

		decoder.column++

		word.Data += string(next_char)
	}
}

func (decoder *Decoder) read_word() (word *token, err error) {
	var beginning []byte
	beginning, err = decoder.input.Peek(1)
	if err != nil {
		err = fmt.Errorf("error peeking in read_word: %w", err)
		return
	}

	if beginning[0] == '"' {
		return decoder.read_quoted_word()
	}

	word = &token{token_word, ""}

	for {
		var next_char rune
		next_char, _, err = decoder.input.ReadRune()
		if err != nil {
			if errors.Is(err, io.EOF) {
				err = nil
				return
			}
			err = fmt.Errorf("error while reading next character: %w", err)
			return
		}

		// whitespace or new line can terminate a word
		switch next_char {
		case ' ':
			decoder.column++
			return
		case '\n':
			decoder.column = 1
			decoder.line++
			return
		case '\r':
			return
		case '\t':
			decoder.column++
			return
		}

		word.Data += string(next_char)
		decoder.column++
	}
}

// Read a token from the input stream while not consuming it
func (decoder *Decoder) peek_token() (t *token, err error) {
	t, err = decoder.next_token()
	if err != nil {
		return
	}

	decoder.peeked_tokens = append(decoder.peeked_tokens, t)
	return
}

// Consume a token
func (decoder *Decoder) next_token() (t *token, err error) {
	if len(decoder.peeked_tokens) > 0 {
		t = decoder.peeked_tokens[0]
		decoder.peeked_tokens = decoder.peeked_tokens[1:]
		return
	}

	var b []byte

main_loop:
	for {
		b, err = decoder.input.Peek(1)
		if err != nil {
			err = fmt.Errorf("error peeking in input: %w", err)
			return
		}

		switch b[0] {
		case '/':
			var ss []byte
			ss, err = decoder.input.Peek(2)
			if err != nil {
				return
			}
			// Read double-slash comment
			if ss[1] == '/' {
				if _, err = decoder.input.ReadString('\n'); err != nil {
					return
				}
				continue main_loop
			} else if ss[1] == '*' {
				// Read a block comment
				var d [2]byte
				decoder.input.Read(d[:])

				for {
					var r rune
					r, _, err = decoder.input.ReadRune()
					if err != nil {
						return
					}

					if r == '*' {
						r, _, err = decoder.input.ReadRune()
						if err != nil {
							return
						}

						if r == '/' {
							continue main_loop
						}
					}
				}
			} else {
				return nil, fmt.Errorf("stray comment")
			}
		// whitespace
		case ' ', '\t':
			decoder.input.ReadByte()
			decoder.column++
			continue
		case '\r':
			decoder.input.ReadByte()
			decoder.column++
			continue
		case '\n':
			decoder.input.ReadByte()
			decoder.line++
			decoder.column = 1
		case '[':
			decoder.input.ReadByte()
			decoder.column++
			t = &token{Type: token_open_table_header}
			return
		case ']':
			decoder.input.ReadByte()
			decoder.column++
			t = &token{Type: token_close_table_header}
			return
		case '{':
			decoder.input.ReadByte()
			decoder.column++
			t = &token{Type: token_open}
			return
		case '}':
			decoder.input.ReadByte()
			decoder.column++
			t = &token{Type: token_close}
			return
		default:
			t, err = decoder.read_word()
			return
		}
	}
}

func (decoder *Decoder) next_word() (word *token, err error) {
	word, err = decoder.next_token()
	if err != nil {
		return nil, err
	}

	if word.Type != token_word {
		return nil, fmt.Errorf("invalid Token type %d", word.Type)
	}

	return
}
