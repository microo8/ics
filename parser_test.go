package ics

import (
	"testing"
)

func TestParser(t *testing.T) {
	for f := range getTests() {
		_, err := Parse(f)
		if err != nil {
			t.Fatal(f.Name(), err)
		}
		f.Close()
	}
}
