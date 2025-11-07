package gcode

import (
	"bytes"
	"fmt"
	"math"
	"strconv"
	"unicode"
)

// Word may either give a command or provide an argument to a command.
type Word struct {
	letter rune
	number float64
	// The original string that declared this word. This is used to avoid parsing / serializing
	// upper/lowercase letters or float poont representation differences, for consistency on output.
	originalStr *string
}

// NewWord creates a Word from given letter and number.
// letter must be capitalised, or it'll panic.
func NewWord(letter rune, number float64) *Word {
	if letter < 'A' || letter > 'Z' {
		panic(fmt.Sprintf("bug: attempting to create word with letter not between A-Z: %c", letter))
	}
	return &Word{letter: letter, number: number}
}

// NewWordParse creates a Word from given letter other than N and a raw number string.
func NewWordParse(letter rune, number string) (*Word, error) {
	parsedNumber, err := strconv.ParseFloat(number, 64)
	if err != nil {
		return nil, err
	}
	normalizeLetter := unicode.ToUpper(letter)
	originalStr := string(letter) + number
	return &Word{letter: normalizeLetter, number: parsedNumber, originalStr: &originalStr}, nil
}

func (w *Word) Letter() rune {
	return w.letter
}

func (w *Word) Number() float64 {
	return w.number
}

func (w *Word) SetNumber(number float64) {
	w.number = number
	w.originalStr = nil
}

func (w *Word) Equal(ow *Word) bool {
	return w.NormalizedString() == ow.NormalizedString()
}

// String gives the representation of the word. If it has not been mutated, then it returns the
// exact original string (thus preserving letter casing and float point representation), otherwise
// it creates a new representation after the mutation.
func (w *Word) String() string {
	if w.originalStr != nil {
		return *w.originalStr
	}
	return w.NormalizedString()
}

// NormalizedString is similar to String(), but always return a consistent representation using
// uppercase letters, single point float precision for commands and 4 ponts precision for arguments.
func (w *Word) NormalizedString() string {
	if w.IsCommand() {
		int, frac := math.Modf(w.number)
		if frac == 0 {
			return fmt.Sprintf("%c%.0f", w.letter, int)
		} else {
			return fmt.Sprintf("%c%.1f", w.letter, w.number)
		}
	}
	return fmt.Sprintf("%c%.4f", w.letter, w.number)
}

// IsCommand returns true if the word is a command (letter G or M).
func (w *Word) IsCommand() bool {
	return w.letter == 'G' || w.letter == 'M'
}

// Block is a line which may include commands to do several different things.
type Block struct {
	system *string
	words  []*Word
}

func NewBlockSystem(system string) *Block {
	return &Block{system: &system}
}

func NewBlockCommand(words ...*Word) *Block {
	return &Block{words: words}
}

func (b *Block) IsSystem() bool {
	return b.system != nil
}

func (b *Block) IsCommand() bool {
	return len(b.words) > 0
}

func (b *Block) AppendCommandWords(words ...*Word) {
	if !b.IsCommand() {
		panic("bug: attempting to add word to a block that's not command")
	}
	b.words = append(b.words, words...)
}

func (b *Block) String() string {
	var buff bytes.Buffer
	if b.system != nil {
		buff.WriteString(string(*b.system))
	}
	for _, w := range b.words {
		buff.WriteString(w.String())
	}
	return buff.String()
}

// Commands returns all G/M words in the block.
func (b *Block) Commands() []*Word {
	var cmds []*Word
	for _, w := range b.words {
		if w.IsCommand() {
			cmds = append(cmds, w)
		}
	}
	return cmds
}

// Arguments returns all non-command words in the block.
func (b *Block) Arguments() []*Word {
	var args []*Word
	for _, w := range b.words {
		if !w.IsCommand() {
			args = append(args, w)
		}
	}
	return args
}

func (b *Block) GetArgumentNumber(letter rune) (*float64, error) {
	if !b.IsCommand() {
		panic("bug: can't fetch argument for system block")
	}
	var number *float64
	for _, w := range b.Arguments() {
		if w.Letter() == letter {
			if number != nil {
				return nil, fmt.Errorf("%s: multiple arguments for letter %c", b, letter)
			}
			n := w.Number()
			number = &n
		}
	}
	return number, nil
}

func (b *Block) SetArgumentNumber(letter rune, number float64) error {
	if !b.IsCommand() {
		return fmt.Errorf("%s: can't set argument for system block", b)
	}
	var set bool
	for _, w := range b.Arguments() {
		if w.Letter() == letter {
			if set {
				return fmt.Errorf("%s: duplicated letter %c", b, letter)
			}
			w.SetNumber(number)
			set = true
		}
	}
	return nil
}

// Empty returns true if no system or command is defined.
func (b *Block) Empty() bool {
	return b.system == nil && len(b.words) == 0
}

var rotateXYCommands = map[string]bool{
	"G0": true, // Coordinated Motion at Rapid Rate
	"G1": true, // Coordinated Motion at Feed Rate
}

var rotateXYIgnoreCommands = map[string]bool{
	"G4":  true, // Dwell
	"G17": true, // Plane Select XY
	"G21": true, // Units Millimeters
	"G53": true, // Move in machine coordinates
	"G90": true, // Distance Mode Absolute
	"G94": true, // Feed Rate Mode Units per Minute
	"M0":  true, // Program pause
	"M3":  true, // Spindle on (clockwise)
	"M5":  true, // Spindle stop
}

// RotateXY rotates work coordinates at the XY plane. Machine coordinates are not affected.
// cx and cy are the center coordinates for the rotation, radians is the angle (looking down at XY
// from Z positive to Z negative).
func (b *Block) RotateXY(cx, cy, radians float64) error {
	if b.system != nil {
		return fmt.Errorf("%s: can't rotate system commands", b)
	}
	var doRotation bool
	for _, w := range b.Commands() {
		commandStr := w.NormalizedString()
		if _, ok := rotateXYIgnoreCommands[commandStr]; ok {
			continue
		}
		if _, ok := rotateXYCommands[commandStr]; ok {
			doRotation = true
			continue
		}
		return fmt.Errorf("%s: rotation unsupported for command: %s", b, w)
	}
	if !doRotation {
		return nil
	}

	x, err := b.GetArgumentNumber('X')
	if err != nil {
		return err
	}
	y, err := b.GetArgumentNumber('Y')
	if err != nil {
		return err
	}
	if x == nil && y == nil {
		return nil
	}
	if x == nil {
		return fmt.Errorf("%s: rotation unsupported for X without Y", b)
	}
	if y == nil {
		return fmt.Errorf("%s: rotation unsupported for Y without X", b)
	}

	sin, cos := math.Sin(radians), math.Cos(radians)
	dx, dy := *x-cx, *y-cy
	rx := dx*cos - dy*sin + cx
	ry := dx*sin + dy*cos + cy

	if err := b.SetArgumentNumber('X', rx); err != nil {
		return err
	}
	if err := b.SetArgumentNumber('Y', ry); err != nil {
		return err
	}

	return nil
}

// type Program []Block
