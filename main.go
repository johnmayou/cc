package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

const (
	KiB = 1024
	MiB = 1024 * KiB
)

var rootCmd = &cobra.Command{
	Use: "json-parse",
	RunE: func(cmd *cobra.Command, args []string) error {
		if len(args) < 1 {
			return errors.New("missing file argument")
		}

		_, err := os.Stat(args[0])
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return errors.New("input file does not exist")
			} else {
				return fmt.Errorf("error reading input file: %w", err)
			}
		}

		inputBytes, err := os.ReadFile(args[0])
		if err != nil {
			return fmt.Errorf("error reading input file: %w", err)
		}

		input := strings.TrimSpace(string(inputBytes))
		if input == "" {
			return fmt.Errorf("input file empty")
		}

		tokens, err := Tokenize(input)
		if err != nil {
			return fmt.Errorf("error tokenizing input: %w", err)
		}

		err = NewParser(tokens).Parse()
		if err != nil {
			return fmt.Errorf("error parsing tokens: %w", err)
		}

		return nil
	},
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

type TokenType int

const (
	CurlyBracketOpen = iota
	CurlyBracketClosed
	BracketOpen
	BracketClosed
	Colon
	Comma
	String
	Number
	True
	False
	Null
)

func (t TokenType) String() string {
	switch t {
	case CurlyBracketOpen:
		return "CurlyBracketOpen"
	case CurlyBracketClosed:
		return "CurlyBracketClosed"
	case BracketOpen:
		return "BracketOpen"
	case BracketClosed:
		return "BracketClosed"
	case Colon:
		return "Colon"
	case Comma:
		return "Comma"
	case String:
		return "String"
	case Number:
		return "Number"
	case True:
		return "True"
	case False:
		return "False"
	case Null:
		return "Null"
	default:
		return fmt.Sprintf("Unknown(%d)", t)
	}
}

type Token struct {
	Type  TokenType
	Start int
	Stop  int
}

func Tokenize(str string) (tokens []Token, err error) {
	pos := 0
	for {
		if pos >= len(str) {
			break
		}

		inrange := func(position int) bool {
			return position < len(str)
		}

		switch {
		case str[pos] == ' ' || str[pos] == '\n':
			pos++
		case str[pos] == '{':
			tokens = append(tokens, Token{Type: CurlyBracketOpen, Start: pos, Stop: pos})
			pos++
		case str[pos] == '}':
			tokens = append(tokens, Token{Type: CurlyBracketClosed, Start: pos, Stop: pos})
			pos++
		case str[pos] == '[':
			tokens = append(tokens, Token{Type: BracketOpen, Start: pos, Stop: pos})
			pos++
		case str[pos] == ']':
			tokens = append(tokens, Token{Type: BracketClosed, Start: pos, Stop: pos})
			pos++
		case str[pos] == ':':
			tokens = append(tokens, Token{Type: Colon, Start: pos, Stop: pos})
			pos++
		case str[pos] == ',':
			tokens = append(tokens, Token{Type: Comma, Start: pos, Stop: pos})
			pos++
		case str[pos] == '"':
			end := pos + 1
			for {
				if str[end] == '"' {
					break
				}

				if str[end] == '\\' {
					if !inrange(end+1) || !isValidEscape(str[end+1]) {
						return nil, fmt.Errorf("invalid escape character in string")
					}
				}

				// Control character.
				if str[end] < 0x20 {
					return nil, fmt.Errorf("invalid character in string. control characters must be escaped")
				}

				if inrange(end + 1) {
					end++
				} else {
					return nil, fmt.Errorf("double quote not closed starting at %d", pos)
				}
			}

			tokens = append(tokens, Token{Type: String, Start: pos, Stop: end})
			pos = end + 1
		case '0' <= str[pos] && str[pos] <= '9':
			if str[pos] == '0' {
				return nil, fmt.Errorf("numbers cannot have leading zeroes")
			}

			end := pos + 1
			for {
				if '0' <= str[end] && str[end] <= '9' {
					if inrange(end + 1) {
						end++
					} else {
						break
					}
				} else {
					end--
					break
				}
			}

			tokens = append(tokens, Token{Type: Number, Start: pos, Stop: end})
			pos = end + 1
		case inrange(pos+3) && str[pos:pos+4] == "true":
			tokens = append(tokens, Token{Type: True, Start: pos, Stop: pos + 3})
			pos += 4
		case inrange(pos+4) && str[pos:pos+5] == "false":
			tokens = append(tokens, Token{Type: False, Start: pos, Stop: pos + 4})
			pos += 5
		case inrange(pos+3) && str[pos:pos+4] == "null":
			tokens = append(tokens, Token{Type: Null, Start: pos, Stop: pos + 3})
			pos += 4
		default:
			return nil, fmt.Errorf("invalid scenario for string: %q", str[pos:])
		}
	}

	return tokens, nil
}

func isValidEscape(c byte) bool {
	switch c {
	case '"', '\\', '/', 'b', 'f', 'n', 'r', 't', 'u':
		return true
	default:
		return false
	}
}

