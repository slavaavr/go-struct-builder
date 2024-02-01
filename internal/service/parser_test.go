package service

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/slavaavr/go-struct-builder/internal/model"
)

func ptr[T any](t T) *T {
	return &t
}

func TestParser_Parse(t *testing.T) {
	cases := []struct {
		name        string
		source      string
		expected    *model.File
		expectedErr error
	}{
		{
			name:   "empty file",
			source: `package test42`,
			expected: &model.File{
				Name:    "x",
				Path:    "x",
				Pkg:     "test42",
				Imports: []model.Import{},
				Structs: []model.Struct{},
			},
			expectedErr: nil,
		},
		{
			name: "no go:generate comment",
			source: `
			package main

			type A struct {
				F1 int
			}`,
			expected: &model.File{
				Name:    "x",
				Path:    "x",
				Pkg:     "main",
				Imports: []model.Import{},
				Structs: []model.Struct{},
			},
			expectedErr: nil,
		},
		{
			name: "type aliases",
			source: `
			package main
			import t1 "time"
			
			//go:generate gosb -source=input.go
			type A struct {
				F1 t1.Time
			}`,
			expected: &model.File{
				Name: "x",
				Path: "x",
				Pkg:  "main",
				Imports: []model.Import{
					{
						Value: `"time"`,
						Alias: ptr("t1"),
					},
				},
				Structs: []model.Struct{
					{
						Name:    "A",
						Private: false,
						Fields: []model.Field{
							{
								Name: "F1",
								Type: model.FieldType{
									Name: "t1.Time",
									Info: model.TypeInfoOther,
								},
								Private:  false,
								Required: true,
							},
						},
					},
				},
			},
			expectedErr: nil,
		},
		{
			name: "eliminate unused import",
			source: `
			package main

			import "time"
			import "go/ast"

			//go:generate gosb -source=input.go
			type A struct {
				F1 time.Time
			}`,
			expected: &model.File{
				Name: "xxx",
				Path: "xxx",
				Pkg:  "main",
				Imports: []model.Import{
					{
						Value: `"time"`,
						Alias: nil,
					},
				},
				Structs: []model.Struct{
					{
						Name:    "A",
						Private: false,
						Fields: []model.Field{
							{
								Name: "F1",
								Type: model.FieldType{
									Name: "time.Time",
									Info: model.TypeInfoOther,
								},
								Private:  false,
								Required: true,
							},
						},
					},
				},
			},
			expectedErr: nil,
		},
		{
			name: "public/private fields",
			source: `
			package main

			//go:generate gosb -source=input.go
			type A struct {
				F1 int
				f2 string
			}`,
			expected: &model.File{
				Name:    "x",
				Path:    "x",
				Pkg:     "main",
				Imports: []model.Import{},
				Structs: []model.Struct{
					{
						Name:    "A",
						Private: false,
						Fields: []model.Field{
							{
								Name: "F1",
								Type: model.FieldType{
									Name: "int",
									Info: model.TypeInfoOther,
								},
								Private:  false,
								Required: true,
							},
							{
								Name: "f2",
								Type: model.FieldType{
									Name: "string",
									Info: model.TypeInfoOther,
								},
								Private:  true,
								Required: true,
							},
						},
					},
				},
			},
			expectedErr: nil,
		},
		{
			name: "required/optional fields",
			source: `
			package main
			
			//go:generate gosb -source=input.go
			type A struct {
				F1 int 
				F2 *int
				F3 int` + "`gosb:\"optional\"`" + `
				F4 *int` + "`gosb:\"required\"`" + `
			}`,
			expected: &model.File{
				Name:    "x",
				Path:    "x",
				Pkg:     "main",
				Imports: []model.Import{},
				Structs: []model.Struct{
					{
						Name:    "A",
						Private: false,
						Fields: []model.Field{
							{
								Name: "F1",
								Type: model.FieldType{
									Name: "int",
									Info: model.TypeInfoOther,
								},
								Private:  false,
								Required: true,
							},
							{
								Name: "F2",
								Type: model.FieldType{
									Name: "*int",
									Info: model.TypeInfoPointer,
								},
								Private:  false,
								Required: false,
							},
							{
								Name: "F3",
								Type: model.FieldType{
									Name: "int",
									Info: model.TypeInfoOther,
								},
								Private:  false,
								Required: false,
							},
							{
								Name: "F4",
								Type: model.FieldType{
									Name: "*int",
									Info: model.TypeInfoPointer,
								},
								Private:  false,
								Required: true,
							},
						},
					},
				},
			},
			expectedErr: nil,
		},
		{
			name: "feature flags",
			source: `
			package main
			
			//go:generate gosb -source=input.go -features=ptr,arr,opt
			type A struct {
				F1 **int 
				F2 []int
				F3 mo.Option[int]
				F4 mo.Option[int] ` + "`gosb:\"required\"`" + `
			}`,
			expected: &model.File{
				Name:    "x",
				Path:    "x",
				Pkg:     "main",
				Imports: []model.Import{},
				Structs: []model.Struct{
					{
						Name:    "A",
						Private: false,
						Fields: []model.Field{
							{
								Name: "F1",
								Type: model.FieldType{
									Name: "**int",
									Info: model.TypeInfoPointer,
								},
								Private:  false,
								Required: false,
							},
							{
								Name: "F2",
								Type: model.FieldType{
									Name: "[]int",
									Info: model.TypeInfoArray,
								},
								Private:  false,
								Required: true,
							},
							{
								Name: "F3",
								Type: model.FieldType{
									Name: "mo.Option[int]",
									Info: model.TypeInfoOption,
								},
								Private:  false,
								Required: false,
							},
							{
								Name: "F4",
								Type: model.FieldType{
									Name: "mo.Option[int]",
									Info: model.TypeInfoOption,
								},
								Private:  false,
								Required: true,
							},
						},
					},
				},
			},
			expectedErr: nil,
		},
		{
			name: "struct not found error",
			source: `
			package main
			
			//go:generate gosb -source=input.go
			type A interface {
				M1()
			}`,
			expected:    nil,
			expectedErr: fmt.Errorf("parsing struct: %w", errors.New("struct not found")),
		},
	}

	for _, c := range cases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			f, err := os.CreateTemp("", "")
			require.NoError(t, err, "creating a temp file")

			defer func() {
				assert.NoError(t, f.Close(), "closing the temp file")
				assert.NoError(t, os.Remove(f.Name()), "deleting the temp file")
			}()

			_, _ = f.WriteString(c.source)
			_, _ = f.Seek(0, 0)

			s := NewParser()

			// because of the dynamic file generation, rewrite the name and the path
			if c.expected != nil {
				c.expected.Name = filepath.Base(f.Name())
				c.expected.Path = filepath.Dir(f.Name())
			}

			actual, actualErr := s.Parse(f)
			require.Equal(t, c.expectedErr, actualErr, "errors are not equal")
			assert.Equal(t, c.expected, actual, "values are not equal")
		})
	}
}
