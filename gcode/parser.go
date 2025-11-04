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

func (p *Parser) handleToken(
	token *Token,
	block *Block,
	words *[]*Word,
	currentRawLetter *rune,
) (bool, error) {
	switch token.Type {
	case TokenTypeEOF:
		if *currentRawLetter != 0 {
			return false, fmt.Errorf("line %d: unexpected word letter at end of file", p.lexer.Line)
		}
		if len(*words) == 0 {
			return true, nil
		}
		block.Words = *words
		return true, nil
	case TokenTypeSpace, TokenTypeComment:
		return false, nil
	case TokenTypeSystem:
		valueStr := string(token.Value)
		block.System = &valueStr
		return false, nil
	case TokenTypeWordLetter:
		if *currentRawLetter != 0 {
			return false, fmt.Errorf("line %d: unexpected word letter %q after previous letter %q", p.lexer.Line, string(token.Value), string(*currentRawLetter))
		}
		*currentRawLetter = rune(token.Value[0])
		return false, nil
	case TokenTypeWordNumber:
		currentRawNumber := string(token.Value)
		if *currentRawLetter == 0 {
			return false, fmt.Errorf("line %d: unexpected word number %q without preceding letter", p.lexer.Line, string(token.Value))
		}
		word, err := NewWord(*currentRawLetter, currentRawNumber)
		if err != nil {
			return false, fmt.Errorf("line %d: bad number: %#v: %w", p.lexer.Line, string(token.Value), err)
		}
		*words = append(*words, word)
		*currentRawLetter = 0
		return false, nil
	case TokenTypeNewLine:
		if *currentRawLetter != 0 {
			return false, fmt.Errorf("line %d: unexpected word letter at end of line", p.lexer.Line-1)
		}
		if len(*words) > 0 || block.System != nil {
			block.Words = *words
			return true, nil
		}
		return false, nil
	}
	return false, nil
}

// Next returns each parsed Block. When no more blocks are available, nil is returned.
func (p *Parser) Next() (*Block, error) {
	block := &Block{}
	var words []*Word
	var currentRawLetter rune

	for {
		token, err := p.lexer.Next()
		if err != nil {
			return nil, err
		}
		done, err := p.handleToken(token, block, &words, &currentRawLetter)
		if err != nil {
			return nil, err
		}
		if done {
			if block.Empty() {
				return nil, nil
			}
			return block, nil
		}
	}
}
