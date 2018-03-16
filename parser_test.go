package ics

import (
	"fmt"
	"strings"
	"testing"
)

func TestParser(t *testing.T) {
	r := strings.NewReader(testICS)
	cal, err := Parse(r)
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("%#v\n", cal)
}
