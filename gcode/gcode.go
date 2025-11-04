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
	letter    rune
	rawLetter rune
	number    float64
	rawNumber string
}

// NewWord creates a Word from given letter other than N and a raw number string.
func NewWord(rawLetter rune, rawNumber string) (*Word, error) {
	number, err := strconv.ParseFloat(rawNumber, 64)
	if err != nil {
		return nil, err
	}
	letter := unicode.ToUpper(rawLetter)
	return &Word{letter: letter, rawLetter: rawLetter, number: number, rawNumber: rawNumber}, nil
}

func (w *Word) Letter() rune {
	return w.letter
}

func (w *Word) Number() float64 {
	return w.number
}

func (w *Word) String() string {
	if len(w.rawNumber) > 0 {
		return string(w.letter) + w.rawNumber
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
