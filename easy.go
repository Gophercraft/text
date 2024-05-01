package text

import "bytes"

func Marshal(value any) ([]byte, error) {
	out := new(bytes.Buffer)
	err := NewEncoder(out).Encode(value)
	return out.Bytes(), err
}

func Unmarshal(b []byte, value any) error {
	in := bytes.NewReader(b)
	err := NewDecoder(in).Decode(value)
	return err
}
