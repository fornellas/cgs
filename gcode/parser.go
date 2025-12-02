package gcode

import (
	"errors"
	"fmt"
	"io"
)

// Modal Groups state.
// See https://www.linuxcnc.org/docs/2.4/html/gcode_overview.html#sec:Modal-Groups and
// https://github.com/gnea/grbl/wiki/Grbl-v1.1-Commands
type ModalGroup struct {
	// Motion (Group 1)
	Motion *Word

	// Plane selection (Group 2)
	PlaneSelection *Word

	// Distance Mode (Group 3)
	DistanceMode *Word

	// Arc IJK Distance Mode (Group 4)
	ArcIjkDistanceMode *Word

	// Feed Rate Mode (Group 5)
	FeedRateMode *Word

	// Units (Group 6)
	Units *Word

	// Cutter Diameter Compensation (Group 7)
	CutterDiameterCompensation *Word

	// Tool Length Offset (Group 8)
	ToolLengthOffset *Block

	// Coordinate System Select (Group 12)
	CoordinateSystemSelect *Word

	// Control Mode (Group 13)
	ControlMode *Word

	// Stopping (Group 4)
	Stopping *Word

	// Spindle (Group 7)
	Spindle *Word

	// Coolant (Group 8)
	Coolant []*Word

	// Override Control (Grbl specific - M56)
	// OverrideControl *Block
}

func (m *ModalGroup) Copy() *ModalGroup {
	nm := *m
	copy(nm.Coolant, m.Coolant)
	return &nm
}

//gocyclo:ignore
func (m *ModalGroup) UpdateFromWord(word *Word) error {
	switch word.NormalizedString() {
	case "G0", "G1", "G2", "G3", "G38.2", "G38.3", "G38.4", "G38.5", "G80":
		m.Motion = word
	case "G17", "G18", "G19":
		m.PlaneSelection = word
	case "G90", "G91":
		m.DistanceMode = word
	case "G91.1":
		m.ArcIjkDistanceMode = word
	case "G93", "G94":
		m.FeedRateMode = word
	case "G20", "G21":
		m.Units = word
	case "G40":
		m.CutterDiameterCompensation = word
	case "G43.1":
		return errors.New("can't update from word G43.1: it must be from a block with Z axis")
	case "G49":
		m.ToolLengthOffset = NewBlockCommand(word)
	case "G54", "G55", "G56", "G57", "G58", "G59":
		m.CoordinateSystemSelect = word
	case "G61":
		m.ControlMode = word
	case "M0", "M1", "M2", "M30":
		m.Stopping = word
	case "M3", "M4", "M5":
		m.Spindle = word
	case "M7", "M8":
		skip := false
		for _, w := range m.Coolant {
			if w.NormalizedString() == word.NormalizedString() {
				skip = true
				break
			}
		}
		if skip {
			break
		}
		newCoolant := []*Word{}
		for _, w := range m.Coolant {
			if w.NormalizedString() == "M9" {
				continue
			}
			newCoolant = append(newCoolant, w)
		}
		m.Coolant = newCoolant
		m.Coolant = append(m.Coolant, word)
	case "M9":
		m.Coolant = []*Word{word}
	}

	return nil
}

func (m *ModalGroup) UpdateFromBlock(block *Block) error {
	for _, word := range block.Commands() {
		if word.NormalizedString() == "G43.1" {
			var z *float64
			for _, argWord := range block.Arguments() {
				if argWord.Letter() == 'Z' {
					zv := argWord.Number()
					z = &zv
				}
			}
			if z == nil {
				return fmt.Errorf("G43.1 requires Z argument")
			}
			m.ToolLengthOffset = NewBlockCommand(NewWord('G', 43.1), NewWord('Z', *z))
		} else {
			if err := m.UpdateFromWord(word); err != nil {
				return err
			}
		}
	}
	return nil
}

// DefaultModalGroup holds Grbl default modal group states.
// See: https://github.com/gnea/grbl/wiki/Grbl-v1.1-Commands.
var DefaultModalGroup ModalGroup = ModalGroup{
	Motion:                     NewWord('G', 0),
	PlaneSelection:             NewWord('G', 17),
	DistanceMode:               NewWord('G', 90),
	ArcIjkDistanceMode:         NewWord('G', 91.1),
	FeedRateMode:               NewWord('G', 94),
	Units:                      NewWord('G', 21),
	CutterDiameterCompensation: NewWord('G', 40),
	ToolLengthOffset:           NewBlockCommand(NewWord('G', 49)),
	CoordinateSystemSelect:     NewWord('G', 54),
	ControlMode:                NewWord('G', 61),
	Stopping:                   nil,
	Spindle:                    NewWord('M', 5),
	Coolant:                    []*Word{NewWord('M', 9)},
}

