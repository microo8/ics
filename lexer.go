package ics

import (
	"bufio"
	"fmt"
	"io"
	"strings"
	"unicode"
)

//itemType identifies the type of lex items.
type itemType int

const (
	itemError itemType = iota // error occurred; value is text of error
	itemEOF

	//basic items
	itemColon
	itemBegin
	itemEnd

	//VCALENDAR items
	itemVCalendar
	itemVersion
	itemProdID
	itemCalscale
	itemMethod
	itemVEvent

	//VEVENT items
	itemSummary
	itemUID
	itemSequence
	itemStatus
	itemTransp
	itemRrule
	itemDTStart
	itemDTEnd
	itemDTStamp
	itemCategories
	itemLocation
	itemGeo
	itemDescription
	itemURL

	//property value items
	itemRecur
	itemDate
	itemTime
	itemLatLong
	itemString
	itemInteger
)

type item struct {
	typ itemType
	val string
}

func (i item) String() string {
	switch i.typ {
	case itemEOF:
		return "EOF"
	case itemError:
		return i.val
	}
	return fmt.Sprintf("%q", i.val)
}

var (
	eof = rune(0)
)

func isWhitespace(ch rune) bool {
	return ch == ' ' || ch == '\t' || ch == '\n'
}

//stateFn represents the state of the scanner as a function that returns the next state.
type stateFn func(*lexer) stateFn

// lexer holds the state of the scanner.
type lexer struct {
	input                     *bufio.Reader   // the data being scanned.
	buf                       strings.Builder //the data already scanned
	line, start, pos, prevpos int
	items                     chan item // channel of scanned items.
}

//run lexes the input by executing state functions until the state is nil.
func (l *lexer) run() {
	for state := lexStart; state != nil; {
		state = state(l)
	}
	close(l.items) // No more tokens will be delivered.
}

// emit passes an item back to the client.
func (l *lexer) emit(t itemType) {
	l.items <- item{t, l.buf.String()}
	l.buf.Reset()
}

// read reads the next rune from the bufferred reader.
// Returns the rune(0) if an error occurs (or io.EOF is returned).
func (l *lexer) read() rune {
	ch, _, err := l.input.ReadRune()
	if ch == '\n' {
		l.line++
		l.prevpos = l.pos
		l.pos = 0
	} else {
		l.pos++
		if err != nil {
			return eof
		}
	}
	l.buf.WriteRune(ch)
	return ch
}

// unread places the previously read rune back on the reader.
func (l *lexer) unread() {
	l.input.UnreadRune()
	if l.pos == 0 {
		l.pos = l.prevpos
		if l.line == 0 {
			panic("Cannot unread! No runes readed")
		}
		l.line--
	} else {
		l.pos--
	}
	buf := l.buf.String()
	l.buf.Reset()
	if len(buf) > 0 {
		l.buf.WriteString(buf[:len(buf)-1])
	}
}

//peek returns but does not consume the next rune in the input.
func (l *lexer) peek() rune {
	r := l.read()
	l.unread()
	return r
}

//readLetters reads all runes that are letters
func (l *lexer) readLetters() string {
	for {
		if ch := l.read(); ch == eof {
			break
		} else if !unicode.IsLetter(ch) {
			l.unread()
			break
		}
	}
	ret := l.buf.String()
	return ret
}

//readDigits reads all runes that are letters
func (l *lexer) readDigits() string {
	for {
		if ch := l.read(); ch == eof {
			break
		} else if !unicode.IsDigit(ch) {
			l.unread()
			break
		}
	}
	ret := l.buf.String()
	return ret
}

//readStringProperty reads property name, following colon and the string value
func (l *lexer) readStringProperty(name string, item itemType) {
	l.emit(item)
	if r := l.read(); r != ':' {
		l.errorf("unexpected character after %s (%c) expected colon (:)", name, r)
		return
	}
	l.emit(itemColon)
	l.acceptToLineBreak()
	l.emit(itemString)
	l.accept("\n\r")
	l.buf.Reset()
}

