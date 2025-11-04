package gcode

import (
	"bytes"
	"fmt"
	"io"
	"strconv"
)

// A Grbl system command (ie: $*)
type System string

// Word may either give a command or provide an argument to a command.
type Word struct {
	// A letter other than N
	Letter rune
	// The raw string value for the number.
	Number string
}

// ParseNumber parses the raw string Number and returns its float64 value.
func (w *Word) ParseNumber() (float64, error) {
	return strconv.ParseFloat(w.Number, 64)
}

func (w *Word) String() string {
	return string(w.Letter) + w.Number
}

// Block is a line which may include commands to do several different things.
type Block struct {
	System    *string
	Command   *Word
	Arguments []Word
}

func (b *Block) String() string {
	var buff bytes.Buffer

	if b.System != nil {
		buff.WriteString(*b.System)
	}
	if b.Command != nil {
		buff.WriteString(b.Command.String())
	}
	for _, arg := range b.Arguments {
		buff.WriteString(arg.String())
	}
	return buff.String()
}

// Empty returns true if no system or command is defined.
func (b *Block) Empty() bool {
	return b.System == nil && b.Command == nil
}

// Parser can parse G-Code in Grbl flavour.
type Parser struct {
	lexer *Lexer
}

func NewParser(r io.Reader) *Parser {
	return &Parser{
		lexer: NewLexer(r),
	}
}

// Next returns each parsed Block. When no more blocks are available, nil is returned.
func (p *Parser) Next() (*Block, error) {
	block := &Block{}
	var currentLetter *rune

	for {
		token, err := p.lexer.Next()
		if err != nil {
			return nil, err
		}
		if token.Type == TokenTypeEOF {
			return nil, nil
		}

		if token.Type == TokenTypeSpace || token.Type == TokenTypeComment {
			continue
		}

		if token.Type == TokenTypeSystem {
			valueStr := string(token.Value)
			block.System = &valueStr
			return block, nil
		}

		if token.Type == TokenTypeWordLetter {
			if currentLetter != nil {
				return nil, fmt.Errorf("line %d: unexpected word letter %q after previous letter", p.lexer.Line, string(token.Value))
			}
			r := rune(token.Value[0])
			currentLetter = &r
			continue
		}

		if token.Type == TokenTypeWordNumber {
			if currentLetter == nil {
				return nil, fmt.Errorf("line %d: unexpected word number %q without preceding letter", p.lexer.Line, string(token.Value))
			}
			word := Word{Letter: *currentLetter, Number: string(token.Value)}
			if block.Command == nil {
				block.Command = &word
			} else {
				block.Arguments = append(block.Arguments, word)
			}
			currentLetter = nil
			continue
		}

		if token.Type == TokenTypeNewLine {
			if currentLetter != nil {
				return nil, fmt.Errorf("line %d: unexpected word letter at end of line", p.lexer.Line-1)
			}
			if !block.Empty() {
				return block, nil
			}
		}
	}
}
