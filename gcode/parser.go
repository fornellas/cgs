package gcode

import (
	"fmt"
	"io"
)

// Modal Groups state.
// See https://www.linuxcnc.org/docs/2.4/html/gcode_overview.html#sec:Modal-Groups and
// https://github.com/gnea/grbl/wiki/Grbl-v1.1-Commands
type ModalGroup struct {
	// Motion ("Group 1")
	Motion *Word

	// Plane selection
	PlaneSelection *Word

	// Diameter / Radius for lathes
	// DiameterRadiusForLathes *Word

	// Distance Mode
	DistanceMode *Word

	// Feed Rate Mode
	FeedRateMode *Word

	// Units
	Units *Word

	// Cutter Radius Compensation
	CutterRadiusCompensation *Word

	// Tool Length Offset
	ToolLengthOffset *Word

	// Return Mode in Canned Cycles
	// ReturnModeInCannedCycles *Word

	// Coordinate System Selection
	CoordinateSystemSelection *Word

	// Stopping
	Stopping *Word

	// Tool Change
	// ToolChange *Word

	// Spindle Turning
	SpindleTurning *Word

	// Coolant
	Coolant []*Word

	// Override Switches
	// OverrideSwitches *Word

	// Flow Control
	// FlowControl *Word
}

func (m *ModalGroup) Copy() *ModalGroup {
	nm := *m
	copy(nm.Coolant, m.Coolant)
	return &nm
}

//gocyclo:ignore
func (m *ModalGroup) Update(word *Word) {
	switch word.NormalizedString() {
	case "G0", "G1", "G2", "G3", "G33", "G38.2", "G38.3", "G38.4", "G38.5", "G73", "G76", "G80", "G81", "G82", "G83", "G84", "G85", "G86", "G87", "G88", "G89":
		m.Motion = word
	case "G17", "G18", "G19":
		m.PlaneSelection = word
	// DiameterRadiusForLathes
	// G7, G8
	case "G90", "G91":
		m.DistanceMode = word
	case "G93", "G94":
		m.FeedRateMode = word
	case "G20", "G21":
		m.Units = word
	case "G40", "G41", "G42", "G41.1", "G42.1":
		m.CutterRadiusCompensation = word
	case "G43", "G43.1", "G49":
		m.ToolLengthOffset = word
	// ReturnModeInCannedCycles
	// G98, G99
	case "G54", "G55", "G56", "G57", "G58", "G59", "G59.1", "G59.2", "G59.3":
		m.CoordinateSystemSelection = word
	case "M0", "M1", "M2", "M30", "M60":
		m.Stopping = word
	// ToolChange
	// M6 Tn
	case "M3", "M4", "M5":
		m.SpindleTurning = word
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
		m.Coolant = append(m.Coolant, word)
	case "M9":
		m.Coolant = []*Word{word}
		// OverrideSwitches
		// M48, M49
		// FlowControl
		// O-
	}
}

// DefaultModalGroup holds Grbl default modal group states.
// See: https://github.com/gnea/grbl/wiki/Grbl-v1.1-Commands.
var DefaultModalGroup ModalGroup = ModalGroup{
	// Motion ("Group 1")
	Motion: NewWord('G', 0),

	// Plane selection
	PlaneSelection: NewWord('G', 17),

	// Diameter / Radius for lathes
	// DiameterRadiusForLathes: NewWord('G', 8),

	// Distance Mode
	DistanceMode: NewWord('G', 90),

	// Feed Rate Mode
	FeedRateMode: NewWord('G', 94),

	// Units
	Units: NewWord('G', 21),

	// Cutter Radius Compensation
	// CutterRadiusCompensation: NewWord('', ),

	// Tool Length Offset
	ToolLengthOffset: NewWord('G', 49),

	// Return Mode in Canned Cycles
	// ReturnModeInCannedCycles: NewWord('', ),

	// Coordinate System Selection
	CoordinateSystemSelection: NewWord('G', 54),

	// Stopping
	Stopping: NewWord('M', 0),

	// Tool Change
	// ToolChange: NewWord('', ),

	// Spindle Turning
	SpindleTurning: NewWord('M', 5),

	// Coolant
	Coolant: []*Word{NewWord('M', 9)},

	// Override Switches
	// OverrideSwitches: NewWord('', ),

	// Flow Control
	// FlowControl: NewWord('', ),
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

//gocyclo:ignore
func (p *Parser) updateModalGroups(block *Block) {
	for _, word := range block.Commands() {
		p.ModalGroup.Update(word)
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
				p.updateModalGroups(p.block)
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
		if eof {
			return blocks, nil
		}
		if block == nil {
			continue
		}
		blocks = append(blocks, block)
	}
}