//acceptToLineBreak reads entire string to line break
func (l *lexer) acceptToLineBreak() {
	for {
		if ch := l.read(); ch == eof {
			break
		} else if ch == '\r' || ch == '\n' {
			if ch = l.read(); ch != '\n' && ch != '\r' {
				l.unread()
			}
			l.unread()
			break
		}
	}
}

//accept consumes the next rune if it's from the valid set.
func (l *lexer) accept(valid string) bool {
	if strings.IndexRune(valid, l.read()) >= 0 {
		return true
	}
	l.unread()
	return false
}

//acceptRun consumes a run of runes from the valid set.
func (l *lexer) acceptRun(valid string) {
	for strings.IndexRune(valid, l.read()) >= 0 {
	}
	l.unread()
}

func (l *lexer) acceptWhitespace() {
	l.acceptRun(" \t\n\r")
	l.buf.Reset()
}

//errorf returns an error token and terminates the scan
//by passing back a nil pointer that will be the next
//state, terminating l.run.
func (l *lexer) errorf(format string, args ...interface{}) stateFn {
	l.items <- item{
		itemError,
		fmt.Sprintf("%d:%d"+format, append([]interface{}{l.line, l.pos}, args...)...),
	}
	return nil
}

func lex(input io.Reader) *lexer {
	l := &lexer{
		input: bufio.NewReader(input),
		items: make(chan item, 5),
	}
	go l.run() // Concurrently run state machine.
	return l
}

func lexStart(l *lexer) stateFn {
	l.acceptWhitespace()
	if word := l.readLetters(); word != "BEGIN" {
		return l.errorf("unexpected word at start (%s) expected BEGIN", word)
	}
	l.emit(itemBegin)
	if r := l.read(); r != ':' {
		return l.errorf("unexpected character after BEGIN (%c) expected colon (:)", r)
	}
	l.emit(itemColon)
	if word := l.readLetters(); word != "VCALENDAR" {
		return l.errorf("unexpected word after BEGIN: (%s) expected VCALENDAR", word)
	}
	l.emit(itemVCalendar)
	if r := l.read(); r != '\r' && r != '\n' {
		return l.errorf("unexpected character after BEGIN:VCALENDAR (%c) expected CR or LF (\\r or \\n)", r)
	}
	l.buf.Reset()
	return lexVCalendar
}

func lexVCalendar(l *lexer) stateFn {
	l.acceptWhitespace()
	word := l.readLetters()
	switch word {
	case "VERSION":
		l.readStringProperty("VERSION", itemVersion)
		return lexVCalendar
	case "PRODID":
		l.readStringProperty("PRODID", itemProdID)
		return lexVCalendar
	case "CALSCALE":
		l.readStringProperty("CALSCALE", itemCalscale)
		return lexVCalendar
	case "METHOD":
		l.readStringProperty("METHOD", itemMethod)
		return lexVCalendar
	case "BEGIN":
		l.emit(itemBegin)
		if r := l.read(); r != ':' {
			return l.errorf("unexpected character after METHOD (%c) expected colon (:)", r)
		}
		l.emit(itemColon)
		if word := l.readLetters(); word != "VEVENT" {
			return l.errorf("unexpected word after BEGIN: (%s) in VCALENDAR, expected VEVENT", word)
		}
		l.emit(itemVEvent)
		if r := l.read(); r != '\r' && r != '\n' {
			return l.errorf("unexpected character after BEGIN:VEVENT (%c) expected CR or LF (\\r or \\n)", r)
		}
		l.buf.Reset()
		return lexVEvent
	case "END":
		l.emit(itemEnd)
		if r := l.read(); r != ':' {
			return l.errorf("unexpected character after END (%c), expected colon (:)", r)
		}
		l.emit(itemColon)
		if word := l.readLetters(); word != "VCALENDAR" {
			return l.errorf("unexpected word after END: (%s), expected VCALENDAR", word)
		}
		l.emit(itemVCalendar)
		return nil
	}
	return l.errorf("unexpected word in VCALENDAR (%s)", word)
}

