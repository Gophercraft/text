package text

import (
	"fmt"
	"io"
)

type TokenType uint8

const (
	TokOpen TokenType = iota
	TokClose
	TokWord
)

type Token struct {
	Type TokenType
	Data string
}

func (decoder *Decoder) getQuotedWord() (*Token, error) {
	tok := &Token{TokWord, ""}
	_, err := decoder.input.ReadByte()
	if err != nil {
		return nil, err
	}

	for {
		nextChar, _, err := decoder.input.ReadRune()
		if err != nil {
			return nil, err
		}

		if nextChar == '"' {
			return tok, nil
		}

		if nextChar == '\n' {
			decoder.line++
			decoder.column = 0
		}

		if nextChar == '\\' {
			escapedChar, _, err := decoder.input.ReadRune()
			if err != nil {
				return nil, err
			}

			switch escapedChar {
			case 'n':
				nextChar = '\n'
			case 'r':
				nextChar = '\r'
			case 't':
				nextChar = '\t'
			case '\\':
				nextChar = '\\'
			case '"':
				nextChar = '"'
			default:
				return nil, fmt.Errorf("unknown escape sequence: \\%c", nextChar)
			}
		}

		decoder.column++

		tok.Data += string(nextChar)
	}
}

func (decoder *Decoder) getWord() (*Token, error) {
	beginning, err := decoder.input.Peek(1)
	if err != nil {
		return nil, err
	}

	if beginning[0] == '"' {
		return decoder.getQuotedWord()
	}

	tok := &Token{TokWord, ""}

	for {
		b, _, err := decoder.input.ReadRune()
		if err != nil {
			return tok, err
		}

		// terminators
		switch b {
		case ' ':
			decoder.column++
			return tok, nil
		case '\n':
			decoder.column = 1
			decoder.line++
			return tok, nil
		case '\r':
			return tok, nil
		case '\t':
			decoder.column++
			return tok, nil
		}

		tok.Data += string(b)
		decoder.column++
	}
}

func (decoder *Decoder) nextToken() (tok *Token, err error) {
	if len(decoder.tokens) > 0 {
		tok = decoder.tokens[0]
		decoder.tokens = decoder.tokens[1:]
		return
	}

	var b []byte

	for {
		b, err = decoder.input.Peek(1)
		if err != nil {
			return nil, err
		}

		switch b[0] {
		case '/':
			ss, err := decoder.input.Peek(2)
			if err != nil {
				return nil, err
			}
			if ss[1] == '/' {
				if _, err := decoder.input.ReadString('\n'); err != nil {
					return nil, err
				}
				continue
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
		case '{':
			decoder.input.ReadByte()
			decoder.column++
			tok = &Token{Type: TokOpen}
			return
		case '}':
			decoder.input.ReadByte()
			decoder.column++
			tok = &Token{Type: TokClose}
			return
		default:
			tok, err = decoder.getWord()
			return
		}
	}
}

func (decoder *Decoder) NextWord() (*Token, error) {
	tok, err := decoder.nextToken()
	if err != nil {
		if err == io.EOF {
			if tok != nil {
				if len(tok.Data) > 0 {
					return tok, nil
				}
			}
		}

		return nil, err
	}

	if tok.Type != TokWord {
		return nil, fmt.Errorf("invalid Token type %d", tok.Type)
	}

	return tok, nil
}
