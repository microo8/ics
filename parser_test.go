package ics

import (
	"fmt"
	"testing"
)

func TestParser(t *testing.T) {
	for f := range getTests() {
		defer f.Close()
		cal, err := Parse(f)
		if err != nil {
			t.Fatal(err)
		}
		fmt.Printf("%#v\n", cal)
	}
}
