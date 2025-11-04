package gcode

import (
	"bytes"
	"fmt"
	"strconv"
)

// A Grbl system command (ie: $*)
type System string

// Word may either give a command or provide an argument to a command.
type Word struct {
	// A letter other than N
	letter rune
	// A real value.
	number float64
	// The raw string value for the number.
	rawNumber string
}

func NewWord(letter rune, rawNumber string) (*Word, error) {
	number, err := strconv.ParseFloat(rawNumber, 64)
	if err != nil {
		return nil, err
	}
	return &Word{letter: letter, number: number, rawNumber: rawNumber}, nil
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
	return fmt.Sprintf("%c%.3f", w.letter, w.number)
}

// Block is a line which may include commands to do several different things.
type Block struct {
	System    *string
	Command   *Word
	Arguments []*Word
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

type Program []Block
