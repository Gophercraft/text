# Gophercraft/text

[![Go Reference](https://pkg.go.dev/badge/github.com/Gophercraft/phylactery.svg)](https://pkg.go.dev/github.com/Gophercraft/phylactery)
[![License: GPL v3](https://img.shields.io/badge/License-GPLv3-blue.svg)](https://www.gnu.org/licenses/gpl-3.0)
[![Chat on discord](https://img.shields.io/discord/556039662997733391.svg)](https://discord.gg/xPtuEjt)

The Gophercraft text format is used in Gophercraft for configuration and data interchange.

You can think about it as a more restricted version of JSON, but with comments

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