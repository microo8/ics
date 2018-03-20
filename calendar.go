package ics

import (
	"fmt"
	"net/url"
	"time"
)

//Calendar represents the iCalendar data
type Calendar struct {
	Properties map[string]string //Non-Standard Properties, Any property name with a "X-" prefix
	Version    string
	Method     string
	ProdID     string
	Calscale   string
	TimeZone   *TimeZone
	Events     []*Event
}

//TimeZone provide a grouping of component properties that defines a time zone.
type TimeZone struct {
	Properties map[string]string //Non-Standard Properties, Any property name with a "X-" prefix
	TZID       string
	TZURL      *url.URL
	Daylight   *Standard
	Standard   *Standard
}

//Standard ...
type Standard struct {
	TZOffsetFrom int
	TZOffsetTo   int
	TZName       string
	DTStart      time.Time
	Rrule        *Rrule
	Rdate        string //TODO
}

//Class the classification enum
type Class int

//classification enum
const (
	ClassPublic Class = iota
	ClassPrivate
	ClassConfidential
)

//Event represents the iCalendar event
type Event struct {
	Properties   map[string]string //Non-Standard Properties, Any property name with a "X-" prefix
	Class        Class
	Created      time.Time
	LastModified time.Time
	Summary      string
	Description  string
	UID          string
	Sequence     int
	Status       string
	Transp       string
	Rrule        *Rrule
	ExDate       []time.Time
	DTStart      time.Time
	DTEnd        time.Time
	DTStamp      time.Time
	Categories   []string
	Location     string
	Geo          GeoPoint
	URL          *url.URL
	Alarm        *Alarm
	Organizer    *Attendee
	Attendee     *Attendee
	Participant  *Attendee
}

//Attendee ...
type Attendee struct {
	Parameters map[string]string
	Value      string
}

//Rrule ...
type Rrule struct {
	Freq       Frequency
	Until      *time.Time
	Count      *int
	Interval   *int
	BySecond   []int
	ByMinute   []int
	ByHour     []int
	ByDay      []WDay
	ByMonthday []int
	ByYearday  []int
	ByWeekNo   []int
	ByMonth    []time.Month
	BySetPos   []int
	Wkst       *time.Weekday
}

//WDay ...
type WDay struct {
	Num     int
	Weekday time.Weekday
}

//Alarm ...
type Alarm struct {
	Trigger     string
	Repeat      int
	Duration    time.Duration
	Action      Action
	Description string
	Attendee    string
	Summary     string
	Attach      string
}

//Action types
type Action int

//Action types
const (
	ActionAudio Action = iota
	ActionDisplay
	ActionEmail
)

//Frequency rule part identifies the type of recurrence rule
type Frequency int

func (f Frequency) String() string {
	switch f {
	case FrequencySecondly:
		return "SECONDLY"
	case FrequencyMinutely:
		return "MINUTELY"
	case FrequencyHourly:
		return "HOURLY"
	case FrequencyDaily:
		return "DAILY"
	case FrequencyWeekly:
		return "WEEKLY"
	case FrequencyMonthly:
		return "MONTHLY"
	case FrequencyYearly:
		return "YEARLY"
	}
	return "not supported Frequency"
}

func parseFrequency(val string) (Frequency, error) {
	switch val {
	case "SECONDLY":
		return FrequencySecondly, nil
	case "MINUTELY":
		return FrequencyMinutely, nil
	case "HOURLY":
		return FrequencyHourly, nil
	case "DAILY":
		return FrequencyDaily, nil
	case "WEEKLY":
		return FrequencyWeekly, nil
	case "MONTHLY":
		return FrequencyMonthly, nil
	case "YEARLY":
		return FrequencyYearly, nil
	}
	return 0, fmt.Errorf("not valid Frequency value (%s)", val)
}

//Frequency values
const (
	FrequencySecondly Frequency = iota
	FrequencyMinutely
	FrequencyHourly
	FrequencyDaily
	FrequencyWeekly
	FrequencyMonthly
	FrequencyYearly
)

//GeoPoint represents the latitude and longitude coordinates
type GeoPoint struct {
	Latitude, Longitude float64
}
