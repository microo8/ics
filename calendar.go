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
	TZID       *time.Location
	TZURL      *url.URL
	Daylight   *Daylight
}

//Daylight ...
type Daylight struct {
	TZOffsetFrom int
	TZOffsetTo   int
}

//Event represents the iCalendar event
type Event struct {
	Summary     string
	Description string
	UID         string
	Sequence    int
	Status      string
	Transp      string
	Rrule       *Rrule
	DTStart     time.Time
	DTEnd       time.Time
	DTStamp     time.Time
	Categories  []string
	Location    string
	Geo         GeoPoint
	URL         *url.URL
	Alarm       *Alarm
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
	ByDay      []time.Weekday
	ByMonthday []int
	ByYearday  []int
	ByWeekNo   []int
	ByMonth    []time.Month
	BySetPos   []int
	Wkst       *time.Weekday
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

func parseIcsDay(val string) (time.Weekday, error) {
	switch val {
	case "MO":
		return time.Monday, nil
	case "TU":
		return time.Tuesday, nil
	case "WE":
		return time.Wednesday, nil
	case "TH":
		return time.Thursday, nil
	case "FR":
		return time.Friday, nil
	case "SA":
		return time.Saturday, nil
	case "SU":
		return time.Sunday, nil
	default:
		return 0, fmt.Errorf("not valid Weekday value (%s)", val)
	}
}

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
