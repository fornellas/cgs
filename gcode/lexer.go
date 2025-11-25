package gcode

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
)

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
	Type  TokenType
	Value []byte
}

func (t *Token) String() string {
	return string(t.Value)
}

type Tokens []*Token

func (ts Tokens) String() string {
	var buf bytes.Buffer
	for _, token := range ts {
		fmt.Fprintf(&buf, "%s", token)
	}
	return buf.String()
}

// Lexer tokenizes G-Code in Grbl flavour.
type Lexer struct {
	// Line the lexer is in
	Line    uint
	scanner *bufio.Scanner
}

// NewLexer creates a new Lexer.
func NewLexer(r io.Reader) *Lexer {
	scanner := bufio.NewScanner(bufio.NewReader(r))
	scanner.Split(split)
	return &Lexer{Line: 1, scanner: scanner}
}

func isSpace(c byte) bool {
	return c == ' ' || c == '\t'
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

func splitSpace(data []byte) (advance int, token []byte, err error) {
	i := 0
	for i < len(data) && isSpace(data[i]) {
		i++
	}
	return i, data[:i], nil
}

func splitParenthesisComment(data []byte, atEOF bool) (advance int, token []byte, err error) {
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

func splitSemicolonComment(data []byte, atEOF bool) (advance int, token []byte, err error) {
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

func splitSystem(data []byte, atEOF bool) (advance int, token []byte, err error) {
	i := 1
	for i < len(data) {
		if isSpace(data[i]) || isCommentStart(data[i]) || isNewLineStart(data[i]) {
			return i, data[:i], nil
		}
		i++
	}
	if atEOF {
		return i, data[:i], nil
	}
	return 0, nil, nil
}

func splitNumber(data []byte, atEOF bool) (advance int, token []byte, err error) {
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
		return 0, nil, fmt.Errorf("invalid number: %s", data[:i])
	}
	if i < len(data) {
		if isSpace(data[i]) || isCommentStart(data[i]) || isLetterStart(data[i]) || isNewLineStart(data[i]) {
			return i, data[:i], nil
		}
		return 0, nil, fmt.Errorf("invalid number: %c", data[i])
	}
	if atEOF {
		return i, data[:i], nil
	}
	return 0, nil, nil
}

func split(data []byte, atEOF bool) (advance int, token []byte, err error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}

	if isSpace(data[0]) {
		return splitSpace(data)
	}

	if isParenthesisCommentStart(data[0]) {
		return splitParenthesisComment(data, atEOF)
	}
	if isSemicolonCommentStart(data[0]) {
		return splitSemicolonComment(data, atEOF)
	}

	if isSystemStart(data[0]) {
		return splitSystem(data, atEOF)
	}

	if isLetterStart(data[0]) {
		return 1, data[:1], nil
	}

	if isNumberStart(data[0]) {
		return splitNumber(data, atEOF)
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

	return 0, nil, fmt.Errorf("unexpected char: %#v", string(data[0]))
}

func (lx *Lexer) Next() (*Token, error) {
	if !lx.scanner.Scan() {
		if err := lx.scanner.Err(); err != nil {
			return nil, fmt.Errorf("line %d: %w", lx.Line, err)
		}
		return &Token{Type: TokenTypeEOF}, nil
	}

	value := lx.scanner.Bytes()
	if len(value) == 0 {
		panic(fmt.Sprintf("bug: empty token received at line %d", lx.Line))
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
		lx.Line++
		return &Token{Value: value, Type: TokenTypeNewLine}, nil
	}

	panic(fmt.Sprintf("bug: unexpected value at line %d: %#v", lx.Line, value))
}
