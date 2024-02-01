package service

import (
	"bytes"
	"errors"
	"fmt"
	"strings"

	toolsimports "golang.org/x/tools/imports"

	"github.com/slavaavr/go-struct-builder/internal/labels"
	"github.com/slavaavr/go-struct-builder/internal/model"
)

type Generator interface {
	Generate(f *model.File) ([]byte, error)
}

type generator struct {
	buf    bytes.Buffer
	indent string

	features []labels.Feature
}

func NewGenerator(
	features []labels.Feature,
) Generator {
	return &generator{
		buf:      bytes.Buffer{},
		indent:   "",
		features: features,
	}
}

func (g *generator) Generate(f *model.File) ([]byte, error) {
	if len(f.Structs) == 0 {
		return nil, errors.New("no structs provided for generator")
	}

	defer func() {
		g.buf.Reset()

		g.indent = ""
	}()

	g.pf("// Code generated by go-struct-builder. DO NOT EDIT.")
	g.pf("// Source: %v", f.Name)
	g.pf("")

	g.pf("package %v", f.Pkg)
	g.pf("")

	if isFileHasStructWithRequiredField(f) {
		f.Imports = append(f.Imports, model.Import{
			Value: `"errors"`,
			Alias: nil,
		})
	}

	if len(f.Imports) > 0 {
		g.pf("import (")
		g.in()

		for _, imp := range f.Imports {
			if imp.Alias != nil {
				g.pf("%s %s", *imp.Alias, imp.Value)
			} else {
				g.pf(imp.Value)
			}
		}

		g.out()
		g.pf(")")
		g.pf("")
	}

	for _, st := range f.Structs {
		if !st.Private {
			g.generateStructGetters(st)
		}

		g.generateBuilder(st)
		g.pf("")
		g.pf("")
	}

	res, err := toolsimports.Process("", g.buf.Bytes(), nil)
	if err != nil {
		return nil, fmt.Errorf("formatting generated code: %w", err)
	}

	return res, nil
}

func (g *generator) generateBuilder(st model.Struct) {
	builderName := g.getBuilderName(st.Name)
	requiredField2Index := g.getRequiredField2IndexMap(st)

	g.generateBuilderStruct(builderName, st)
	g.generateBuilderConstructor(builderName, requiredField2Index, st)
	g.generateBuilderMethods(builderName, requiredField2Index, st)
	g.generateBuildMethod(builderName, requiredField2Index, st)
}

func (g *generator) generateBuilderStruct(builderName string, st model.Struct) {
	g.pf("type %s struct {", builderName)
	g.in()
	g.pf("x *%s", st.Name)
	g.pf("mask []byte")
	g.out()
	g.pf("}")
	g.pf("")
}

func (g *generator) generateBuilderConstructor(
	builderName string,
	requiredField2Index map[model.Field]int,
	st model.Struct,
) {
	requiredFieldsMask := g.getRequiredFieldsMask(requiredField2Index, st)

	if st.Private {
		g.pf("func new%s() *%s {", makeStringCapital(builderName), builderName)
	} else {
		g.pf("func New%s() *%s {", builderName, builderName)
	}

	g.in()

	if isStructHasRequiredField(st) {
		g.pf("/**")
		g.in()
		g.pf("Required fields:")

		for _, fld := range st.Fields {
			if fld.Required {
				g.pf("%d) %s %s", requiredField2Index[fld], fld.Name, fld.Type.Name)
			}
		}

		g.out()
		g.pf("*/")
		g.pf("")
	}

	g.pf("return &%s{", builderName)
	g.in()
	g.pf("x: new(%s),", st.Name)
	g.pf("mask: []byte{%s},", mapBytesToString(requiredFieldsMask))
	g.out()
	g.pf("}")
	g.out()
	g.pf("}")
	g.pf("")
}

func (g *generator) generateBuilderMethods(
	builderName string,
	requiredField2Index map[model.Field]int,
	st model.Struct,
) {
	for _, fld := range st.Fields {
		g.generateBuilderMethodByField(builderName, requiredField2Index, fld)

		switch {
		case fld.Type.Info == model.TypeInfoPointer && g.hasFeature(labels.FeatureFlagPtr):
			g.generateBuilderMethodFeaturePtr(builderName, requiredField2Index, fld)

		case fld.Type.Info == model.TypeInfoArray && g.hasFeature(labels.FeatureFlagArr):
			g.generateBuilderMethodFeatureArr(builderName, requiredField2Index, fld)

		case fld.Type.Info == model.TypeInfoOption && g.hasFeature(labels.FeatureFlagOpt):
			g.generateBuilderMethodFeatureOpt(builderName, requiredField2Index, fld)
		}
	}
}

const (
	setMaskBitPattern = "b.mask[%d/8] &= ^uint8(1 << %d %% 8)"
)

func (g *generator) generateBuilderMethodByField(
	builderName string,
	requiredField2Index map[model.Field]int,
	fld model.Field,
) {
	g.pf("func (b *%s) Set%s(v %s) *%s {", builderName, g.getMethodName(fld), fld.Type.Name, builderName)
	g.in()
	g.pf("b.x.%s = v", fld.Name)

	if fld.Required {
		idx := requiredField2Index[fld]
		g.pf(setMaskBitPattern, idx, idx)
	}

	g.pf("return b")
	g.out()
	g.pf("}")
	g.pf("")
}

