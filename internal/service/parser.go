package service

import (
	"errors"
	"fmt"
	"go/ast"
	goparser "go/parser"
	gotoken "go/token"
	gotypes "go/types"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/slavaavr/go-struct-builder/internal/labels"
	"github.com/slavaavr/go-struct-builder/internal/model"
)

type Parser interface {
	Parse(f *os.File) (*model.File, error)
}

type parser struct {
	commentLabels []string
}

func NewParser() Parser {
	return &parser{
		commentLabels: []string{labels.Gosb, labels.GenerateCmd},
	}
}

func (s *parser) Parse(f *os.File) (*model.File, error) {
	filename := f.Name()
	fileSet := gotoken.NewFileSet()

	file, err := goparser.ParseFile(fileSet, "", f, goparser.ParseComments)
	if err != nil {
		return nil, fmt.Errorf("parsing source file='%v': %w", filename, err)
	}

	var (
		imports = make([]model.Import, 0)
		structs = make([]model.Struct, 0)
	)

	for _, decl := range file.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}

		if gd.Tok == gotoken.IMPORT {
			imports = append(imports, s.parseImports(gd)...)
		} else if gd.Tok == gotoken.TYPE {
			if s.containCommentLabels(gd) {
				res, err := s.parseStruct(gd)
				if err != nil {
					return nil, fmt.Errorf("parsing struct: %w", err)
				}

				structs = append(structs, *res)
			}
		}
	}

	return &model.File{
		Name:    filepath.Base(filename),
		Path:    filepath.Dir(filename),
		Pkg:     file.Name.Name,
		Imports: s.updateImports(structs, imports),
		Structs: structs,
	}, nil
}

func (s *parser) updateImports(structs []model.Struct, imports []model.Import) []model.Import {
	newImports := make([]model.Import, 0, len(imports))

	for _, imp := range imports {
		if s.isImportUsed(imp, structs) {
			newImports = append(newImports, imp)
		}
	}

	return newImports
}

func (s *parser) isImportUsed(imp model.Import, structs []model.Struct) bool {
	var pkgPrefix string

	if imp.Alias != nil {
		pkgPrefix = *imp.Alias
	} else {
		value := strings.Trim(imp.Value, "\"")
		idx := strings.LastIndexByte(value, '/')
		if idx != -1 {
			pkgPrefix = value[idx+1:]
		} else {
			pkgPrefix = value
		}
	}

	pkgPrefix += "."

	for _, st := range structs {
		for _, fld := range st.Fields {
			if strings.Contains(fld.Type.Name, pkgPrefix) {
				return true
			}
		}
	}

	return false
}

func (s *parser) parseStruct(decl *ast.GenDecl) (*model.Struct, error) {
	for _, spec := range decl.Specs {
		ts, ok := spec.(*ast.TypeSpec)
		if !ok {
			continue
		}

		typ, ok := ts.Type.(*ast.StructType)
		if !ok {
			continue
		}

		structName := ts.Name.Name
		fields := make([]model.Field, 0)

		if typ.Fields != nil {
			for _, f := range typ.Fields.List {
				field, err := s.parseField(f)
				if err != nil {
					return nil, fmt.Errorf("parsing %s.%s field", structName, f.Names[0].Name)
				}

				fields = append(fields, *field)
			}
		}

		return &model.Struct{
			Name:    structName,
			Private: !isStringCapital(structName),
			Fields:  fields,
		}, nil
	}

	return nil, errors.New("struct not found")
}

const (
	moOptionType = "mo.Option"
)

func (s *parser) parseField(f *ast.Field) (*model.Field, error) {
	fieldType := gotypes.ExprString(f.Type)
	typeInfo := model.TypeInfoOther

	switch {
	case fieldType[0] == '*':
		typeInfo = model.TypeInfoPointer

	case strings.HasPrefix(fieldType, "[]"):
		typeInfo = model.TypeInfoArray

	case strings.HasPrefix(fieldType, moOptionType):
		typeInfo = model.TypeInfoOption
	}

	required := true

	if typeInfo == model.TypeInfoPointer || typeInfo == model.TypeInfoOption {
		required = false
	}

	var fieldTag string

	if f.Tag != nil {
		value, err := strconv.Unquote(f.Tag.Value)
		if err != nil {
			return nil, fmt.Errorf("unquote field tag")
		}

		fieldTag = reflect.StructTag(value).Get(labels.Gosb)
	}

	switch fieldTag {
	case labels.StructTagRequired:
		required = true

	case labels.StructTagOptional:
		required = false
	}

	fieldName := s.getFieldName(f.Names, fieldType)

	return &model.Field{
		Name: fieldName,
		Type: model.FieldType{
			Name: fieldType,
			Info: typeInfo,
		},
		Private:  !isStringCapital(fieldName),
		Required: required,
	}, nil
}

var fieldNameFromTypeRegexp = regexp.MustCompile(`^\*?(?:\w+\.)?(\w+)`)

func (s *parser) getFieldName(names []*ast.Ident, typ string) string {
	var fieldName string

	if names == nil {
		// embedded field
		fieldName = fieldNameFromTypeRegexp.FindStringSubmatch(typ)[1]
	} else {
		fieldName = names[0].Name
	}

	return fieldName
}

func (s *parser) parseImports(decl *ast.GenDecl) []model.Import {
	imports := make([]model.Import, 0)

	for _, spec := range decl.Specs {
		impspec, ok := spec.(*ast.ImportSpec)
		if !ok {
			continue
		}

		var alias *string
		if impspec.Name != nil {
			alias = &impspec.Name.Name
		}

		imports = append(imports, model.Import{
			Value: impspec.Path.Value,
			Alias: alias,
		})
	}

	return imports
}

func (s *parser) containCommentLabels(decl *ast.GenDecl) bool {
	if decl.Doc == nil {
		return false
	}

	for _, comment := range decl.Doc.List {
		if containsAll(comment.Text, s.commentLabels) {
			return true
		}
	}

	return false
}

func containsAll(text string, ss []string) bool {
	for _, s := range ss {
		if !strings.Contains(text, s) {
			return false
		}
	}

	return true
}

func isStringCapital(s string) bool {
	return makeStringCapital(s) == s
}
