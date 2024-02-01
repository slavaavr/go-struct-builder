package service

import (
	"flag"
	"io"
	"os"
	"path"
	"strings"
	"testing"
)

var updateGolden = flag.Bool("update_golden", false, "update .golden files")

func goldenFile(t *testing.T, file string, actual string) string {
	t.Helper()

	filePath := path.Join("testdata", strings.ReplaceAll(file, " ", "_")+".golden")

	var mode int
	if *updateGolden {
		mode = os.O_RDWR | os.O_CREATE | os.O_TRUNC
	} else {
		mode = os.O_RDONLY
	}

	f, err := os.OpenFile(filePath, mode, 0644)
	if err != nil {
		t.Fatalf("error opening the file %s: %s", filePath, err)
	}

	defer func() {
		_ = f.Close()
	}()

	if *updateGolden {
		if _, err := f.WriteString(actual); err != nil {
			t.Fatalf("error writing to file %s: %s", filePath, err)
		}

		t.Logf("golden file %s has been updated", filePath)

		return actual
	}

	content, err := io.ReadAll(f)
	if err != nil {
		t.Fatalf("error reading the file %s: %s", filePath, err)
	}

	return string(content)
}
