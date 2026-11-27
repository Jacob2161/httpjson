# httpjson

[![Go Reference](https://pkg.go.dev/badge/github.com/jacob2161/httpjson.svg)](https://pkg.go.dev/github.com/jacob2161/httpjson)
[![CI](https://github.com/jacob2161/httpjson/actions/workflows/ci.yml/badge.svg)](https://github.com/jacob2161/httpjson/actions/workflows/ci.yml)

Strict JSON request bodies for net/http, and JSON responses written one way.

```bash
go get github.com/jacob2161/httpjson
```

## Usage

```go
var request struct{ Name string }
if err := httpjson.ReadRequest(req, 1<<20, &request); err != nil {
	// *http.MaxBytesError, httpjson.ErrEmptyBody, httpjson.ErrSecondValue,
	// or a *json.SyntaxError / *json.UnmarshalTypeError / unknown field error.
	http.Error(w, err.Error(), http.StatusBadRequest)
	return
}
_ = httpjson.Write(w, http.StatusCreated, response)
```

- The body is bounded, unknown fields are refused, and the whole body is read: trailing bytes or a second value are an error, not silently ignored.
- A read failure after the value (a body past the limit, a slow client) is reported as itself, not as a second value.
- Mapping errors to your API's error responses is left to you.

Tested on the two newest Go releases. Licensed under the MIT license.
