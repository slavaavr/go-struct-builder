package service

import (
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/slavaavr/go-struct-builder/internal/labels"
)

// nolint: goconst
func TestGenerator_Generate(t *testing.T) {
	makeActualGoldenFile := func(source string, actual []byte) string {
		return fmt.Sprintf("--- source code ---\n%s\n\n\n--- generated code ---\n\n%s", source, actual)
	}

	cases := []struct {
		name        string
		source      string
		features    []labels.Feature
		expectedErr error
	}{
		{
			name:        "empty file",
			source:      `package main`,
			features:    nil,
			expectedErr: errors.New("no structs provided for generator"),
		},
		{
			name: "no go:generate comment",
			source: `
			package main

			type A struct {
				F1 int
			}`,
			features:    nil,
			expectedErr: errors.New("no structs provided for generator"),
		},
		{
			name: "basic example",
			source: `
			package main

			//go:generate gosb -source=input.go
			type A struct {
				F1 int
				F2 string
			}`,
			features:    nil,
			expectedErr: nil,
		},
		{
			name: "private fields",
			source: `
			package main

			//go:generate gosb -source=input.go
			type A struct {
				f1 int
				f2 string
			}`,
			features:    nil,
			expectedErr: nil,
		},
		{
			name: "field tags",
			source: `
			package main
			
			//go:generate gosb -source=input.go
			type A struct {
				F1 int 
				F2 *int
				F3 int` + "`gosb:\"optional\"`" + `
				F4 *int` + "`gosb:\"required\"`" + `
			}`,
			features:    nil,
			expectedErr: nil,
		},
		{
			name: "two builders",
			source: `
			package main

			//go:generate gosb -source=input.go
			type A struct {
				F1 int
			}

			type B struct {
				F2 int
			}

			//go:generate gosb -source=input.go
			type C struct {
				F3 int
			}`,
			features:    nil,
			expectedErr: nil,
		},
		{
			name: "optional fields",
			source: `
			package main

			//go:generate gosb -source=input.go
			type A struct {
				F1 *int
				F2 *string
				F3 *float64
			}`,
			features:    nil,
			expectedErr: nil,
		},
		{
			name: "generics",
			source: `
			package main
			
			type genericType[T any] struct {
				Field T
			}
	
			//go:generate gosb -source=input.go
			type A struct {
				F1 string
				F2 genericType[string]
				F3 genericType[genericType[string]]
			}`,
			features:    nil,
			expectedErr: nil,
		},
		{
			name: "features",
			source: `
			package main

			import "github.com/samber/mo"
			import "time"
	
			//go:generate gosb -source=input.go -features=ptr,arr,opt
			type A struct {
				F1 *time.Time
				F2 []int
				F3 mo.Option[int]
				F4 mo.Option[int] ` + "`gosb:\"required\"`" + `
			}`,
			features: []labels.Feature{
				labels.FeatureFlagPtr,
				labels.FeatureFlagArr,
				labels.FeatureFlagOpt,
			},
			expectedErr: nil,
		},
		{
			name: "private struct",
			source: `
			package main

			//go:generate gosb -source=input.go
			type a struct {
				f1 int
				f2 string
			}`,
			features:    nil,
			expectedErr: nil,
		},
		{
			name: "struct embedding",
			source: `
			package main

			import "github.com/samber/mo"
			import "time"
	
			//go:generate gosb -source=input.go
			type A struct {
				B
				*C
				mo.Option[int]
				*mo.Future[int]
				time.Time
				F1 int
			}

			type B struct {
				F1 int
			}

			type C struct {
				F1 int
			}`,
			features:    nil,
			expectedErr: nil,
		},
		{
			name: "type alias",
			source: `
			package main
			import t1 "time"
			
			//go:generate gosb -source=input.go
			type A struct {
				F1 t1.Time
			}`,
			features:    nil,
			expectedErr: nil,
		},
		{
			name: "unused import",
			source: `
			package main
		
			import t1 "time"
			import t2 "time"

			//go:generate gosb -source=input.go
			type A struct {
				F1 t1.Time
			}

			type B struct {
				F2 t2.Time
			}`,
			features:    nil,
			expectedErr: nil,
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

			g := NewGenerator(c.features)

			parsedFile, err := NewParser().Parse(f)
			require.NoError(t, err)

			parsedFile.Name = "input.go"

			data, err := g.Generate(parsedFile)
			if c.expectedErr != nil {
				require.Equal(t, c.expectedErr, err)
			} else {
				require.NoError(t, err)
				actual := makeActualGoldenFile(c.source, data)
				expected := goldenFile(t, c.name, actual)
				assert.Equal(t, expected, actual)
			}
		})
	}
}
