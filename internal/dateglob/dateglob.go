package dateglob

import (
	"fmt"
	"github.com/SimonSchneider/goslu/date"
	"strconv"
	"unicode"
)

// DateGlob represents a glob for dates.
// The glob is in the format of day-month-year where each part can be on the format
// of a number, a range of numbers, a list of numbers or a wildcard.
// The wildcard is represented by a star.
// Examples:
//   - 1-1-*
//   - 1,2,3-1,2,3-*
//   - 1-*-*
type DateGlob struct {
	Day   string
	Month string
	Year  string
}

func New(day, month, year string) DateGlob {
	return DateGlob{
		Day:   day,
		Month: month,
		Year:  year,
	}
}

func (dg DateGlob) String() string {
	return fmt.Sprintf("%s-%s-%s", dg.Day, dg.Month, dg.Year)
}

func (dg DateGlob) NextMatch(from date.Date) (date.Date, error) {
	for i := 0; i < 1000; i++ {
		next := from.Add(date.Duration(i) * date.Day)
		if dg.Match(next) {
			return next, nil
		}
	}
	return from, fmt.Errorf("no match found")
}

func (dg DateGlob) Match(d date.Date) bool {
	if dg.Year != "*" && strconv.Itoa(d.ToStdTime().Year()) != dg.Year {
		return false
	}
	if dg.Month != "*" && strconv.Itoa(int(d.ToStdTime().Month())) != dg.Month {
		return false
	}
	if dg.Day != "*" && strconv.Itoa(d.ToStdTime().Day()) != dg.Day {
		return false
	}
	return true
}

func Parse(glob string) (DateGlob, error) {
	dg := DateGlob{}
	tokens := lex(glob)
	elems, err := parse(tokens)
	if err != nil {
		return dg, err
	}
	if len(elems.elements) != 3 {
		return dg, fmt.Errorf("unexpected number of elements in glob: %d", len(elems.elements))
	}
	dg.Day = elems.elements[2]
	dg.Month = elems.elements[1]
	dg.Year = elems.elements[0]
	return dg, nil
}

func lex(glob string) []string {
	tokens := make([]string, 0)
	currToken := ""
	for _, c := range glob {
		if unicode.IsDigit(c) {
			if currToken == "" || unicode.IsDigit(rune(currToken[0])) {
				currToken += string(c)
			} else {
				tokens = append(tokens, currToken)
				currToken = string(c)
			}
		} else {
			if currToken != "" {
				tokens = append(tokens, currToken)
				currToken = string(c)
			} else {
				currToken += string(c)
			}
		}
	}
	if currToken != "" {
		tokens = append(tokens, currToken)
	}
	return tokens
}

type parser struct {
	elements []string
	scope    []string
}

func parse(tokens []string) (parser, error) {
	p := parser{elements: []string{""}}
	for i, t := range tokens {
		if err := p.push(t); err != nil {
			return parser{}, fmt.Errorf("parsing token: '%s' at %d: %w", t, i, err)
		}
	}
	return p, nil
}

func (p *parser) push(token string) error {
	if token == "{" || token == "(" || token == "[" {
		p.scope = append(p.scope, token)
		p.addToCurrent(token)
	} else if token == "}" || token == ")" || token == "]" {
		if len(p.scope) == 0 {
			return fmt.Errorf("unexpected closing scope token: %s", token)
		}
		lastScope := p.scope[len(p.scope)-1]
		if (lastScope == "{" && token != "}") || (lastScope == "(" && token != ")") || (lastScope == "[" && token != "]") {
			return fmt.Errorf("unexpected closing scope token: %s", token)
		}
		p.scope = p.scope[:len(p.scope)-1]
		p.addToCurrent(token)
	} else {
		if token == "-" {
			if len(p.scope) == 0 {
				if p.current() == "" {
					return fmt.Errorf("unexpected token, no elements for current")
				}
				p.elements = append(p.elements, "")
			} else {
				p.addToCurrent(token)
			}
		} else {
			p.addToCurrent(token)
		}
	}
	return nil
}

func (p *parser) current() string {
	return p.elements[len(p.elements)-1]
}

func (p *parser) addToCurrent(token string) {
	p.elements[len(p.elements)-1] += token
}
