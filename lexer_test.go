package ics

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func getTests() chan *os.File {
	ch := make(chan *os.File)
	go func() {
		files, _ := ioutil.ReadDir("data")
		for _, f := range files {
			if !strings.HasSuffix(f.Name(), ".ics") {
				continue
			}
			icsFile, err := os.Open(filepath.Join("data", f.Name()))
			if err != nil {
				panic(err)
			}
			ch <- icsFile
		}
		close(ch)
	}()
	return ch
}

func TestLexer(t *testing.T) {
	for f := range getTests() {
		defer f.Close()
		l := lex(f)
		for item := range l.items {
			if item.typ == itemError {
				t.Fatal(item)
			}
		}
	}
}
