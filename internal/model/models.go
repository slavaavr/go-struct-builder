package model

type File struct {
	Name    string
	Path    string
	Pkg     string
	Imports []Import
	Structs []Struct
}

type Import struct {
	Value string
	Alias *string
}

type Struct struct {
	Name    string
	Private bool
	Fields  []Field
}

type Field struct {
	Name     string
	Type     FieldType
	Private  bool
	Required bool
}

type FieldType struct {
	Name string
	Info TypeInfo
}

type TypeInfo int

const (
	TypeInfoOther TypeInfo = iota
	TypeInfoArray
	TypeInfoPointer
	TypeInfoOption
)
