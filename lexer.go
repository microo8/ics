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
	itemBegin
	itemEnd
	itemProperty
	itemParam
	itemValue
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
	var buf strings.Builder
	for {
		ch := l.read()
		if ch == eof {
			break
		} else if !unicode.IsLetter(ch) {
			l.unread()
			break
		}
		buf.WriteRune(ch)
	}
	return buf.String()
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

func (l *lexer) readProperty() {
	l.accept("ABCDEFGHIJKLMNOPQRSTUVWXYZ-")
	l.emit(itemProperty)
	l.acceptWhitespace()
	r := l.read()
	switch r {
	case ';':
		l.buf.Reset()
		l.accept("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz=-; ")
		l.emit(itemParam)
		if r := l.read(); r != ':' {
			l.errorf("unexpected character after params (%c) expected colon (:)", r)
		}
		l.readValue()
	case ':':
		l.readValue()
	default:
		l.errorf("unexpected character after property (%c) expected colon (:)", r)
		return
	}

}
func (l *lexer) readValue() {
	l.buf.Reset()
	l.acceptToLineBreak()
	l.emit(itemValue)
	l.acceptWhitespace()
}

//acceptToLineBreak reads entire string to line break
func (l *lexer) acceptToLineBreak() {
	for {
		if ch := l.read(); ch == eof {
			break
		} else if ch == '\r' || ch == '\n' {
			if ch = l.read(); ch == ' ' {
				continue
			}
			l.unread()
			l.unread()
			break
		}
	}
}

//acceptRun consumes a run of runes from the valid set.
func (l *lexer) accept(valid string) {
	for strings.ContainsRune(valid, l.read()) {
	}
	l.unread()
}

func (l *lexer) acceptWhitespace() {
	l.accept(" \t\n\r")
	l.buf.Reset()
}

func (l *lexer) ignoreWhitespace() {
	for {
		ch, _, err := l.input.ReadRune()
		if ch == '\n' {
			l.line++
			l.prevpos = l.pos
			l.pos = 0
		} else {
			if !strings.ContainsRune(" \t\n\r", ch) {
				l.input.UnreadRune()
				return
			}
			l.pos++
			if err != nil {
				return
			}
		}
	}
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
	if r := l.read(); r != ':' {
		return l.errorf("unexpected character after BEGIN (%c) expected colon (:)", r)
	}
	if word := l.readLetters(); word != "VCALENDAR" {
		return l.errorf("unexpected word after BEGIN: (%s) expected VCALENDAR", word)
	}
	l.emit(itemBegin)
	if r := l.read(); r != '\r' && r != '\n' {
		return l.errorf("unexpected character after BEGIN:VCALENDAR (%c) expected CR or LF (\\r or \\n)", r)
	}
	l.buf.Reset()
	return lexVCalendar
}

func lexVCalendar(l *lexer) stateFn {
	l.acceptWhitespace()
	word := l.readLetters()
	l.ignoreWhitespace()
	if r := l.read(); r != ':' && r != '-' {
		return l.errorf("unexpected character (%c) after property name (%s) expected (:) or (-)", r, word)
	}
	l.unread()
	switch word {
	case "X":
		l.readProperty()
		return lexVCalendar
	case "BEGIN":
		if r := l.read(); r != ':' {
			return l.errorf("unexpected character after METHOD (%c) expected colon (:)", r)
		}
		word := l.readLetters()
		l.emit(itemBegin)
		switch word {
		case "VEVENT":
			l.acceptWhitespace()
			return lexVEvent
		case "VTIMEZONE":
			l.acceptWhitespace()
			return lexVTimeZone
		default:
			return l.errorf("unexpected word after BEGIN: (%s) in VCALENDAR, expected VEVENT or VTIMEZONE", word)
		}
	case "END":
		if r := l.read(); r != ':' {
			return l.errorf("unexpected character after END (%c), expected colon (:)", r)
		}
		if word := l.readLetters(); word != "VCALENDAR" {
			return l.errorf("unexpected word after END: (%s), expected VCALENDAR", word)
		}
		l.emit(itemEnd)
		return nil
	default:
		l.readProperty()
		return lexVCalendar
	}
}

