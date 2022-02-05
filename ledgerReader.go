package ledger

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	markerPrefix = ";__ledger_file"
)

var includedFiles = make(map[string]bool)

// NewLedgerReader reads a file and includes any files with include directives
// and returns the whole combined ledger as a buffer for parsing.
func NewLedgerReader(filename string) (*bytes.Buffer, error) {
	var buf bytes.Buffer

	err := includeFile(filename, &buf)
	return &buf, err
}

// includeFile reads filename into buf, adding special marker comments
// when there are step changes in file location due to 'include' directive.
func includeFile(filename string, buf *bytes.Buffer) error {
	if filename == "" {
		return errors.New("must specify filename")
	}
	filename = filepath.Clean(filename)
	lineNum := 0

	// check for include cyles
	if includedFiles[filename] {
		return fmt.Errorf("include cycle: '%s'", filename)
	}
	includedFiles[filename] = true

	defer delete(includedFiles, filename)

	f, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer f.Close()
	s := bufio.NewScanner(f)

	// mark the start of this file
	fmt.Fprintln(buf, marker(filename, lineNum))

	for s.Scan() {
		line := s.Text()

		if prefix, incname, found := strings.Cut(line, " "); found && prefix == "include" {
			// Resolve filepaths
			includedPath := filepath.Join(filename, "..", incname)
			includedPaths, err := filepath.Glob(includedPath)

			// Include all resolved filepaths
			for i := 0; i < len(includedPaths) && err == nil; i++ {
				if !includedFiles[includedPaths[i]] {
					err = includeFile(includedPaths[i], buf)
				}
			}
			if err != nil {
				return fmt.Errorf("%s:%d: %s", filename, lineNum, err.Error())
			}
			lineNum++

			// mark the resumption point for this file
			fmt.Fprintln(buf, marker(filename, lineNum))
		} else {
			fmt.Fprintln(buf, s.Text())
			lineNum++
		}
	}
	return nil
}

func marker(filename string, lineNum int) string {
	return fmt.Sprintf("%s*-*%s*-*%d", markerPrefix, filename, lineNum)
}

func parseMarker(s string) (string, int) {
	v := strings.Split(s, "*-*")
	lineNum, _ := strconv.Atoi(v[2])
	return v[1], lineNum
}
