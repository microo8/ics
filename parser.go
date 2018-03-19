package ics

import (
	"fmt"
	"io"
	"net/url"
	"strconv"
	"strings"
	"time"
)

//Parse parses the ics input and returns a Calendar struct
func Parse(r io.Reader) (*Calendar, error) {
	s := &scanner{l: lex(r)}
	cal, err := parseCalendar(s)
	if err != nil {
		return nil, fmt.Errorf("error parsing Calendar: %s", err)
	}
	return cal, nil
}

type scanner struct {
	curItem  item
	prevItem *item
	l        *lexer
}

func (s *scanner) next() item {
	if s.prevItem == nil {
		s.curItem = <-s.l.items
		fmt.Println(s.curItem)
		return s.curItem
	}
	i := *s.prevItem
	s.prevItem = nil
	fmt.Println(i)
	return i
}

func (s *scanner) backup() {
	s.prevItem = &s.curItem
}

func (s *scanner) stringProperty(name string) (string, error) {
	if i := s.next(); i.typ != itemColon {
		return "", fmt.Errorf("unexpected (%s) after %s, expected colon (:)", name, i.val)
	}
	i := s.next()
	if i.typ != itemString {
		return "", fmt.Errorf("unexpected (%s) after %s:, expected string value", name, i.val)
	}
	return i.val, nil
}

func (s *scanner) xProperty() (string, string, error) {
	i := s.next()
	propName := i.val[2:]
	if i := s.next(); i.typ != itemColon {
		return "", "", fmt.Errorf("unexpected (%s) after X, expected colon (:)", i.val)
	}
	i = s.next()
	if i.typ != itemString {
		return "", "", fmt.Errorf("unexpected (%s) after X, expected string value", i.val)
	}
	return propName, i.val, nil
}

func parseCalendar(s *scanner) (*Calendar, error) {
	if i := s.next(); i.typ != itemBegin {
		return nil, fmt.Errorf("iCalendar must start with BEGIN:VCALENDAR, not (%s)", i.val)
	}
	if i := s.next(); i.typ != itemColon {
		return nil, fmt.Errorf("iCalendar must start with BEGIN:VCALENDAR, not (%s)", i.val)
	}
	if i := s.next(); i.typ != itemVCalendar {
		return nil, fmt.Errorf("iCalendar must start with BEGIN:VCALENDAR, not (%s)", i.val)
	}
	cal := &Calendar{}
	for i := s.next(); i.typ != itemEOF; i = s.next() {
		switch i.typ {
		case itemError:
			return nil, fmt.Errorf(i.val)
		case itemEnd:
			if i := s.next(); i.typ != itemColon {
				return nil, fmt.Errorf("unexpected (%s) after END, expected colon (:)", i.val)
			}
			if i := s.next(); i.typ != itemVCalendar {
				return nil, fmt.Errorf("unexpected (%s) after END:, expected VCALENDAR", i.val)
			}
			return cal, nil
		case itemVersion:
			val, err := s.stringProperty("VERSION")
			if err != nil {
				return nil, err
			}
			cal.Version = val
		case itemProdID:
			val, err := s.stringProperty("PRODID")
			if err != nil {
				return nil, err
			}
			cal.ProdID = val
		case itemCalscale:
			val, err := s.stringProperty("CALSCALE")
			if err != nil {
				return nil, err
			}
			cal.Calscale = val
		case itemMethod:
			val, err := s.stringProperty("METHOD")
			if err != nil {
				return nil, err
			}
			cal.Method = val
		case itemX:
			s.backup()
			propName, val, err := s.xProperty()
			if err != nil {
				return nil, err
			}
			if cal.Properties == nil {
				cal.Properties = make(map[string]string)
			}
			cal.Properties[propName] = val
		case itemBegin:
			if i := s.next(); i.typ != itemColon {
				return nil, fmt.Errorf("unexpected (%s) after BEGIN, expected colon (:)", i.val)
			}
			i := s.next()
			switch i.typ {
			case itemVEvent:
				e, err := parseEvent(s)
				if err != nil {
					return nil, fmt.Errorf("error parsing VEVENT: %s", err)
				}
				cal.Events = append(cal.Events, e)
			case itemVTimeZone:
				tz, err := parseTimeZone(s)
				if err != nil {
					return nil, fmt.Errorf("error parsing VTIMEZONE: %s", err)
				}
				cal.TimeZone = tz
			default:
				return nil, fmt.Errorf("unexpected (%s) after BEGIN:, expected VEVENT or VTIMEZONE", i.val)
			}
		default:
			return nil, fmt.Errorf("unexpected item (%s) in VCALENDAR", i)
		}
	}
	return cal, nil
}