func lexVEvent(l *lexer) stateFn {
	l.acceptWhitespace()
	word := l.readLetters()
	l.ignoreWhitespace()
	r := l.read()
	if !strings.ContainsRune(":;-", r) {
		return l.errorf("unexpected character (%c) after property name (%s) expected (:;-)", r, word)
	}
	l.unread()
	switch word {
	case "BEGIN":
		l.read()
		word := l.readLetters()
		l.emit(itemBegin)
		switch word {
		case "VALARM":
			l.acceptWhitespace()
			return lexVAlarm
		default:
			return l.errorf("unexpected word after BEGIN: (%s) expected TODO", word)
		}
	case "END":
		l.read()
		if word := l.readLetters(); word != "VEVENT" {
			return l.errorf("unexpected word after END: (%s), expected VEVENT", word)
		}
		l.emit(itemEnd)
		return lexVCalendar
	case "DTSTART", "DTEND", "DTSTAMP":
		l.emit(itemProperty)
		return lexDateTime
	default:
		l.readProperty()
		return lexVEvent
	}
}

func lexDateTime(l *lexer) stateFn {
	if l.read() == ';' {
		l.buf.Reset()
		for l.read() != ':' {
		}
		l.unread()
		l.emit(itemParam)
		l.read()
	}
	l.buf.Reset()
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
		l.emit(itemValue)
	case '\r', '\n':
		l.unread()
		l.emit(itemValue)
	default:
		return l.errorf("unsupported DATE-TIME value (%s)", l.buf.String())
	}
	l.acceptWhitespace()
	return lexVEvent
}

func lexVAlarm(l *lexer) stateFn {
	l.acceptWhitespace()
	word := l.readLetters()
	l.ignoreWhitespace()
	switch word {
	case "END":
		if r := l.read(); r != ':' {
			return l.errorf("unexpected character after END (%c), expected colon (:)", r)
		}
		if word := l.readLetters(); word != "VALARM" {
			return l.errorf("unexpected word after END: (%s), expected VALARM", word)
		}
		l.emit(itemEnd)
		return lexVEvent
	default:
		l.readProperty()
		return lexVAlarm
	}
}

func lexVTimeZone(l *lexer) stateFn {
	l.acceptWhitespace()
	word := l.readLetters()
	l.ignoreWhitespace()
	if r := l.read(); r != ':' && r != '-' {
		return l.errorf("unexpected character (%c) after property name (%s) expected (:) or (-)", r, word)
	}
	l.unread()
	switch word {
	case "END":
		l.read()
		if word := l.readLetters(); word != "VTIMEZONE" {
			return l.errorf("unexpected word after END: (%s), expected VTIMEZONE", word)
		}
		l.emit(itemEnd)
		return lexVCalendar
	case "X":
		l.readProperty()
		return lexVTimeZone
	case "BEGIN":
		l.read()
		switch l.readLetters() {
		case "DAYLIGHT", "STANDARD":
			l.emit(itemBegin)
			return lexStandard
		default:
			return l.errorf("unexpected %s in VTIMEZONE, expected BEGIN:DAYLIGHT or BEGIN:STANDARD", word)
		}
	default:
		l.readProperty()
		return lexVTimeZone
	}
}

func lexStandard(l *lexer) stateFn {
	l.acceptWhitespace()
	word := l.readLetters()
	l.ignoreWhitespace()
	if r := l.read(); r != ':' && r != '-' {
		return l.errorf("unexpected character (%c) after property name (%s) expected (:) or (-)", r, word)
	}
	l.unread()
	switch word {
	case "END":
		l.read()
		switch l.readLetters() {
		case "DAYLIGHT", "STANDARD":
			l.emit(itemEnd)
			return lexVTimeZone
		default:
			return l.errorf("unexpected word after END: (%s), expected DAYLIGHT", word)
		}
	default:
		l.readProperty()
		return lexStandard
	}
}
