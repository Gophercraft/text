# Gophercraft/text

[![Go Reference](https://pkg.go.dev/badge/github.com/Gophercraft/phylactery.svg)](https://pkg.go.dev/github.com/Gophercraft/text)
[![License: GPL v3](https://img.shields.io/badge/License-GPLv3-blue.svg)](https://www.gnu.org/licenses/gpl-3.0)
[![Chat on discord](https://img.shields.io/discord/556039662997733391.svg)](https://discord.gg/xPtuEjt)

The Gophercraft text format is used in Gophercraft for configuration and data interchange.

You can think about it as a more lightweight JSON, but with some extra bells and whistles.

Another difference from JSON is that it cannot be used to represent arbitrary structures. Data must be serialized according to Go types.

## Example document

```go
type Document struct {
  StringField  string
  IntegerSlice []int
  Map          map[string]string
}
```

```c
{
  /*
  *
  *
  * Block comments
  * 
  */
  
  // Or double-slash comments are allowed

  // Quotes are not required for words if they contain no whitespace or reserved characters
  StringField QuotesUnnecessary

  // Keys and values are both words. All words may be quoted.
  "IntegerSlice"
  {
    1
    2
    3
    "4" // Not a string, but can be quoted.
  }

  Map
  {
    "Key" value
    other_key "another value"
  }
}
```

## Tabular documents

Encoded values can also be expressed in a tabular form (unkeyed structs).

This can be desirable when working with database files that are extremely large and contain many records, as it reduces the redudancy of repeating struct keys.

```c
// Table header
[ StringField IntegerSlice Map ]
// Empty row
{ "" {} {} }
// Non-empty row
{ "string value" { 1 2 3 4 } { key value otherkey othervalue } }
```

## Usage

Easy functions for dealing with a single record:

```go
bytes, err := text.Marshal(&record)

err = text.Unmarshal(bytes, &record)
```

Use Encoder and Decoder to deal with many records in one stream, or make use of custom options:

```go
encoder := text.NewEncoder(file)

// When using standard notation
encoder.Indent = "\t"

// When using tabular notation, this is what is placed between columns
encoder.Indent = " "

// Set tabular to true if you wish to encode
// table files with a much smaller size (See: Tabular documents)
encoder.Tabular = true|false

for _, record := range records {
  err = encoder.Encode(&record)
  // ...
}
```


```go
decoder := text.NewDecoder(file)

for {
  err = decoder.Decode(&record)
  if errors.Is(err, io.EOF) {
    // done reading
    break
  } else if err != nil {
    // deal with error
    break
  }
  // handle decoded record
}

// close fd
file.Close()
```