func parseEvent(s *scanner) (*Event, error) {
	e := &Event{}
	for i := s.next(); ; i = s.next() {
		switch i.typ {
		case itemEnd:
			if i := s.next(); i.typ != itemColon {
				return nil, fmt.Errorf("unexpected (%s) after END, expected colon (:)", i.val)
			}
			if i := s.next(); i.typ != itemVEvent {
				return nil, fmt.Errorf("unexpected (%s) after END:, expected VEVENT", i.val)
			}
			return e, nil
		case itemSummary:
			val, err := s.stringProperty("SUMMARY")
			if err != nil {
				return nil, err
			}
			e.Summary = val
		case itemUID:
			val, err := s.stringProperty("UID")
			if err != nil {
				return nil, err
			}
			e.UID = val
		case itemStatus:
			val, err := s.stringProperty("STATUS")
			if err != nil {
				return nil, err
			}
			e.Status = val
		case itemTransp:
			val, err := s.stringProperty("TRANSP")
			if err != nil {
				return nil, err
			}
			e.Transp = val
		case itemLocation:
			val, err := s.stringProperty("LOCATION")
			if err != nil {
				return nil, err
			}
			e.Location = val
		case itemCategories:
			val, err := s.stringProperty("CATEGORIES")
			if err != nil {
				return nil, err
			}
			e.Categories = strings.Split(val, ",")
		case itemDescription:
			val, err := s.stringProperty("DESCRIPTION")
			if err != nil {
				return nil, err
			}
			e.Description = val
		case itemURL:
			val, err := s.stringProperty("URL")
			if err != nil {
				return nil, err
			}
			eventURL, err := url.Parse(val)
			if err != nil {
				return nil, err
			}
			e.URL = eventURL
		case itemSequence:
			if i := s.next(); i.typ != itemColon {
				return nil, fmt.Errorf("unexpected (%s) after SEQUENCE, expected colon (:)", i.val)
			}
			i := s.next()
			if i.typ != itemInteger {
				return nil, fmt.Errorf("unexpected (%s) after SEQUENCE:, expected integer value", i.val)
			}
			val, err := strconv.Atoi(i.val)
			if err != nil {
				return nil, fmt.Errorf("error parsing SEQUENCE value: %s", err)
			}
			e.Sequence = val
		case itemRrule:
			if i := s.next(); i.typ != itemColon {
				return nil, fmt.Errorf("unexpected (%s) after RRULE, expected colon (:)", i.val)
			}
			i := s.next()
			if i.typ != itemRecur {
				return nil, fmt.Errorf("unexpected (%s) after RRULE:, expected Recur value", i.val)
			}
			val, err := parseRecur(i.val)
			if err != nil {
				return nil, fmt.Errorf("error parsing Recur: %s", err)
			}
			e.Rrule = val
		case itemDTStart:
			t, err := parseDateTimeProperty(s, "DTSTART")
			if err != nil {
				return nil, err
			}
			e.DTStart = t
		case itemDTEnd:
			t, err := parseDateTimeProperty(s, "DTEND")
			if err != nil {
				return nil, err
			}
			e.DTEnd = t
		case itemDTStamp:
			t, err := parseDateTimeProperty(s, "DTSTAMP")
			if err != nil {
				return nil, err
			}
			e.DTStamp = t
		case itemGeo:
			if i := s.next(); i.typ != itemColon {
				return nil, fmt.Errorf("unexpected (%s) after GEO, expected colon (:)", i.val)
			}
			i := s.next()
			if i.typ != itemLatLong {
				return nil, fmt.Errorf("unexpected (%s) after GEO geo-point value", i.val)
			}
			point := strings.Split(i.val, ";")
			if len(point) != 2 {
				return nil, fmt.Errorf("not valid geo-point value (%s)", i.val)
			}
		case itemBegin:
			if i := s.next(); i.typ != itemColon {
				return nil, fmt.Errorf("unexpected (%s) after BEGIN, expected colon (:)", i.val)
			}
			i := s.next()
			if i.typ != itemVAlarm {
				return nil, fmt.Errorf("unexpected (%s) after BEGIN, expected VALARM", i.val)
			}
			alarm, err := parseAlarm(s)
			if err != nil {
				return nil, err
			}
			e.Alarm = alarm
		default:
			return nil, fmt.Errorf("unexpected item (%s) in VEVENT", i)
		}
	}
}