func (g *generator) generateBuilderMethodFeaturePtr(
	builderName string,
	requiredField2Index map[model.Field]int,
	fld model.Field,
) {
	fldType := strings.TrimPrefix(fld.Type.Name, "*")

	g.pf("func (b *%s) Set%sV(v %s) *%s {", builderName, g.getMethodName(fld), fldType, builderName)
	g.in()
	g.pf("b.x.%s = &v", fld.Name)

	if fld.Required {
		idx := requiredField2Index[fld]
		g.pf(setMaskBitPattern, idx, idx)
	}

	g.pf("return b")
	g.out()
	g.pf("}")
	g.pf("")
}

func (g *generator) generateBuilderMethodFeatureArr(
	builderName string,
	requiredField2Index map[model.Field]int,
	fld model.Field,
) {
	fldType := strings.TrimPrefix(fld.Type.Name, "[]")

	g.pf("func (b *%s) Set%sV(v ...%s) *%s {", builderName, g.getMethodName(fld), fldType, builderName)
	g.in()
	g.pf("b.x.%s = append(b.x.%s, v...)", fld.Name, fld.Name)

	if fld.Required {
		idx := requiredField2Index[fld]
		g.pf(setMaskBitPattern, idx, idx)
	}

	g.pf("return b")
	g.out()
	g.pf("}")
	g.pf("")
}

func (g *generator) generateBuilderMethodFeatureOpt(
	builderName string,
	requiredField2Index map[model.Field]int,
	fld model.Field,
) {
	fldType := strings.TrimPrefix(fld.Type.Name, moOptionType+"[")
	fldType = strings.TrimSuffix(fldType, "]")

	g.pf("func (b *%s) Set%sV(v %s) *%s {", builderName, g.getMethodName(fld), fldType, builderName)
	g.in()
	g.pf("b.x.%s = mo.Some(v)", fld.Name)

	if fld.Required {
		idx := requiredField2Index[fld]
		g.pf(setMaskBitPattern, idx, idx)
	}

	g.pf("return b")
	g.out()
	g.pf("}")
	g.pf("")
}

func (g *generator) generateBuildMethod(
	builderName string,
	requiredField2Index map[model.Field]int,
	st model.Struct,
) {
	if isStructHasRequiredField(st) {
		g.pf("func (b *%s) Build() (*%s, error) {", builderName, st.Name)
		g.in()

		for _, fld := range st.Fields {
			if fld.Required {
				idx := requiredField2Index[fld]
				g.pf("if (b.mask[%d/8] & (1 << %d %% 8)) != 0 {", idx, idx)
				g.in()
				g.pf(`return nil, errors.New("%s.%s field is not provided")`, st.Name, fld.Name)
				g.out()
				g.pf("}")
				g.pf("")
			}
		}

		g.pf("return b.x, nil")
		g.out()
	} else {
		g.pf("func (b *%s) Build() *%s {", builderName, st.Name)
		g.in()
		g.pf("return b.x")
		g.out()
	}

	g.pf("}")
}

const bitsInByte = 8

func (g *generator) getRequiredFieldsMask(
	requiredField2Index map[model.Field]int,
	st model.Struct,
) []byte {
	requiredFieldsCount := 0

	for _, fld := range st.Fields {
		if fld.Required {
			requiredFieldsCount++
		}
	}

	res := make([]byte, requiredFieldsCount/bitsInByte+1)

	for _, fld := range st.Fields {
		if fld.Required {
			idx := requiredField2Index[fld]
			res[idx/bitsInByte] |= 1 << idx % bitsInByte
		}
	}

	return res
}

func (g *generator) getRequiredField2IndexMap(st model.Struct) map[model.Field]int {
	i := 0
	res := make(map[model.Field]int)

	for _, fld := range st.Fields {
		if fld.Required {
			i++

			res[fld] = i
		}
	}

	return res
}

func (g *generator) getMethodName(fld model.Field) string {
	return makeStringCapital(fld.Name)
}

func (g *generator) generateStructGetters(st model.Struct) {
	for _, fld := range st.Fields {
		if fld.Private {
			g.pf("func (t *%s) %s() %s {", st.Name, makeStringCapital(fld.Name), fld.Type.Name)
			g.in()
			g.pf("return t.%s", fld.Name)
			g.out()
			g.pf("}")
			g.pf("")
		}
	}
}

func (g *generator) hasFeature(f labels.Feature) bool {
	for _, tmp := range g.features {
		if tmp == f {
			return true
		}
	}

	return false
}

func (g *generator) getBuilderName(structName string) string {
	return fmt.Sprintf("%sBuilder", structName)
}

func (g *generator) pf(format string, args ...interface{}) {
	_, err := fmt.Fprintf(&g.buf, g.indent+format+"\n", args...)
	if err != nil {
		panic(fmt.Sprintf("error writing to the buf: %s", err))
	}
}

func (g *generator) in() {
	g.indent += "\t"
}

func (g *generator) out() {
	if len(g.indent) > 0 {
		g.indent = g.indent[0 : len(g.indent)-1]
	}
}

func mapBytesToString(bs []byte) string {
	var res strings.Builder

	for _, b := range bs {
		res.WriteString(fmt.Sprintf(",0x%x", b))
	}

	return res.String()[1:]
}

func isFileHasStructWithRequiredField(f *model.File) bool {
	for _, s := range f.Structs {
		if isStructHasRequiredField(s) {
			return true
		}
	}

	return false
}

func isStructHasRequiredField(st model.Struct) bool {
	for _, fld := range st.Fields {
		if fld.Required {
			return true
		}
	}

	return false
}

func makeStringCapital(s string) string {
	return strings.ToUpper(s[:1]) + s[1:]
}