// Parser can parse Grbl flavour G-Code.
type Parser struct {
	// ModalGroup holds the state of each modal group as parsing progresses by caling Parser.Next().
	// DefaultModalGroup is used for the initial state.
	ModalGroup ModalGroup
	Lexer      *Lexer
	block      *Block
	words      []*Word
	letter     *rune
}

func NewParser(r io.Reader) *Parser {
	return &Parser{
		ModalGroup: DefaultModalGroup,
		Lexer:      NewLexer(r),
	}
}

func (p *Parser) handleTokenTypeEOF() (bool, error) {
	if p.letter != nil {
		return false, fmt.Errorf("line %d: unexpected word letter at end of file", p.Lexer.Line)
	}
	if p.block != nil {
		return true, nil
	}
	if len(p.words) == 0 {
		return true, nil
	}
	p.block = NewBlockCommand(p.words...)
	return true, nil
}

func (p *Parser) handleTokenTypeLetter(token *Token) (bool, error) {
	if p.letter != nil {
		return false, fmt.Errorf("line %d: unexpected word letter %q after previous letter %q", p.Lexer.Line, token, string(*p.letter))
	}
	letter := rune(token.Value[0])
	p.letter = &letter
	return false, nil
}

func (p *Parser) handleTokenTypeNumber(token *Token) (bool, error) {
	number := string(token.Value)
	if p.letter == nil {
		return false, fmt.Errorf("line %d: unexpected word number %q without preceding letter", p.Lexer.Line, string(token.Value))
	}
	word, err := NewWordParse(*p.letter, number)
	if err != nil {
		return false, fmt.Errorf("line %d: bad number: %#v: %w", p.Lexer.Line, string(token.Value), err)
	}
	p.words = append(p.words, word)
	p.letter = nil
	return false, nil
}

func (p *Parser) handleTokenTypeNewLine() (bool, error) {
	if p.letter != nil {
		return false, fmt.Errorf("line %d: unexpected word letter at end of line", p.Lexer.Line-1)
	}
	if len(p.words) > 0 || p.block != nil {
		if p.block == nil {
			p.block = NewBlockCommand(p.words...)
		} else {
			if len(p.words) > 0 {
				if !p.block.IsCommand() {
					panic(fmt.Sprintf("bug: pending words for non-command block: %#v, %#v", p.words, p.block))
				}
				p.block.AppendCommandWords(p.words...)
			}
		}
	}
	return true, nil
}

func (p *Parser) handleToken(token *Token) (bool, error) {
	switch token.Type {
	case TokenTypeEOF:
		return p.handleTokenTypeEOF()
	case TokenTypeSpace, TokenTypeComment:
		return false, nil
	case TokenTypeSystem:
		if len(p.words) > 0 || p.letter != nil {
			return false, fmt.Errorf("line %d: system command cannot follow command words", p.Lexer.Line)
		}
		p.block = NewBlockSystem(string(token.Value))
		return false, nil
	case TokenTypeWordLetter:
		return p.handleTokenTypeLetter(token)
	case TokenTypeWordNumber:
		return p.handleTokenTypeNumber(token)
	case TokenTypeNewLine:
		return p.handleTokenTypeNewLine()
	default:
		panic(fmt.Sprintf("unknown token type: %#v", token))
	}
}

// Next returns the next parsed line. The first returned bool indicates EOF: when true, parsing is
// complete. If the line contained a block, it is returned. Tokens contains all tokens for the
// parsed line.
func (p *Parser) Next() (bool, *Block, Tokens, error) {
	p.block = nil
	p.words = nil
	p.letter = nil
	var tokens Tokens
	for {
		token, err := p.Lexer.Next()
		if err != nil {
			return false, nil, nil, err
		}
		tokens = append(tokens, token)
		eol, err := p.handleToken(token)
		if err != nil {
			return false, nil, nil, err
		}
		if eol {
			if p.block != nil {
				if err := p.ModalGroup.UpdateFromBlock(p.block); err != nil {
					return false, nil, nil, err
				}
			}
			return token.Type == TokenTypeEOF, p.block, tokens, nil
		}
	}
}

// Blocks parses and returns all remaining blocks from the parser.
// It calls Next() repeatedly until all blocks are consumed or an error occurs.
// Returns a slice of all parsed blocks, or an error if parsing fails.
func (p *Parser) Blocks() ([]*Block, error) {
	blocks := []*Block{}
	for {
		eof, block, _, err := p.Next()
		if err != nil {
			return nil, err
		}
		if block != nil {
			blocks = append(blocks, block)
		}
		if eof {
			return blocks, nil
		}
	}
}
