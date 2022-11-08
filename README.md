# Gophercraft Text Format

To keep the format simple and sane, it cannot deal with arbitrary data: data is parsed according to a known schema.

## Example document

```go
type Document struct {
  StringField string
  IntegerSlice []int
  Embedded    struct {
    Value float32
  }
}
```

```c
{
  // Comments are allowed. C-style block comments are not yet implemented, but they may be in the future.
  StringField QuotesUnnecessary

  // Keys and values are both words. All words may be quoted.
  "IntegerSlice"
  {
    1
    2
    3
    "4" // Not a string, but can be quoted.
  }

  // Brackets are not always necessary to set a value in a struct.
  Embedded.Value 123.456
}
```