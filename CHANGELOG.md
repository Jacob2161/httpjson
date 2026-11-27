# Changelog

This module follows semantic versioning. Before v1, a minor version may
change the API; each such change says what callers must do.

## Unreleased

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
