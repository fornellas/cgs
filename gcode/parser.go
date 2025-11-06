package gcode

import (
	"fmt"
	"io"
)

// Parser can parse G-Code in Grbl flavour.
type Parser struct {
	lexer            *Lexer
	block            *Block
	words            []*Word
	currentRawLetter rune
}

func NewParser(r io.Reader) *Parser {
	return &Parser{
		lexer: NewLexer(r),
	}
}

func (p *Parser) handleTokenTypeEOF() (bool, error) {
	if p.currentRawLetter != 0 {
		return false, fmt.Errorf("line %d: unexpected word letter at end of file", p.lexer.Line)
	}
	if len(p.words) == 0 {
		return true, nil
	}
	p.block = NewBlockCommand(p.words...)
	return true, nil
}

func (p *Parser) handleTokenTypeLetter(token *Token) (bool, error) {
	if p.currentRawLetter != 0 {
		return false, fmt.Errorf("line %d: unexpected word letter %q after previous letter %q", p.lexer.Line, string(token.Value), string(p.currentRawLetter))
	}
	p.currentRawLetter = rune(token.Value[0])
	return false, nil
}

func (p *Parser) handleTokenTypeNumber(token *Token) (bool, error) {
	currentRawNumber := string(token.Value)
	if p.currentRawLetter == 0 {
		return false, fmt.Errorf("line %d: unexpected word number %q without preceding letter", p.lexer.Line, string(token.Value))
	}
	word, err := NewWordParse(p.currentRawLetter, currentRawNumber)
	if err != nil {
		return false, fmt.Errorf("line %d: bad number: %#v: %w", p.lexer.Line, string(token.Value), err)
	}
	p.words = append(p.words, word)
	p.currentRawLetter = 0
	return false, nil
}

func (p *Parser) handleTokenTypeNewLine() (bool, error) {
	if p.currentRawLetter != 0 {
		return false, fmt.Errorf("line %d: unexpected word letter at end of line", p.lexer.Line-1)
	}
	if len(p.words) > 0 || p.block != nil {
		if p.block == nil {
			p.block = NewBlockCommand(p.words...)
		} else {
			p.block.AppendCommandWords(p.words...)
		}
		return true, nil
	}
	return false, nil
}

func (p *Parser) handleToken(token *Token) (bool, error) {
	switch token.Type {
	case TokenTypeEOF:
		return p.handleTokenTypeEOF()
	case TokenTypeSpace, TokenTypeComment:
		return false, nil
	case TokenTypeSystem:
		p.block = NewBlockSystem(string(token.Value))
		return true, nil
	case TokenTypeWordLetter:
		return p.handleTokenTypeLetter(token)
	case TokenTypeWordNumber:
		return p.handleTokenTypeNumber(token)
	case TokenTypeNewLine:
		return p.handleTokenTypeNewLine()
	}
	return false, nil
}

// Next returns each parsed Block. When no more blocks are available, nil is returned.
func (p *Parser) Next() (*Block, error) {
	p.block = nil
	p.words = nil
	p.currentRawLetter = 0
	for {
		token, err := p.lexer.Next()
		if err != nil {
			return nil, err
		}
		done, err := p.handleToken(token)
		if err != nil {
			return nil, err
		}
		if done {
			return p.block, nil
		}
	}
}
