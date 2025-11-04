package gcode

import (
	"bytes"
	"strconv"
)

// A Grbl system command (ie: $*)
type System string

// Word may either give a command or provide an argument to a command.
type Word struct {
	// A letter other than N
	Letter rune
	// The raw string value for the number.
	Number string
}

// ParseNumber parses the raw string Number and returns its float64 value.
func (w *Word) ParseNumber() (float64, error) {
	return strconv.ParseFloat(w.Number, 64)
}

func (w *Word) String() string {
	return string(w.Letter) + w.Number
}

// Block is a line which may include commands to do several different things.
type Block struct {
	System    *string
	Command   *Word
	Arguments []Word
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
