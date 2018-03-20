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

func parseCalendar(s *scanner) (*Calendar, error) {
	if i := s.next(); i.typ != itemBegin || i.val != "BEGIN:VCALENDAR" {
		return nil, fmt.Errorf("iCalendar must start with BEGIN:VCALENDAR, not (%s)", i.val)
	}
	cal := &Calendar{}
	for i := s.next(); i.typ != itemEOF; i = s.next() {
		switch i.typ {
		case itemError:
			return nil, fmt.Errorf(i.val)
		case itemEnd:
			if i.val != "END:VCALENDAR" {
				return nil, fmt.Errorf("unexpected (%s) expected END:VCALENDAR", i.val)
			}
			return cal, nil
		case itemBegin:
			switch i.val {
			case "BEGIN:VEVENT":
				e, err := parseEvent(s)
				if err != nil {
					return nil, fmt.Errorf("error parsing VEVENT: %s", err)
				}
				cal.Events = append(cal.Events, e)
			case "BEGIN:VTIMEZONE":
				tz, err := parseTimeZone(s)
				if err != nil {
					return nil, fmt.Errorf("error parsing VTIMEZONE: %s", err)
				}
				cal.TimeZone = tz
			default:
				return nil, fmt.Errorf("unexpected (%s) after BEGIN:, expected VEVENT or VTIMEZONE", i.val)
			}
		case itemProperty:
			iVal := s.next()
			if iVal.typ != itemValue {
				return nil, fmt.Errorf("unexpected item (%s), expected value", i)
			}
			if len(i.val) > 2 && i.val[:2] == "X-" {
				if cal.Properties == nil {
					cal.Properties = make(map[string]string)
				}
				cal.Properties[i.val[2:]] = iVal.val
				continue
			}
			switch i.val {
			case "VERSION":
				cal.Version = iVal.val
			case "PRODID":
				cal.ProdID = iVal.val
			case "CALSCALE":
				cal.Calscale = iVal.val
			case "METHOD":
				cal.Method = iVal.val
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
		case itemError:
			return nil, fmt.Errorf(i.val)
		case itemEnd:
			if i.val != "END:VEVENT" {
				return nil, fmt.Errorf("unexpected (%s) expected END:VEVENT", i.val)
			}
			fmt.Println("Event", e)
			return e, nil
		case itemBegin:
			if i.val != "BEGIN:VALARM" {
				return nil, fmt.Errorf("unexpected (%s) expected BEGIN:VALARM", i.val)
			}
			alarm, err := parseAlarm(s)
			if err != nil {
				return nil, err
			}
			e.Alarm = alarm
		case itemProperty:
			iVal := s.next()
			var par map[string]string
			if iVal.typ == itemParam {
				var err error
				par, err = parseParam(iVal.val)
				if err != nil {
					return nil, err
				}
				iVal = s.next()
			}
			if iVal.typ != itemValue {
				return nil, fmt.Errorf("unexpected item (%s), expected value", i)
			}
			val := iVal.val
			if len(i.val) > 2 && i.val[:2] == "X-" {
				if e.Properties == nil {
					e.Properties = make(map[string]string)
				}
				e.Properties[i.val[2:]] = val
				continue
			}
			switch i.val {
			case "CLASS":
				switch val {
				case "PUBLIC":
					e.Class = ClassPublic
				case "PRIVATE":
					e.Class = ClassPrivate
				case "CONFIDENTIAL":
					e.Class = ClassConfidential
				default:
					return nil, fmt.Errorf("not valid value for classification (%s)", val)
				}
			case "SUMMARY":
				e.Summary = val
			case "UID":
				e.UID = val
			case "STATUS":
				e.Status = val
			case "TRANSP":
				e.Transp = val
			case "LOCATION":
				e.Location = val
			case "CATEGORIES":
				e.Categories = strings.Split(val, ",")
			case "DESCRIPTION":
				e.Description = val
			case "URL":
				val, err := url.Parse(val)
				if err != nil {
					return nil, fmt.Errorf("cannot parse url: %s", err)
				}
				e.URL = val
			case "SEQUENCE":
				val, err := strconv.Atoi(val)
				if err != nil {
					return nil, fmt.Errorf("error parsing SEQUENCE value: %s", err)
				}
				e.Sequence = val
			case "RRULE":
				val, err := parseRecur(val)
				if err != nil {
					return nil, fmt.Errorf("error parsing Recur: %s", err)
				}
				e.Rrule = val
			case "GEO":
				point := strings.Split(val, ";")
				if len(point) != 2 {
					return nil, fmt.Errorf("not valid geo-point value (%s)", val)
				}
				v, err := strconv.ParseFloat(point[0], 64)
				if err != nil {
					return nil, fmt.Errorf("not valid geo-point value (%s)", val)
				}
				e.Geo.Latitude = v
				v, err = strconv.ParseFloat(point[0], 64)
				if err != nil {
					return nil, fmt.Errorf("not valid geo-point value (%s)", val)
				}
				e.Geo.Longitude = v
			case "EXDATE":
				exDates := strings.Split(val, ",")
				e.ExDate = make([]time.Time, len(exDates))
				for i, val := range exDates {
					t, err := parseDateTime(nil, val)
					if err != nil {
						return nil, err
					}
					e.ExDate[i] = t
				}
			case "CREATED":
				t, err := parseDateTime(par, val)
				if err != nil {
					return nil, err
				}
				e.Created = t
			case "LAST-MODIFIED":
				t, err := parseDateTime(par, val)
				if err != nil {
					return nil, err
				}
				e.LastModified = t
			case "DTSTART":
				t, err := parseDateTime(par, val)
				if err != nil {
					return nil, err
				}
				e.DTStart = t
			case "DTEND":
				t, err := parseDateTime(par, val)
				if err != nil {
					return nil, err
				}
				e.DTEnd = t
			case "DTSTAMP":
				t, err := parseDateTime(par, val)
				if err != nil {
					return nil, err
				}
				e.DTStamp = t
			case "ORGANIZER":
				e.Organizer = &Attendee{Parameters: par, Value: val}
			case "ATTENDEE":
				e.Attendee = &Attendee{Parameters: par, Value: val}
			case "PARTICIPANT":
				e.Participant = &Attendee{Parameters: par, Value: val}
			default:
				return nil, fmt.Errorf("unexpected item (%s) in VEVENT", i)
			}
		default:
			return nil, fmt.Errorf("unexpected item (%s) in VEVENT", i)
		}
	}
}

func parseParam(val string) (map[string]string, error) {
	if val == "" {
		return nil, nil
	}
	ret := make(map[string]string)
	for _, pair := range strings.Split(val, ";") {
		keyValue := strings.Split(pair, "=")
		if len(keyValue) != 2 {
			return nil, fmt.Errorf("not valid params: %s", pair)
		}
		ret[keyValue[0]] = keyValue[1]
	}
	return ret, nil
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
			until, err := parseDateTime(nil, val)
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
			list := strings.Split(val, ",")
			r.ByDay = make([]WDay, len(list))
			for i, val := range list {
				sign := 1
				switch val[0] {
				case '+':
					val = val[1:]
				case '-':
					sign = -1
					val = val[1:]
				}
				var num int
				var weekday string
				fmt.Sscanf(val, "%d%s", &num, &weekday)
				wd, err := parseIcsDay(weekday)
				if err != nil {
					return nil, fmt.Errorf("not valid BYDAY value: %s", err)
				}
				r.ByDay[i] = WDay{Num: sign * num, Weekday: wd}
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

const (
	//IcsFormat ics date time format
	IcsFormat = "20060102T150405"
	//IcsFormatUTC ics UTC date time format
	IcsFormatUTC = "20060102T150405Z"
	//IcsFormatDate ics date format
	IcsFormatDate = "20060102"
)

func parseDateTime(par map[string]string, val string) (time.Time, error) {
	if timeZone, ok := par["TZID"]; ok {
		loc, err := time.LoadLocation(timeZone)
		if err != nil {
			return time.Time{}, fmt.Errorf("cannot load time-zone (%s)", timeZone)
		}
		t, err := parseDateTime(nil, val)
		if err != nil {
			return time.Time{}, err
		}
		return t.In(loc), nil
	}
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
			if i.val != "END:VALARM" {
				return nil, fmt.Errorf("unexpected (%s) expected END:VALARM", i.val)
			}
			return a, nil
		case itemProperty:
			iVal := s.next()
			if iVal.typ != itemValue {
				return nil, fmt.Errorf("unexpected item (%s), expected value", i)
			}
			val := iVal.val
			switch i.val {
			case "TRIGGER":
				a.Trigger = val
			}
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
			if i.val != "END:VTIMEZONE" {
				return nil, fmt.Errorf("unexpected (%s) END:VTIMEZONE", i.val)
			}
			return tz, nil
		case itemBegin:
			switch i.val {
			case "BEGIN:DAYLIGHT", "BEGIN:STANDARD":
				s, err := parseStandard(s)
				if err != nil {
					return nil, fmt.Errorf("error parsing STANDARD: %s", err)
				}
				tz.Standard = s
			default:
				return nil, fmt.Errorf("unexpected (%s) after BEGIN:, expected DAYLIGHT or STANDARD", i.val)
			}
		case itemProperty:
			iVal := s.next()
			if iVal.typ != itemValue {
				return nil, fmt.Errorf("unexpected item (%s), expected value", i)
			}
			val := iVal.val
			switch i.val {
			case "TZID":
				tz.TZID = val
			case "TZURL":
				val, err := url.Parse(val)
				if err != nil {
					return nil, err
				}
				tz.TZURL = val
			default:
				return nil, fmt.Errorf("unexpected property in VTIMEZONE: (%s)", i.val)
			}
		default:
			return nil, fmt.Errorf("unexpected item in VTIMEZONE: (%s)", i.val)
		}
	}
}

func parseStandard(s *scanner) (*Standard, error) {
	d := &Standard{}
	for i := s.next(); ; i = s.next() {
		switch i.typ {
		case itemEnd:
			if i.val != "END:DAYLIGHT" && i.val != "END:STANDARD" {
				return nil, fmt.Errorf("unexpected (%s) END:DAYLIGHT or END:STANDARD", i.val)
			}
			return d, nil
		case itemProperty:
			iVal := s.next()
			var par map[string]string
			if iVal.typ == itemParam {
				var err error
				par, err = parseParam(iVal.val)
				if err != nil {
					return nil, err
				}
				iVal = s.next()
			}
			if iVal.typ != itemValue {
				return nil, fmt.Errorf("unexpected item (%s), expected value", i)
			}
			val := iVal.val
			switch i.val {
			case "RDATE":
				d.Rdate = val
			case "TZNAME":
				d.TZName = val
			case "TZOFFSETFROM":
				val, err := parseDuration(val)
				if err != nil {
					return nil, err
				}
				d.TZOffsetFrom = val
			case "TZOFFSETTO":
				val, err := parseDuration(val)
				if err != nil {
					return nil, err
				}
				d.TZOffsetTo = val
			case "DTSTART":
				t, err := parseDateTime(par, val)
				if err != nil {
					return nil, err
				}
				d.DTStart = t
			case "RRULE":
				val, err := parseRecur(val)
				if err != nil {
					return nil, fmt.Errorf("error parsing Recur: %s", err)
				}
				d.Rrule = val
			default:
				return nil, fmt.Errorf("unexpected property in DAYLIGHT: (%s)", i.val)
			}
		default:
			return nil, fmt.Errorf("unexpected property in DAYLIGHT: (%s)", i.val)
		}
	}
}

func parseDuration(val string) (int, error) {
	var h, m int
	_, err := fmt.Sscanf(val[1:], "%02d%02d", &h, &m)
	if err != nil {
		return 0, fmt.Errorf("cannot parse DURATION value %s", err)
	}
	ret := h*3600 + m*60
	if val[0] == '-' {
		return ret * -1, nil
	}
	return ret, nil
}
