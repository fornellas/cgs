package gcode

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLexerTestData(t *testing.T) {
	matches, err := filepath.Glob("testdata/*.nc")
	require.NoError(t, err)
	require.NotEmpty(t, matches)

	for _, path := range matches {
		t.Run(path, func(t *testing.T) {
			f, err := os.Open(path)
			require.NoError(t, err)
			defer func() { require.NoError(t, f.Close()) }()

			var buf bytes.Buffer
			lx := NewLexer(f)
			for {
				token, err := lx.Next()
				require.NoError(t, err)
				if token.Type == TokenTypeEOF {
					break
				}
				n, err := buf.Write(token.Value)
				require.Equal(t, len(token.Value), n)
				require.NoError(t, err)
			}

			orig, err := os.ReadFile(path)
			require.NoError(t, err)
			require.Equal(t, string(orig), buf.String())
		})
	}
}

func TestLexerTestCases(t *testing.T) {
	for i, expectedTokens := range []Tokens{
		// System commands
		Tokens{
			&Token{Type: TokenTypeSystem, Value: []byte("$$")},
		},
		Tokens{
			&Token{Type: TokenTypeSpace, Value: []byte("  ")},
			&Token{Type: TokenTypeSystem, Value: []byte("$2=0")},
			&Token{Type: TokenTypeSpace, Value: []byte(" ")},
			&Token{Type: TokenTypeComment, Value: []byte("; foo")},
		},
		// Parenthesis comment
		Tokens{
			&Token{Type: TokenTypeComment, Value: []byte("(this is a comment)")},
		},
		// Parenthesis comment with special chars
		Tokens{
			&Token{Type: TokenTypeComment, Value: []byte("(G-code test: 123@#$)")},
		},
		// Simple G command
		Tokens{
			&Token{Type: TokenTypeWordLetter, Value: []byte("G")},
			&Token{Type: TokenTypeWordNumber, Value: []byte("0")},
		},
		// G command with decimal (G38.2)
		Tokens{
			&Token{Type: TokenTypeWordLetter, Value: []byte("G")},
			&Token{Type: TokenTypeWordNumber, Value: []byte("38.2")},
		},
		// Negative number
		Tokens{
			&Token{Type: TokenTypeWordLetter, Value: []byte("X")},
			&Token{Type: TokenTypeWordNumber, Value: []byte("-10.5")},
		},
		// Positive number with sign
		Tokens{
			&Token{Type: TokenTypeWordLetter, Value: []byte("Y")},
			&Token{Type: TokenTypeWordNumber, Value: []byte("+5.25")},
		},

		// Decimal with trailing dot
		Tokens{
			&Token{Type: TokenTypeWordLetter, Value: []byte("F")},
			&Token{Type: TokenTypeWordNumber, Value: []byte("100.")},
		},
		// Multiple words in sequence
		Tokens{
			&Token{Type: TokenTypeWordLetter, Value: []byte("G")},
			&Token{Type: TokenTypeWordNumber, Value: []byte("1")},
			&Token{Type: TokenTypeSpace, Value: []byte(" ")},
			&Token{Type: TokenTypeWordLetter, Value: []byte("X")},
			&Token{Type: TokenTypeWordNumber, Value: []byte("10")},
			&Token{Type: TokenTypeSpace, Value: []byte(" ")},
			&Token{Type: TokenTypeWordLetter, Value: []byte("Y")},
			&Token{Type: TokenTypeWordNumber, Value: []byte("20")},
		},
		// Arc parameters I, J, K
		Tokens{
			&Token{Type: TokenTypeWordLetter, Value: []byte("I")},
			&Token{Type: TokenTypeWordNumber, Value: []byte("-5.5")},
			&Token{Type: TokenTypeSpace, Value: []byte(" ")},
			&Token{Type: TokenTypeWordLetter, Value: []byte("J")},
			&Token{Type: TokenTypeWordNumber, Value: []byte("3.2")},
			&Token{Type: TokenTypeSpace, Value: []byte(" ")},
			&Token{Type: TokenTypeWordLetter, Value: []byte("K")},
			&Token{Type: TokenTypeWordNumber, Value: []byte("0")},
		},
		// M command
		Tokens{
			&Token{Type: TokenTypeWordLetter, Value: []byte("M")},
			&Token{Type: TokenTypeWordNumber, Value: []byte("3")},
		},
		// System command with number
		Tokens{
			&Token{Type: TokenTypeSystem, Value: []byte("$100=0")},
		},
		// Spindle speed S word
		Tokens{
			&Token{Type: TokenTypeWordLetter, Value: []byte("S")},
			&Token{Type: TokenTypeWordNumber, Value: []byte("1000")},
		},
		// Line number N word
		Tokens{
			&Token{Type: TokenTypeWordLetter, Value: []byte("N")},
			&Token{Type: TokenTypeWordNumber, Value: []byte("100")},
		},
		// Parameter words P, R, L
		Tokens{
			&Token{Type: TokenTypeWordLetter, Value: []byte("P")},
			&Token{Type: TokenTypeWordNumber, Value: []byte("1")},
			&Token{Type: TokenTypeSpace, Value: []byte(" ")},
			&Token{Type: TokenTypeWordLetter, Value: []byte("R")},
			&Token{Type: TokenTypeWordNumber, Value: []byte("5.0")},
			&Token{Type: TokenTypeSpace, Value: []byte(" ")},
			&Token{Type: TokenTypeWordLetter, Value: []byte("L")},
			&Token{Type: TokenTypeWordNumber, Value: []byte("20")},
		},
		// Tool T word
		Tokens{
			&Token{Type: TokenTypeWordLetter, Value: []byte("T")},
			&Token{Type: TokenTypeWordNumber, Value: []byte("1")},
		},
		// Newline LF only
		Tokens{
			&Token{Type: TokenTypeWordLetter, Value: []byte("G")},
			&Token{Type: TokenTypeWordNumber, Value: []byte("0")},
			&Token{Type: TokenTypeNewLine, Value: []byte("\n")},
		},
		// Newline CRLF
		Tokens{
			&Token{Type: TokenTypeWordLetter, Value: []byte("G")},
			&Token{Type: TokenTypeWordNumber, Value: []byte("1")},
			&Token{Type: TokenTypeNewLine, Value: []byte("\r\n")},
		},
		// Complex line: G-code with comment
		Tokens{
			&Token{Type: TokenTypeWordLetter, Value: []byte("G")},
			&Token{Type: TokenTypeWordNumber, Value: []byte("28")},
			&Token{Type: TokenTypeComment, Value: []byte("; go home")},
			&Token{Type: TokenTypeNewLine, Value: []byte("\n")},
		},
		// Multiple spaces
		Tokens{
			&Token{Type: TokenTypeSpace, Value: []byte("    ")},
			&Token{Type: TokenTypeWordLetter, Value: []byte("G")},
			&Token{Type: TokenTypeWordNumber, Value: []byte("0")},
		},
		// Tab character
		Tokens{
			&Token{Type: TokenTypeSpace, Value: []byte("\t")},
			&Token{Type: TokenTypeWordLetter, Value: []byte("X")},
			&Token{Type: TokenTypeWordNumber, Value: []byte("10")},
		},
		// Mixed spaces and tabs
		Tokens{
			&Token{Type: TokenTypeSpace, Value: []byte(" \t ")},
			&Token{Type: TokenTypeWordLetter, Value: []byte("Y")},
			&Token{Type: TokenTypeWordNumber, Value: []byte("5.0")},
		},
		// Semicolon comment to end of line
		Tokens{
			&Token{Type: TokenTypeWordLetter, Value: []byte("G")},
			&Token{Type: TokenTypeWordNumber, Value: []byte("92")},
			&Token{Type: TokenTypeSpace, Value: []byte(" ")},
			&Token{Type: TokenTypeComment, Value: []byte("; set position")},
		},
		// Probe commands
		Tokens{
			&Token{Type: TokenTypeWordLetter, Value: []byte("G")},
			&Token{Type: TokenTypeWordNumber, Value: []byte("38.2")},
			&Token{Type: TokenTypeSpace, Value: []byte(" ")},
			&Token{Type: TokenTypeWordLetter, Value: []byte("Z")},
			&Token{Type: TokenTypeWordNumber, Value: []byte("-10")},
		},
		// Zero values
		Tokens{
			&Token{Type: TokenTypeWordLetter, Value: []byte("Z")},
			&Token{Type: TokenTypeWordNumber, Value: []byte("0")},
			&Token{Type: TokenTypeSpace, Value: []byte(" ")},
			&Token{Type: TokenTypeWordLetter, Value: []byte("F")},
			&Token{Type: TokenTypeWordNumber, Value: []byte("0.0")},
		},
		// All axes
		Tokens{
			&Token{Type: TokenTypeWordLetter, Value: []byte("X")},
			&Token{Type: TokenTypeWordNumber, Value: []byte("1.0")},
			&Token{Type: TokenTypeSpace, Value: []byte(" ")},
			&Token{Type: TokenTypeWordLetter, Value: []byte("Y")},
			&Token{Type: TokenTypeWordNumber, Value: []byte("2.0")},
			&Token{Type: TokenTypeSpace, Value: []byte(" ")},
			&Token{Type: TokenTypeWordLetter, Value: []byte("Z")},
			&Token{Type: TokenTypeWordNumber, Value: []byte("3.0")},
		},
		// Coordinate systems G54-G59
		Tokens{
			&Token{Type: TokenTypeWordLetter, Value: []byte("G")},
			&Token{Type: TokenTypeWordNumber, Value: []byte("54")},
			&Token{Type: TokenTypeSpace, Value: []byte(" ")},
			&Token{Type: TokenTypeWordLetter, Value: []byte("G")},
			&Token{Type: TokenTypeWordNumber, Value: []byte("55")},
		},
		// Plane selection
		Tokens{
			&Token{Type: TokenTypeWordLetter, Value: []byte("G")},
			&Token{Type: TokenTypeWordNumber, Value: []byte("17")},
			&Token{Type: TokenTypeSpace, Value: []byte(" ")},
			&Token{Type: TokenTypeWordLetter, Value: []byte("G")},
			&Token{Type: TokenTypeWordNumber, Value: []byte("90")},
		},
		// G2/G3 arc
		Tokens{
			&Token{Type: TokenTypeWordLetter, Value: []byte("G")},
			&Token{Type: TokenTypeWordNumber, Value: []byte("2")},
			&Token{Type: TokenTypeSpace, Value: []byte(" ")},
			&Token{Type: TokenTypeWordLetter, Value: []byte("X")},
			&Token{Type: TokenTypeWordNumber, Value: []byte("10")},
			&Token{Type: TokenTypeSpace, Value: []byte(" ")},
			&Token{Type: TokenTypeWordLetter, Value: []byte("Y")},
			&Token{Type: TokenTypeWordNumber, Value: []byte("10")},
			&Token{Type: TokenTypeSpace, Value: []byte(" ")},
			&Token{Type: TokenTypeWordLetter, Value: []byte("R")},
			&Token{Type: TokenTypeWordNumber, Value: []byte("5")},
		},
		// Dwell G4
		Tokens{
			&Token{Type: TokenTypeWordLetter, Value: []byte("G")},
			&Token{Type: TokenTypeWordNumber, Value: []byte("4")},
			&Token{Type: TokenTypeSpace, Value: []byte(" ")},
			&Token{Type: TokenTypeWordLetter, Value: []byte("P")},
			&Token{Type: TokenTypeWordNumber, Value: []byte("1.5")},
		},
		// Tool length offset G43.1
		Tokens{
			&Token{Type: TokenTypeWordLetter, Value: []byte("G")},
			&Token{Type: TokenTypeWordNumber, Value: []byte("43.1")},
			&Token{Type: TokenTypeSpace, Value: []byte(" ")},
			&Token{Type: TokenTypeWordLetter, Value: []byte("Z")},
			&Token{Type: TokenTypeWordNumber, Value: []byte("0.5")},
		},
		// Feed rate modes
		Tokens{
			&Token{Type: TokenTypeWordLetter, Value: []byte("G")},
			&Token{Type: TokenTypeWordNumber, Value: []byte("93")},
			&Token{Type: TokenTypeSpace, Value: []byte(" ")},
			&Token{Type: TokenTypeWordLetter, Value: []byte("F")},
			&Token{Type: TokenTypeWordNumber, Value: []byte("0.5")},
		},
		// Unit modes
		Tokens{
			&Token{Type: TokenTypeWordLetter, Value: []byte("G")},
			&Token{Type: TokenTypeWordNumber, Value: []byte("20")},
		},
		// Distance modes
		Tokens{
			&Token{Type: TokenTypeWordLetter, Value: []byte("G")},
			&Token{Type: TokenTypeWordNumber, Value: []byte("91")},
		},
		// Absolute override
		Tokens{
			&Token{Type: TokenTypeWordLetter, Value: []byte("G")},
			&Token{Type: TokenTypeWordNumber, Value: []byte("53")},
		},
		// Motion cancel
		Tokens{
			&Token{Type: TokenTypeWordLetter, Value: []byte("G")},
			&Token{Type: TokenTypeWordNumber, Value: []byte("80")},
		},
		// M0, M1, M2, M30 program flow
		Tokens{
			&Token{Type: TokenTypeWordLetter, Value: []byte("M")},
			&Token{Type: TokenTypeWordNumber, Value: []byte("0")},
		},
		// M5 spindle stop
		Tokens{
			&Token{Type: TokenTypeWordLetter, Value: []byte("M")},
			&Token{Type: TokenTypeWordNumber, Value: []byte("5")},
		},
		// M8 coolant on
		Tokens{
			&Token{Type: TokenTypeWordLetter, Value: []byte("M")},
			&Token{Type: TokenTypeWordNumber, Value: []byte("8")},
		},
		// M9 coolant off
		Tokens{
			&Token{Type: TokenTypeWordLetter, Value: []byte("M")},
			&Token{Type: TokenTypeWordNumber, Value: []byte("9")},
		},
		// Large negative number
		Tokens{
			&Token{Type: TokenTypeWordLetter, Value: []byte("Y")},
			&Token{Type: TokenTypeWordNumber, Value: []byte("-999.9999")},
		},
		// G10 set coordinates
		Tokens{
			&Token{Type: TokenTypeWordLetter, Value: []byte("G")},
			&Token{Type: TokenTypeWordNumber, Value: []byte("10")},
			&Token{Type: TokenTypeSpace, Value: []byte(" ")},
			&Token{Type: TokenTypeWordLetter, Value: []byte("P")},
			&Token{Type: TokenTypeWordNumber, Value: []byte("1")},
			&Token{Type: TokenTypeSpace, Value: []byte(" ")},
			&Token{Type: TokenTypeWordLetter, Value: []byte("L")},
			&Token{Type: TokenTypeWordNumber, Value: []byte("20")},
		},
		// G30 go home
		Tokens{
			&Token{Type: TokenTypeWordLetter, Value: []byte("G")},
			&Token{Type: TokenTypeWordNumber, Value: []byte("30")},
		},
		// G30.1 set home
		Tokens{
			&Token{Type: TokenTypeWordLetter, Value: []byte("G")},
			&Token{Type: TokenTypeWordNumber, Value: []byte("30.1")},
		},
		// G92.1 reset offset
		Tokens{
			&Token{Type: TokenTypeWordLetter, Value: []byte("G")},
			&Token{Type: TokenTypeWordNumber, Value: []byte("92.1")},
		},
		// G40 cutter comp cancel
		Tokens{
			&Token{Type: TokenTypeWordLetter, Value: []byte("G")},
			&Token{Type: TokenTypeWordNumber, Value: []byte("40")},
		},
		// G49 cancel tool length offset
		Tokens{
			&Token{Type: TokenTypeWordLetter, Value: []byte("G")},
			&Token{Type: TokenTypeWordNumber, Value: []byte("49")},
		},
		// G61 exact path
		Tokens{
			&Token{Type: TokenTypeWordLetter, Value: []byte("G")},
			&Token{Type: TokenTypeWordNumber, Value: []byte("61")},
		},
		// M1 optional stop
		Tokens{
			&Token{Type: TokenTypeWordLetter, Value: []byte("M")},
			&Token{Type: TokenTypeWordNumber, Value: []byte("1")},
		},
		// M2 program end
		Tokens{
			&Token{Type: TokenTypeWordLetter, Value: []byte("M")},
			&Token{Type: TokenTypeWordNumber, Value: []byte("2")},
		},
		// M30 program end and reset
		Tokens{
			&Token{Type: TokenTypeWordLetter, Value: []byte("M")},
			&Token{Type: TokenTypeWordNumber, Value: []byte("30")},
		},
		// M4 spindle CCW (if supported)
		Tokens{
			&Token{Type: TokenTypeWordLetter, Value: []byte("M")},
			&Token{Type: TokenTypeWordNumber, Value: []byte("4")},
		},
		// M7 coolant mist (if supported)
		Tokens{
			&Token{Type: TokenTypeWordLetter, Value: []byte("M")},
			&Token{Type: TokenTypeWordNumber, Value: []byte("7")},
		},
		// Complex g-code with all elements
		Tokens{
			&Token{Type: TokenTypeComment, Value: []byte("(start)")},
			&Token{Type: TokenTypeSpace, Value: []byte(" ")},
			&Token{Type: TokenTypeWordLetter, Value: []byte("G")},
			&Token{Type: TokenTypeWordNumber, Value: []byte("90")},
			&Token{Type: TokenTypeSpace, Value: []byte(" ")},
			&Token{Type: TokenTypeWordLetter, Value: []byte("G")},
			&Token{Type: TokenTypeWordNumber, Value: []byte("1")},
			&Token{Type: TokenTypeSpace, Value: []byte(" ")},
			&Token{Type: TokenTypeComment, Value: []byte("; linear)")},
		},
	} {
		gcode := expectedTokens.String()
		t.Run(fmt.Sprintf("#%d %#v", i, gcode), func(t *testing.T) {
			lexer := NewLexer(strings.NewReader(gcode))
			var tokens Tokens
			for {
				token, err := lexer.Next()
				require.NoError(t, err)
				if token.Type == TokenTypeEOF {
					break
				}
				tokens = append(tokens, token)
			}
			require.Equal(t, len(expectedTokens), len(tokens))
			for i := range tokens {
				require.Equal(t, expectedTokens[i], tokens[i])
			}
		})
	}
}