func parseDateTimeProperty(s *scanner, propertyName string) (time.Time, error) {
	i := s.next()
	switch i.typ {
	case itemColon:
		i := s.next()
		if i.typ != itemTime && i.typ != itemDate {
			return time.Time{}, fmt.Errorf("unexpected (%s) after %s date-time value", i.val, propertyName)
		}
		t, err := parseDateTime(i.val)
		if err != nil {
			return time.Time{}, err
		}
		return t, nil
	case itemSemiColon:
		i := s.next()
		if i.typ != itemTimeZone {
			return time.Time{}, fmt.Errorf("unexpected (%s) after %s; expected TZID", i.val, propertyName)
		}
		loc, err := time.LoadLocation(i.val)
		if err != nil {
			return time.Time{}, fmt.Errorf("error loading time-zone: %s", err)
		}
		i = s.next()
		if i.typ != itemColon {
			return time.Time{}, fmt.Errorf("unexpected (%s) after time-zone, expected colon (:)", i.val)
		}
		i = s.next()
		if i.typ != itemTime && i.typ != itemDate {
			return time.Time{}, fmt.Errorf("unexpected (%s) after %s date-time value", i.val, propertyName)
		}
		t, err := parseDateTime(i.val)
		if err != nil {
			return time.Time{}, err
		}
		return t.In(loc), nil
	default:
		return time.Time{}, fmt.Errorf("unexpected (%s) after %s, expected colon (:)", i.val, propertyName)
	}
}

func parseRecur(val string) (*Rrule, error) {
	r := &Rrule{}
	for _, pair := range strings.Split(val, ";") {
		keyValue := strings.Split(pair, "=")
		if len(keyValue) != 2 {
			return nil, fmt.Errorf("not valid recur-rule-part: %s", pair)
		}
		val := keyValue[1]
		switch keyValue[0] {
		case "FREQ":
			freq, err := parseFrequency(val)
			if err != nil {
				return nil, err
			}
			r.Freq = freq
		case "UNTIL":
			until, err := parseDateTime(val)
			if err != nil {
				return nil, fmt.Errorf("not valid UNTIL value: %s", err)
			}
			r.Until = &until
		case "COUNT":
			count, err := strconv.Atoi(val)
			if err != nil {
				return nil, fmt.Errorf("not valid COUNT value: %s", err)
			}
			r.Count = &count
		case "INTERVAL":
			interval, err := strconv.Atoi(val)
			if err != nil {
				return nil, fmt.Errorf("not valid INTERVAL value: %s", err)
			}
			r.Interval = &interval
		case "BYSECOND":
			list, err := parseIntList(val, 0, 60)
			if err != nil {
				return nil, fmt.Errorf("not valid BYSECOND value: %s", err)
			}
			r.BySecond = list
		case "BYMINUTE":
			list, err := parseIntList(val, 0, 59)
			if err != nil {
				return nil, fmt.Errorf("not valid BYMINUTE value: %s", err)
			}
			r.ByMinute = list
		case "BYHOUR":
			list, err := parseIntList(val, 0, 24)
			if err != nil {
				return nil, fmt.Errorf("not valid BYHOUR value: %s", err)
			}
			r.ByHour = list
		case "BYDAY":
			days := strings.Split(val, ",")
			r.ByDay = make([]time.Weekday, len(days))
			for i, day := range days {
				d, err := parseIcsDay(day)
				if err != nil {
					return nil, err
				}
				r.ByDay[i] = d
			}
		case "BYMONTHDAY":
			list, err := parseIntList(val, 1, 31)
			if err != nil {
				return nil, fmt.Errorf("not valid BYMONTHDAY value: %s", err)
			}
			r.ByMonthday = list
		case "BYYEARDAY":
			list, err := parseIntList(val, 1, 366)
			if err != nil {
				return nil, fmt.Errorf("not valid BYYEARDAY value: %s", err)
			}
			r.ByYearday = list
		case "BYWEEKNO":
			list, err := parseIntList(val, 1, 53)
			if err != nil {
				return nil, fmt.Errorf("not valid BYWEEKNO value: %s", err)
			}
			r.ByWeekNo = list
		case "BYMONTH":
			list, err := parseIntList(val, 1, 12)
			if err != nil {
				return nil, fmt.Errorf("not valid BYMONTH value: %s", err)
			}
			r.ByMonth = make([]time.Month, len(list))
			for i, m := range list {
				r.ByMonth[i] = time.Month(m)
			}
		case "BYSETPOS":
			list, err := parseIntList(val, 1, 366)
			if err != nil {
				return nil, fmt.Errorf("not valid BYSETPOS value: %s", err)
			}
			r.BySetPos = list
		case "WKST":
			wd, err := parseIcsDay(val)
			if err != nil {
				return nil, err
			}
			r.Wkst = &wd
		}
	}
	return r, nil
}