const maxDepth = 19

type Parser struct {
	tokens []Token
	pos    int

	depth int
}

func NewParser(tokens []Token) *Parser {
	return &Parser{tokens: tokens, pos: 0, depth: 0}
}

func (p *Parser) Parse() error {
	for {
		if p.done() {
			break
		}

		curr := p.tokens[p.pos]
		switch {
		case curr.Type == CurlyBracketOpen:
			if err := p.parseObject(); err != nil {
				return err
			}
		case curr.Type == BracketOpen:
			if err := p.parseArray(); err != nil {
				return err
			}
		default:
			return fmt.Errorf("invalid top level token: %s", curr.Type)
		}
	}

	return nil
}

func (p *Parser) parseObject() error {
	p.depth++
	if p.depth > maxDepth {
		return fmt.Errorf("nesting too deep")
	}
	defer func() { p.depth-- }()

	// Opening.
	if _, err := p.consume(CurlyBracketOpen); err != nil {
		return err
	}

	// Might be closing immediately.
	if curr, ok := p.curr(); ok {
		if curr.Type == CurlyBracketClosed {
			_, err := p.consume(curr.Type)
			return err
		}
	} else {
		return fmt.Errorf("expected token after opening object bracket but found nothing")
	}

	for {
		// Key.
		if _, err := p.consume(String); err != nil {
			return err
		}

		// Colon.
		if _, err := p.consume(Colon); err != nil {
			return err
		}

		// Value.
		if err := p.parseValue(); err != nil {
			return err
		}

		// Comma (continue) or closing curly bracket (end of object).
		if curr, ok := p.curr(); ok {
			switch curr.Type {
			case Comma:
				if _, err := p.consume(curr.Type); err != nil {
					return err
				}
			case CurlyBracketClosed:
				_, err := p.consume(curr.Type)
				return err
			default:
				return fmt.Errorf("object closing bracket or comma expected but got %s: \n%s", curr.Type, p.prettyTokens())
			}
		} else {
			return fmt.Errorf("object closing bracket or comma expected but found nothing: \n%s", p.prettyTokens())
		}
	}
}

func (p *Parser) parseArray() error {
	p.depth++
	if p.depth > maxDepth {
		return fmt.Errorf("nesting too deep")
	}
	defer func() { p.depth-- }()

	// Opening.
	if _, err := p.consume(BracketOpen); err != nil {
		return err
	}

	for {
		curr, ok := p.curr()
		if !ok {
			return fmt.Errorf("array closing bracket expected but found nothing")
		}
		if curr.Type == BracketClosed {
			_, err := p.consume(curr.Type)
			return err
		}
		if err := p.parseValue(); err != nil {
			return err
		}
	}
}

func (p *Parser) parseValue() error {
	curr, ok := p.curr()
	if !ok {
		return fmt.Errorf("expected object value but found nothing")
	}

	switch {
	case curr.Type == CurlyBracketOpen:
		return p.parseObject()
	case curr.Type == BracketOpen:
		return p.parseArray()
	case curr.Type == String || curr.Type == Number || curr.Type == True || curr.Type == False || curr.Type == Null:
		_, err := p.consume(curr.Type)
		return err
	default:
		return fmt.Errorf("invalid value token: %s", curr.Type)
	}
}

func (p *Parser) curr() (Token, bool) {
	if p.done() {
		var zero Token
		return zero, false
	}

	return p.tokens[p.pos], true
}

func (p *Parser) consume(tt TokenType) (Token, error) {
	if p.done() {
		var zero Token
		return zero, fmt.Errorf("no more tokens to consume, did not find %s", tt)
	}

	curr := p.tokens[p.pos]
	if curr.Type != tt {
		var zero Token
		return zero, fmt.Errorf("expected to consume %s but found %s", tt, curr.Type)
	}

	p.pos++
	return curr, nil
}

func (p *Parser) done() bool {
	return p.pos >= len(p.tokens)
}

func (p *Parser) prettyTokens() string {
	pretty := make([]string, 0, len(p.tokens))

	for i, token := range p.tokens {
		tt := token.Type.String()
		if i == p.pos {
			tt = ">>> " + tt + " <<<"
		}

		pretty = append(pretty, tt)
	}

	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	if err := enc.Encode(pretty); err != nil {
		panic(fmt.Errorf("error marshaling tokens: %w", err))
	}
	return buf.String()
}

type Stack[T any] struct {
	items []T
}

func (s *Stack[T]) Push(item T) {
	s.items = append(s.items, item)
}

func (s *Stack[T]) Pop() (T, bool) {
	if s.IsEmpty() {
		var zero T
		return zero, false
	}

	top := s.items[len(s.items)-1]
	s.items = s.items[:len(s.items)-1]
	return top, true
}

func (s *Stack[T]) Peek() (T, bool) {
	if s.IsEmpty() {
		var zero T
		return zero, false
	}
	return s.items[len(s.items)-1], true
}

func (s *Stack[T]) IsEmpty() bool {
	return len(s.items) == 0
}
