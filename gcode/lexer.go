package gcode

import (
	"bufio"
	"errors"
	"fmt"
	"io"
)

const grblMaxIntDigits = 8 // Grbl's MAX_INT_DIGITS is 8

type TokenType int

const (
	TokenTypeEOF TokenType = iota
	TokenTypeSpace
	TokenTypeComment
	TokenTypeSystem
	TokenTypeWordLetter
	TokenTypeWordNumber
	TokenTypeNewLine
)

var tokenTypeNames = map[TokenType]string{
	TokenTypeEOF:        "EOF",
	TokenTypeSpace:      "Space",
	TokenTypeComment:    "Comment",
	TokenTypeSystem:     "System",
	TokenTypeWordLetter: "WordLetter",
	TokenTypeWordNumber: "WordNumber",
	TokenTypeNewLine:    "NewLine",
}

func (tt TokenType) String() string {
	if name, ok := tokenTypeNames[tt]; ok {
		return name
	}
	panic(fmt.Sprintf("unexpected TokenType: %d", tt))
}

type Token struct {
	Value string
	Type  TokenType
}

// Lexer tokenizes G-Code. Its implementation is derived directly from Grbl source code.
type Lexer struct {
	lines   uint
	scanner *bufio.Scanner
}

// NewLexer creates a new Lexer.
func NewLexer(rd io.Reader) *Lexer {
	scanner := bufio.NewScanner(bufio.NewReader(rd))
	scanner.Split(split)
	return &Lexer{scanner: scanner}
}

func isSpace(c byte) bool {
	return c == ' '
}

func isParenthesisCommentStart(c byte) bool {
	return c == '('
}

func isSemicolonCommentStart(c byte) bool {
	return c == ';'
}

func isCommentStart(c byte) bool {
	return isParenthesisCommentStart(c) || isSemicolonCommentStart(c)
}

func isSystemStart(c byte) bool {
	return c == '$'
}

func isLetterStart(c byte) bool {
	return (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z')
}

func isNumberStart(c byte) bool {
	return c == '-' || c == '+' || (c >= '0' && c <= '9')
}

func isNewLineStart(c byte) bool {
	return c == '\n' || c == '\r'
}

func split(data []byte, atEOF bool) (advance int, token []byte, err error) {
	// EOF
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}

	// Space
	if isSpace(data[0]) {
		i := 0
		for i < len(data) && isSpace(data[i]) {
			i++
		}
		return i, data[:i], nil
	}

	// Comment
	if isParenthesisCommentStart(data[0]) {
		i := 1
		for i < len(data) {
			if data[i] == ')' {
				i++
				return i, data[:i], nil
			}
			if data[i] == '\n' {
				return 0, nil, errors.New("end of line reached without closing parenthesis")
			}
			i++
		}
		if atEOF {
			return 0, nil, errors.New("end of file reached without closing parenthesis")
		}
		return 0, nil, nil
	}
	if isSemicolonCommentStart(data[0]) {
		i := 1
		for i < len(data) {
			if data[i] == '\n' {
				if i > 1 && data[i-1] == '\r' {
					i--
					return i, data[:i], nil
				}
				return i, data[:i], nil
			}
			i++
		}
		if atEOF {
			return len(data), data, nil
		}
		return 0, nil, nil
	}

	// System
	if isSystemStart(data[0]) {
		i := 1
		for i < len(data) {
			if data[i] == '\n' {
				if i > 1 && data[i-1] == '\r' {
					i--
					return i, data[:i], nil
				}
				return i, data[:i], nil
			}
			i++
		}
		if atEOF {
			return len(data), data, nil
		}
		return 0, nil, nil
	}

	// WordLetter
	if isLetterStart(data[0]) {
		return 1, data[:1], nil
	}

	// WordNumber
	if isNumberStart(data[0]) {
		i := 0
		if data[i] == '-' || data[i] == '+' {
			i++
		}
		ndigit := 0
		isdecimal := false
		for i < len(data) {
			c := data[i]
			if c >= '0' && c <= '9' {
				ndigit++
				i++
			} else if c == '.' && !isdecimal {
				isdecimal = true
				i++
			} else {
				break
			}
		}
		if ndigit == 0 {
			return 0, nil, fmt.Errorf("Invalid number: %s", data[:i])
		} else {
			return i, data[:i], nil
		}
	}

	// NewLine
	if data[0] == '\n' {
		return 1, data[:1], nil
	}
	if data[0] == '\r' {
		if len(data) > 1 {
			if data[1] == '\n' {
				return 2, data[:2], nil
			}
		} else {
			if atEOF {
				return 0, nil, fmt.Errorf("CR before EOF")
			}
		}
	}

	return 0, nil, fmt.Errorf("unexpected char: %v", data[0])
}

func (lx *Lexer) Next() (*Token, error) {
	if !lx.scanner.Scan() {
		if err := lx.scanner.Err(); err != nil {
			return nil, fmt.Errorf("Line %d: %w", lx.lines, err)
		}
		return &Token{Type: TokenTypeEOF}, nil
	}

	value := lx.scanner.Text()
	if len(value) == 0 {
		panic(fmt.Sprintf("bug: empty token received at line %d", lx.lines))
	}

	if isSpace(value[0]) {
		return &Token{Value: value, Type: TokenTypeSpace}, nil
	}

	if isCommentStart(value[0]) {
		return &Token{Value: value, Type: TokenTypeComment}, nil
	}

	if isSystemStart(value[0]) {
		return &Token{Value: value, Type: TokenTypeSystem}, nil
	}

	if isLetterStart(value[0]) {
		return &Token{Value: value, Type: TokenTypeWordLetter}, nil
	}

	if isNumberStart(value[0]) {
		return &Token{Value: value, Type: TokenTypeWordNumber}, nil
	}

	if isNewLineStart(value[0]) {
		lx.lines++
		return &Token{Value: value, Type: TokenTypeNewLine}, nil
	}

	panic(fmt.Sprintf("bug: unexpected value at line %d: %v", lx.lines, value))
}
