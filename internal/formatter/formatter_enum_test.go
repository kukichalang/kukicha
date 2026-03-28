package formatter

import "testing"

func TestFormatEnumDecl(t *testing.T) {
	source := `enum Status
    OK = 200
    NotFound = 404
    Error = 500

func main()
    s := Status.OK
`
	expected := `enum Status
    OK = 200
    NotFound = 404
    Error = 500

func main()
    s := Status.OK
`
	assertFormatted(t, source, expected)
}

func TestFormatEnumDeclString(t *testing.T) {
	source := `enum LogLevel
    Debug = "debug"
    Info = "info"

func main()
    l := LogLevel.Debug
`
	expected := `enum LogLevel
    Debug = "debug"
    Info = "info"

func main()
    l := LogLevel.Debug
`
	assertFormatted(t, source, expected)
}