func lexVEvent(l *lexer) stateFn {
	l.acceptWhitespace()
	word := l.readLetters()
	switch word {
	case "SUMMARY":
		l.readStringProperty("SUMMARY", itemSummary)
		return lexVEvent
	case "UID":
		l.readStringProperty("UID", itemUID)
		return lexVEvent
	case "STATUS":
		l.readStringProperty("STATUS", itemStatus)
		return lexVEvent
	case "TRANSP":
		l.readStringProperty("TRANSP", itemTransp)
		return lexVEvent
	case "CATEGORIES":
		l.readStringProperty("CATEGORIES", itemCategories)
		return lexVEvent
	case "LOCATION":
		l.readStringProperty("LOCATION", itemLocation)
		return lexVEvent
	case "DESCRIPTION":
		l.readStringProperty("DESCRIPTION", itemDescription)
		return lexVEvent
	case "URL":
		l.readStringProperty("URL", itemURL)
		return lexVEvent
	case "SEQUENCE":
		l.emit(itemSequence)
		if r := l.read(); r != ':' {
			return l.errorf("unexpected character after SEQUENCE (%c) expected colon (:)", r)
		}
		l.emit(itemColon)
		if digits := l.readDigits(); digits == "" {
			return l.errorf("unexpected error after SEQUENCE, expected an interger")
		}
		l.emit(itemInteger)
		return lexVEvent
	case "RRULE":
		l.emit(itemRrule)
		if r := l.read(); r != ':' {
			return l.errorf("unexpected character after RRULE (%c) expected colon (:)", r)
		}
		l.emit(itemColon)
		l.acceptToLineBreak()
		l.emit(itemRecur)
		return lexVEvent
	case "GEO":
		l.emit(itemGeo)
		if r := l.read(); r != ':' {
			return l.errorf("unexpected character after GEO (%c) expected colon (:)", r)
		}
		l.emit(itemColon)
		l.acceptToLineBreak()
		l.emit(itemLatLong)
		return lexVEvent
	case "DTSTART":
		l.emit(itemDTStart)
		if r := l.read(); r != ':' {
			return l.errorf("unexpected character after DTSTART (%c) expected colon (:)", r)
		}
		l.emit(itemColon)
		return lexDateTime
	case "DTEND":
		l.emit(itemDTEnd)
		if r := l.read(); r != ':' {
			return l.errorf("unexpected character after DTEND (%c) expected colon (:)", r)
		}
		l.emit(itemColon)
		return lexDateTime
	case "DTSTAMP":
		l.emit(itemDTStamp)
		if r := l.read(); r != ':' {
			return l.errorf("unexpected character after DTSTAMP (%c) expected colon (:)", r)
		}
		l.emit(itemColon)
		return lexDateTime
	case "END":
		l.emit(itemEnd)
		if r := l.read(); r != ':' {
			return l.errorf("unexpected character after END (%c), expected colon (:)", r)
		}
		l.emit(itemColon)
		if word := l.readLetters(); word != "VEVENT" {
			return l.errorf("unexpected word after END: (%s), expected VEVENT", word)
		}
		l.emit(itemVEvent)
		return lexVCalendar
	}
	return l.errorf("unexpected word in VEVENT (%s)", word)
}

func lexDateTime(l *lexer) stateFn {
	l.readDigits()
	r := l.read()
	switch r {
	case 'T':
		l.read()
		l.readDigits()
		ch := l.read()
		if ch != '\r' && ch != '\n' && ch != 'Z' {
			return l.errorf("unsupported DATE-TIME value (%s)", l.buf.String())
		}
		if ch == '\r' || ch == '\n' {
			l.unread()
		}
		l.emit(itemTime)
	case '\r', '\n':
		l.unread()
		l.emit(itemDate)
	default:
		return l.errorf("unsupported DATE-TIME value (%s)", l.buf.String())
	}
	l.acceptWhitespace()
	return lexVEvent
}
