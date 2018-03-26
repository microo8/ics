package ics

import (
	"os"
	"testing"
	"time"
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

func TestGetByDate(t *testing.T) {
	f, err := os.Open("data/ics_SK.ics")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	cal, err := Parse(f)
	if err != nil {
		t.Fatal(err)
	}
	events := cal.GetEventsByDate(time.Date(2018, 3, 30, 0, 0, 0, 0, time.UTC))
	if len(events) == 0 {
		t.Fatal("not found event")
	}
}