func parseIntList(val string, from, to int) ([]int, error) {
	list := strings.Split(val, ",")
	if len(list) == 0 {
		return nil, nil
	}
	resultList := make([]int, len(list))
	for i, sec := range list {
		s, err := strconv.Atoi(sec)
		if err != nil {
			return nil, err
		}
		if s < from || s > to {
			return nil, fmt.Errorf("must be in range %d-%d", from, to)
		}
		resultList[i] = s
	}
	return resultList, nil
}

const (
	//IcsFormat ics date time format
	IcsFormat = "20060102T150405"
	//IcsFormatUTC ics UTC date time format
	IcsFormatUTC = "20060102T150405Z"
	//IcsFormatDate ics date format
	IcsFormatDate = "20060102"
)

func parseDateTime(val string) (time.Time, error) {
	t, err := time.Parse(IcsFormatUTC, val)
	if err == nil {
		return t, nil
	}
	t, err = time.Parse(IcsFormat, val)
	if err == nil {
		return t, nil
	}
	return time.Parse(IcsFormatDate, val)
}

func parseAlarm(s *scanner) (*Alarm, error) {
	a := &Alarm{}
	for i := s.next(); ; i = s.next() {
		switch i.typ {
		case itemEnd:
			if i := s.next(); i.typ != itemColon {
				return nil, fmt.Errorf("unexpected (%s) after END, expected colon (:)", i.val)
			}
			if i := s.next(); i.typ != itemVAlarm {
				return nil, fmt.Errorf("unexpected (%s) after END:, expected VALARM", i.val)
			}
			return a, nil
		default:
			return nil, fmt.Errorf("unexpected item in VALARM: (%s)", i.val)
		}
	}
}

func parseTimeZone(s *scanner) (*TimeZone, error) {
	tz := &TimeZone{}
	for i := s.next(); ; i = s.next() {
		switch i.typ {
		case itemEnd:
			if i := s.next(); i.typ != itemColon {
				return nil, fmt.Errorf("unexpected (%s) after END, expected colon (:)", i.val)
			}
			if i := s.next(); i.typ != itemVTimeZone {
				return nil, fmt.Errorf("unexpected (%s) after END:, expected VTIMEZONE", i.val)
			}
			return tz, nil
		case itemX:
			s.backup()
			propName, val, err := s.xProperty()
			if err != nil {
				return nil, err
			}
			if tz.Properties == nil {
				tz.Properties = make(map[string]string)
			}
			tz.Properties[propName] = val
		case itemTZID:
			val, err := s.stringProperty("TZID")
			if err != nil {
				return nil, err
			}
			loc, err := time.LoadLocation(val)
			if err != nil {
				return nil, fmt.Errorf("cannot load TZID: %s", err)
			}
			tz.TZID = loc
		case itemURL:
			val, err := s.stringProperty("TZURL")
			if err != nil {
				return nil, err
			}
			tzURL, err := url.Parse(val)
			if err != nil {
				return nil, err
			}
			tz.TZURL = tzURL
		case itemTZOffsetFrom:
			val, err := s.stringProperty("TZOFFSETFROM")
			if err != nil {
				return nil, err
			}
			var h, m int
			fmt.Sscanf(val[1:], "%02d%02d", &h, &m)
		default:
			return nil, fmt.Errorf("unexpected item in VTIMEZONE: (%s)", i.val)
		}
	}
}
