package gcode

import (
	"bytes"
	"fmt"
	"strconv"
	"unicode"
)

// A Grbl system command (ie: $*)
type System string

// Word may either give a command or provide an argument to a command.
type Word struct {
	letter rune
	number float64
	// The original string that declared this word. This is used to avoid parsing / serializing
	// upper/lowercase letters or float poont representation differences, for consistency on output.
	originalStr *string
}

// NewWord creates a Word from given letter other than N and a raw number string.
func NewWord(letter rune, number string) (*Word, error) {
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
		return fmt.Sprintf("%c%.1f", w.letter, w.number)
	}
	return fmt.Sprintf("%c%.4f", w.letter, w.number)
}

// IsCommand returns true if the word is a command (letter G or M).
func (w *Word) IsCommand() bool {
	return w.letter == 'G' || w.letter == 'M'
}

// Block is a line which may include commands to do several different things.
type Block struct {
	System *string
	Words  []*Word
}

func (b *Block) String() string {
	var buff bytes.Buffer
	if b.System != nil {
		buff.WriteString(*b.System)
	}
	for _, w := range b.Words {
		buff.WriteString(w.String())
	}
	return buff.String()
}

// Commands returns all G/M words in the block.
func (b *Block) Commands() []*Word {
	var cmds []*Word
	for _, w := range b.Words {
		if w.IsCommand() {
			cmds = append(cmds, w)
		}
	}
	return cmds
}

// Arguments returns all non-command words in the block.
func (b *Block) Arguments() []*Word {
	var args []*Word
	for _, w := range b.Words {
		if !w.IsCommand() {
			args = append(args, w)
		}
	}
	return args
}

// Empty returns true if no system or command is defined.
func (b *Block) Empty() bool {
	return b.System == nil && len(b.Words) == 0
}

type Program []Block
