package httpjson

import (
	"errors"
	"io"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/iotest"
)

// namedRequest is the request type the table decodes.
type namedRequest struct {
	Name string `json:"name"`
}

// TestReadRequest checks one value, whitespace, a second value, garbage, an
// oversized suffix, an unknown field, an empty body, and a request at the
// size limit.
func TestReadRequest(t *testing.T) {
	const limit = 64
	limitReached := `{"name":"` + strings.Repeat("a", limit-len(`{"name":""}`)) + `"}`
	readCases := []struct {
		name     string
		body     string
		want     func(error) bool
		wantText string
	}{
		{name: "one value", body: `{"name":"first"}`, want: isNil, wantText: "nil"},
		{
			name:     "trailing whitespace",
			body:     "{\"name\":\"first\"}\n \t",
			want:     isNil,
			wantText: "nil",
		},
		{
			name:     "second value",
			body:     `{"name":"first"}{"name":"second"}`,
			want:     isError(ErrSecondValue),
			wantText: "ErrSecondValue",
		},
		{
			name:     "garbage after",
			body:     `{"name":"first"}garbage`,
			want:     isError(ErrSecondValue),
			wantText: "ErrSecondValue",
		},
		{
			name:     "oversized suffix",
			body:     `{"name":"first"}` + strings.Repeat(" ", limit),
			want:     isTooLarge,
			wantText: "an *http.MaxBytesError",
		},
		{
			name:     "unknown field",
			body:     `{"name":"first","other":1}`,
			want:     notNil,
			wantText: "an error",
		},
		{name: "empty", body: "", want: isError(ErrEmptyBody), wantText: "ErrEmptyBody"},
		{
			name:     "only whitespace",
			body:     "  \n",
			want:     isError(ErrEmptyBody),
			wantText: "ErrEmptyBody",
		},
		{name: "at the limit", body: limitReached, want: isNil, wantText: "nil"},
		{
			name:     "past the limit",
			body:     limitReached + " ",
			want:     isTooLarge,
			wantText: "an *http.MaxBytesError",
		},
	}
	for _, readCase := range readCases {
		t.Run(readCase.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(readCase.body))
			var request namedRequest
			if err := ReadRequest(req, limit, &request); !readCase.want(err) {
				t.Fatalf("ReadRequest(%q) = %v, want %s", readCase.body, err, readCase.wantText)
			}
		})
	}
}

func isNil(err error) bool  { return err == nil }
func notNil(err error) bool { return err != nil }
func isError(target error) func(error) bool {
	return func(err error) bool { return errors.Is(err, target) }
}

// isTooLarge reports the body limit's error.
func isTooLarge(err error) bool {
	var tooLarge *http.MaxBytesError
	return errors.As(err, &tooLarge)
}

// TestWriteMatchesTheEncoder checks that Write sends json.Encoder's bytes,
// with HTML characters escaped, sorted keys, and a trailing newline, the
// status it is given (200 when it is 0), and application/json.
func TestWriteMatchesTheEncoder(t *testing.T) {
	response := map[string]any{"name": "first", "html": "<a&b>", "count": 3}
	recorder := httptest.NewRecorder()
	if err := Write(recorder, http.StatusCreated, response); err != nil {
		t.Fatal(err)
	}
	want := "{\"count\":3,\"html\":\"\\u003ca\\u0026b\\u003e\",\"name\":\"first\"}\n"
	if recorder.Code != http.StatusCreated || recorder.Body.String() != want ||
		recorder.Header().Get("Content-Type") != "application/json" {
		t.Fatalf(
			"Write sent %d %q %q, want %d %q %q",
			recorder.Code,
			recorder.Header().Get("Content-Type"),
			recorder.Body.String(),
			http.StatusCreated,
			"application/json",
			want,
		)
	}
	unset := httptest.NewRecorder()
	if err := Write(unset, 0, response); err != nil {
		t.Fatal(err)
	}
	if unset.Code != http.StatusOK || unset.Body.String() != want {
		t.Fatalf(
			"Write without a status sent %d %q, want %d %q",
			unset.Code,
			unset.Body.String(),
			http.StatusOK,
			want,
		)
	}
}

// errStalledBody stands for a body read that failed after the first value.
var errStalledBody = errors.New("body stalled")

// TestReadRequestReturnsReadFailures checks that a read that fails after a
// whole value, such as a body that arrived too slowly, is returned as
// itself rather than as a second value.
func TestReadRequestReturnsReadFailures(t *testing.T) {
	body := io.MultiReader(
		strings.NewReader(`{"name":"first"}  `),
		iotest.ErrReader(errStalledBody),
	)
	req := httptest.NewRequest(http.MethodPost, "/", body)
	var request namedRequest
	if err := ReadRequest(req, 1<<10, &request); !errors.Is(err, errStalledBody) {
		t.Fatalf("a read failing after the value: %v, want the read's own error", err)
	}
}

// TestIncompleteSecondValue checks that the start of a second value that
// ends early is ErrSecondValue too.
func TestIncompleteSecondValue(t *testing.T) {
	for _, body := range []string{`{} {`, `{} "unterminated`, `{} [1,`} {
		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
		var target struct{}
		if err := ReadRequest(req, 1<<10, &target); !errors.Is(err, ErrSecondValue) {
			t.Errorf("%q: %v, want ErrSecondValue", body, err)
		}
	}
}

// TestWriteUnencodable checks that a value that cannot be encoded leaves the
// response unwritten.
func TestWriteUnencodable(t *testing.T) {
	recorder := httptest.NewRecorder()
	if err := Write(recorder, http.StatusCreated, math.NaN()); err == nil {
		t.Fatal("Write(NaN) returned no error, want the encoding error")
	}
	if recorder.Code != http.StatusOK || recorder.Body.Len() != 0 ||
		recorder.Header().Get("Content-Type") != "" {
		t.Fatalf(
			"the response was written: %d %q, want it untouched",
			recorder.Code,
			recorder.Body.String(),
		)
	}
}

// TestTruncatedTransportIsItsOwnError checks that a body whose reader fails
// with io.ErrUnexpectedEOF after the value reports that, not a second value.
func TestTruncatedTransportIsItsOwnError(t *testing.T) {
	body := io.MultiReader(strings.NewReader(`{} `), iotest.ErrReader(io.ErrUnexpectedEOF))
	req := httptest.NewRequest(http.MethodPost, "/", body)
	var target struct{}
	err := ReadRequest(req, 1<<10, &target)
	if !errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, ErrSecondValue) {
		t.Fatalf("a truncated transport: %v, want io.ErrUnexpectedEOF alone", err)
	}
}
