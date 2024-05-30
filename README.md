# go-struct-builder

`go-struct-builder` is a tool that implements builder pattern for structs by using `codegen`

[![Build Status][ci-badge]][ci-runs]

## Installation

### Go 1.18+

```bash
go install github.com/slavaavr/go-struct-builder/cmd/gosb@v1.0.0
```

## Quick start

- To generate a struct builder add the appropriate `go:generate` comment:
```go
// input.go - file where struct is located

//go:generate gosb -source=input.go
type A struct {
	F1 int
	F2 string
}
```
Then run the command:
```bash
go generate ./...
```
That's it. Examples can be found in the [testdata](https://github.com/slavaavr/go-struct-builder/tree/master/internal/service/testdata) folder.

## Required / Optional fields

- There are two types of struct fields: `required` and `optional`. By default, every pointer** value is optional and the rest of them are required.
To change the default behaviour add the appropriate struct tags:
```go
//go:generate gosb -source=input.go
type B struct {
	F1 *int `gosb:"required"`
	F2 string `gosb:"optional"`
}
```
Generated builder will check if `required` fields were provided.
- For a `private` struct a `private` builder will be generated. 
- If struct has `private` fields, along with the builder `getter methods` will be generated.

** and the `Option` type from`github.com/samber/mo` package.

## Flags

The `gosb` command is used to generate builder pattern for structs annotated with `go:generate gosb` comment.
It supports the following flags:

- `-source`: A file containing struct the builder must be generated for
- `-features`: Comma separated list of features:
    - `ptr`: Generates additional method for every pointer field without the pointer in the argument
    - `arr`: Generates additional method for every array field by using vararg in the argument
    - `opt`: Generates additional method for every `Option` field provided by the `github.com/samber/mo` library by unwrapping the `Option` type and setting a value directly

[ci-badge]:      https://github.com/slavaavr/go-struct-builder/actions/workflows/main.yaml/badge.svg
[ci-runs]:       https://github.com/slavaavr/go-struct-builder/actions