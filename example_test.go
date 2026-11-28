package httpjson_test

import (
	json "encoding/json/v2"
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
		_, tooLarge := errors.AsType[*http.MaxBytesError](err)
		if tooLarge {
			status = http.StatusRequestEntityTooLarge
		}
		// The decoder's message changes between Go releases; the field at
		// fault does not.
		body := map[string]string{"error": "invalid request"}
		semantic, isSemantic := errors.AsType[*json.SemanticError](err)
		if isSemantic {
			body["field"] = string(semantic.JSONPointer)
		}
		_ = httpjson.Write(w, status, body)
		return
	}
	_ = httpjson.Write(w, http.StatusCreated, map[string]string{"name": request.Name})
}

func Example() {
	for _, body := range []string{`{"name":"ada"}`, `{"name":"ada","admin":true}`} {
		recorder := httptest.NewRecorder()
		handleCreateUser(
			recorder,
			httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(body)),
		)
		fmt.Print(recorder.Code, " ", recorder.Body.String())
	}
	// Output:
	// 201 {"name":"ada"}
	// 400 {"error":"invalid request","field":"/admin"}
}
