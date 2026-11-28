# Changelog

This module follows semantic versioning. Before v1, a minor version may
change the API; each such change says what callers must do.

## v0.2.0 (2026-11-28)

- Requires Go 1.27, the newest Go release; Go 1.26 is no longer
  supported.
- CI runs staticcheck and govulncheck at the versions pinned in
  `tools/go.mod`, a module of its own that importers never see.
- ReadRequest and Write use `encoding/json/v2`: member names match
  exactly, and duplicate members and invalid UTF-8 are refused. Write no
  longer escapes `<`, `>`, and `&`, and writes nil slices and maps as
  `[]` and `{}`. Decoding errors read differently.

## v0.1.1 (2026-11-27)

- `Write` documents its argument as the response it encodes. No API
  change.
- CI also checks formatting, `go mod tidy -diff`, and `govulncheck`.

## v0.1.0 (2026-11-27)

First release.

- `ReadRequest` decodes exactly one bounded JSON value from a request body,
  refuses unknown fields, and reports an empty body (`ErrEmptyBody`), a
  second value (`ErrSecondValue`), or a body past the limit
  (`*http.MaxBytesError`).
- `Write` encodes a JSON response before sending anything, so an encoding
  failure leaves the response free for an error response.
