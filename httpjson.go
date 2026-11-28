// Package httpjson reads and writes the JSON bodies of HTTP APIs: a request
// body that holds exactly one bounded JSON value with no unknown fields, and
// responses written the same way everywhere. It holds only the transport
// mechanics; each API maps the errors to its own error responses.
package httpjson

import (
	"encoding/json/jsontext"
	json "encoding/json/v2"
	"errors"
	"io"
	"net/http"
)

var (
	// ErrEmptyBody reports a request without a JSON value.
	ErrEmptyBody = errors.New("the request body is empty")
	// ErrSecondValue reports a request body with more than one JSON value,
	// or bytes after the value that are not whitespace.
	ErrSecondValue = errors.New("the request body holds more than one JSON value")
)

// ReadRequest decodes the one JSON value of a request body into target. The
// body is bounded to limit bytes (an *http.MaxBytesError past it), unknown
// fields are refused, an empty body is ErrEmptyBody, and anything after the
// value but whitespace is ErrSecondValue, so the whole body is read and
// checked rather than its first value alone.
func ReadRequest(req *http.Request, limit int64, target any) error {
	body := &errorRecorder{reader: http.MaxBytesReader(nil, req.Body, limit)}
	decoder := jsontext.NewDecoder(body)
	err := json.UnmarshalDecode(decoder, target, json.RejectUnknownMembers(true))
	if errors.Is(err, io.EOF) {
		return ErrEmptyBody
	}
	if err != nil {
		return err
	}
	_, err = decoder.ReadValue()
	if errors.Is(err, io.EOF) {
		return nil
	}
	// A failed read after the value, such as a body past the limit or one
	// that arrived too slowly, is reported as itself; only more JSON, the
	// start of more JSON that ends early, or bytes that are not JSON is a
	// second value.
	if body.err != nil {
		return body.err
	}
	_, isSyntax := errors.AsType[*jsontext.SyntacticError](err)
	if err != nil && !isSyntax && !errors.Is(err, io.ErrUnexpectedEOF) {
		return err
	}
	return ErrSecondValue
}

// errorRecorder keeps the first error its reader returned other than io.EOF,
// so a failed read is told apart from JSON that the decoder found cut short.
type errorRecorder struct {
	reader io.Reader
	err    error
}

func (r *errorRecorder) Read(target []byte) (int, error) {
	n, err := r.reader.Read(target)
	if err != nil && !errors.Is(err, io.EOF) && r.err == nil {
		r.err = err
	}
	return n, err
}

// Write writes response as a JSON response body with status, or with the
// status the response writer holds already (200 unless set) when status is
// 0: map keys sorted, nil slices and maps as [] and {}, and a trailing
// newline. response is encoded before anything is sent, so one that cannot
// be encoded returns its error with the response still unwritten, free for
// an error response.
func Write(w http.ResponseWriter, status int, response any) error {
	encoded, err := json.Marshal(response, json.Deterministic(true))
	if err != nil {
		return err
	}
	w.Header().Set("Content-Type", "application/json")
	if status != 0 {
		w.WriteHeader(status)
	}
	_, err = w.Write(append(encoded, '\n'))
	return err
}
