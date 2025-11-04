package gcode

import (
	"fmt"
	"io"
)

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
