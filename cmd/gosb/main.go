package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/slavaavr/go-struct-builder/internal/labels"
	"github.com/slavaavr/go-struct-builder/internal/model"
	"github.com/slavaavr/go-struct-builder/internal/service"
)

var (
	source   = flag.String("source", "", "[Required] Input Go source file")
	features = flag.String("features", "", "[Optional] Comma separated list of features [ptr,arr,opt]")
)

func main() {
	flag.Parse()

	if *source == "" {
		log.Fatalf("source flag is not provided")
	}

	features, err := labels.ParseFeatures(*features)
	if err != nil {
		log.Fatalf("parsing features flag: %s", err)
	}

	srcDir, err := filepath.Abs(filepath.Dir(*source))
	if err != nil {
		log.Fatalf("getting the source directory: %s", err)
	}

	filename := path.Join(srcDir, *source)

	file, err := os.Open(filename)
	if err != nil {
		log.Fatalf("openning the file='%s': %s", filename, err)
	}

	p := service.NewParser()

	parsedFile, err := p.Parse(file)
	if err != nil {
		log.Fatalf("parsing the file='%s': %s", filename, err)
	}

	g := service.NewGenerator(features)

	data, err := g.Generate(parsedFile)
	if err != nil {
		log.Fatalf("generating builder: %s", err)
	}

	if err = saveOutput(data, parsedFile); err != nil {
		log.Fatalf("saving output file: %s", err)
	}
}

func saveOutput(data []byte, f *model.File) error {
	outputFile := path.Join(f.Path, getOutputFileName(f.Name))

	if err := os.WriteFile(outputFile, data, os.ModePerm); err != nil {
		return fmt.Errorf("writing to output file='%s': %w", outputFile, err)
	}

	return nil
}

func getOutputFileName(file string) string {
	return fmt.Sprintf("%s_builder.go", strings.TrimSuffix(file, ".go"))
}
