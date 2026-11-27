package httpjson_test

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/jacob2161/httpjson"
)

type createUser struct {
	Name string `json:"name"`
}

func handleCreateUser(w http.ResponseWriter, req *http.Request) {
	var request createUser
	if err := httpjson.ReadRequest(req, 1<<20, &request); err != nil {
		status := http.StatusBadRequest
		if _, tooLarge := errors.AsType[*http.MaxBytesError](err); tooLarge {
			status = http.StatusRequestEntityTooLarge
		}
		_ = httpjson.Write(w, status, map[string]string{"error": err.Error()})
		return
	}
	_ = httpjson.Write(w, http.StatusCreated, map[string]string{"name": request.Name})
}

func Example() {
	for _, body := range []string{`{"name":"ada"}`, `{"name":"ada","admin":true}`} {
		recorder := httptest.NewRecorder()
		handleCreateUser(recorder, httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(body)))
		fmt.Print(recorder.Code, " ", recorder.Body.String())
	}
	// Output:
	// 201 {"name":"ada"}
	// 400 {"error":"json: unknown field \"admin\""}
}
