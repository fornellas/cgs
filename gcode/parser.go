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

func (p *Parser) handleSystemToken(block *Block, token *Token) (*Block, error) {
	valueStr := string(token.Value)
	block.System = &valueStr
	return block, nil
}

func (p *Parser) handleWordLetterToken(currentLetter *rune, token *Token) (*rune, error) {
	if currentLetter != nil {
		return nil, fmt.Errorf("line %d: unexpected word letter %q after previous letter", p.lexer.Line, string(token.Value))
	}
	r := rune(token.Value[0])
	return &r, nil
}

func (p *Parser) handleWordNumberToken(block *Block, currentLetter *rune, token *Token) (*Block, *rune, error) {
	if currentLetter == nil {
		return nil, nil, fmt.Errorf("line %d: unexpected word number %q without preceding letter", p.lexer.Line, string(token.Value))
	}
	word, err := NewWord(*currentLetter, string(token.Value))
	if err != nil {
		return nil, nil, fmt.Errorf("line %d: bad number: %#v: %w", p.lexer.Line, string(token.Value), err)
	}
	if block.Command == nil {
		block.Command = word
	} else {
		block.Arguments = append(block.Arguments, word)
	}
	return block, nil, nil
}

func (p *Parser) handleNewLineToken(currentLetter *rune) error {
	if currentLetter != nil {
		return fmt.Errorf("line %d: unexpected word letter at end of line", p.lexer.Line-1)
	}
	return nil
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
		switch token.Type {
		case TokenTypeEOF:
			return nil, nil
		case TokenTypeSpace, TokenTypeComment:
			continue
		case TokenTypeSystem:
			return p.handleSystemToken(block, token)
		case TokenTypeWordLetter:
			var err error
			currentLetter, err = p.handleWordLetterToken(currentLetter, token)
			if err != nil {
				return nil, err
			}
			continue
		case TokenTypeWordNumber:
			var err error
			block, currentLetter, err = p.handleWordNumberToken(block, currentLetter, token)
			if err != nil {
				return nil, err
			}
			continue
		case TokenTypeNewLine:
			if err := p.handleNewLineToken(currentLetter); err != nil {
				return nil, err
			}
			if !block.Empty() {
				return block, nil
			}
		}
	}
}
