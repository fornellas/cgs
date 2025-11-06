package gcode

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWordNormalizedString(t *testing.T) {
	testCases := []struct {
		letter   rune
		number   float64
		expected string
	}{
		{'G', 1.0, "G1"},
		{'G', 1.1, "G1.1"},
		{'G', 1.2, "G1.2"},
		{'X', 1.2345, "X1.2345"},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("%c%f", tc.letter, tc.number), func(t *testing.T) {
			word := NewWord(tc.letter, tc.number)
			require.Equal(t, tc.expected, word.NormalizedString())
		})
	}
}
