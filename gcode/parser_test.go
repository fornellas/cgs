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

//gocyclo:ignore
func filterGcodeLines(t *testing.T, path string) []string {
	f, err := os.Open(path)
	require.NoError(t, err)
	defer func() { require.NoError(t, f.Close()) }()

	inComment := false
	var lineBuf []byte
	buf := make([]byte, 4096)
	var filteredLines []string
	for {
		n, err := f.Read(buf)
		if n == 0 && err != nil {
			break
		}
		for i := range n {
			c := buf[i]
			if c == '(' {
				inComment = true
			}
			if c == ')' {
				inComment = false
				continue
			}
			if c == '\n' {
				if !inComment {
					trimmed := string(bytes.TrimSpace(lineBuf))
					if trimmed != "" && !bytes.HasPrefix(bytes.TrimSpace(lineBuf), []byte(";")) {
						// Remove comments after ';'
						if idx := bytes.IndexByte(lineBuf, ';'); idx != -1 {
							trimmed = string(bytes.TrimSpace(lineBuf[:idx]))
						}
						// Remove parentheses comments
						withoutParens := []byte{}
						inParens := false
						for _, c := range []byte(trimmed) {
							if c == '(' {
								inParens = true
								continue
							}
							if c == ')' {
								inParens = false
								continue
							}
							if !inParens {
								withoutParens = append(withoutParens, c)
							}
						}
						trimmed = string(bytes.ReplaceAll(bytes.TrimSpace(withoutParens), []byte(" "), nil))
						if trimmed != "" {
							filteredLines = append(filteredLines, trimmed)
						}
					}
				}
				lineBuf = lineBuf[:0]
				continue
			}
			if !inComment && c != '\r' {
				lineBuf = append(lineBuf, c)
			}
		}
		if err != nil {
			break
		}
	}
	if len(lineBuf) > 0 && !inComment {
		trimmed := string(bytes.TrimSpace(lineBuf))
		if trimmed != "" && !bytes.HasPrefix(bytes.TrimSpace(lineBuf), []byte(";")) {
			// Remove comments after ';'
			if idx := bytes.IndexByte(lineBuf, ';'); idx != -1 {
				trimmed = string(bytes.TrimSpace(lineBuf[:idx]))
			}
			// Remove parentheses comments
			withoutParens := []byte{}
			inParens := false
			for _, c := range []byte(trimmed) {
				if c == '(' {
					inParens = true
					continue
				}
				if c == ')' {
					inParens = false
					continue
				}
				if !inParens {
					withoutParens = append(withoutParens, c)
				}
			}
			trimmed = string(bytes.ReplaceAll(bytes.TrimSpace(withoutParens), []byte(" "), nil))
			if trimmed != "" {
				filteredLines = append(filteredLines, trimmed)
			}
		}
	}
	return filteredLines
}

func TestParserWithTestData(t *testing.T) {
	matches, err := filepath.Glob("testdata/*.nc")
	require.NoError(t, err)
	require.NotEmpty(t, matches)

	for _, path := range matches {
		t.Run(path, func(t *testing.T) {
			var parsedLines []string
			f, err := os.Open(path)
			require.NoError(t, err)
			defer func() { require.NoError(t, f.Close()) }()

			parser := NewParser(f)
			for {
				eof, block, _, err := parser.Next()
				require.NoError(t, err)
				if eof {
					break
				}
				if block != nil {
					parsedLines = append(parsedLines, block.String())
				}
			}

			filteredLines := filterGcodeLines(t, path)
			require.Equal(t, filteredLines, parsedLines)
		})
	}
}

func TestParserTestCases(t *testing.T) {
	type nextReturn struct {
		eof           bool
		block         *Block
		errorContains string
	}
	testCases := []struct {
		lines       []string
		nextReturns []nextReturn
	}{
		// Basic motion commands
		{
			lines: []string{" G0 ; foo"},
			nextReturns: []nextReturn{
				{eof: true, block: NewBlockCommand(NewWord('G', 0))},
			},
		},
		{
			lines: []string{" G1 ; foo"},
			nextReturns: []nextReturn{
				{eof: true, block: NewBlockCommand(NewWord('G', 1))},
			},
		},
		{
			lines: []string{" G2 ; foo"},
			nextReturns: []nextReturn{
				{eof: true, block: NewBlockCommand(NewWord('G', 2))},
			},
		},
		{
			lines: []string{" G3 ; foo"},
			nextReturns: []nextReturn{
				{eof: true, block: NewBlockCommand(NewWord('G', 3))},
			},
		},
		// System commands
		{
			lines: []string{" $$ ; foo"},
			nextReturns: []nextReturn{
				{eof: true, block: NewBlockSystem("$$")},
			},
		},
		// Multiline with comments
		{
			lines: []string{
				" G1 ; foo",
				"; bar",
			},
			nextReturns: []nextReturn{
				{eof: false, block: NewBlockCommand(NewWord('G', 1))},
				{eof: true},
			},
		},
		// G0 with coordinates
		{
			lines: []string{"G0 X10 Y20 Z5"},
			nextReturns: []nextReturn{
				{eof: true, block: NewBlockCommand(
					NewWord('G', 0),
					NewWord('X', 10),
					NewWord('Y', 20),
					NewWord('Z', 5),
				)},
			},
		},
		// G1 with feed rate
		{
			lines: []string{"G1 X10 F100"},
			nextReturns: []nextReturn{
				{eof: true, block: NewBlockCommand(
					NewWord('G', 1),
					NewWord('X', 10),
					NewWord('F', 100),
				)},
			},
		},
		// G2 arc with offsets (I, J, K)
		{
			lines: []string{"G2 X10 Y10 I5 J5"},
			nextReturns: []nextReturn{
				{eof: true, block: NewBlockCommand(
					NewWord('G', 2),
					NewWord('X', 10),
					NewWord('Y', 10),
					NewWord('I', 5),
					NewWord('J', 5),
				)},
			},
		},
		// G3 arc with radius
		{
			lines: []string{"G3 X10 Y10 R5"},
			nextReturns: []nextReturn{
				{eof: true, block: NewBlockCommand(
					NewWord('G', 3),
					NewWord('X', 10),
					NewWord('Y', 10),
					NewWord('R', 5),
				)},
			},
		},
		// G4 dwell with P parameter
		{
			lines: []string{"G4 P1000"},
			nextReturns: []nextReturn{
				{eof: true, block: NewBlockCommand(
					NewWord('G', 4),
					NewWord('P', 1000),
				)},
			},
		},
		// G10 set coordinate data
		{
			lines: []string{"G10 L20 P1 X0 Y0 Z0"},
			nextReturns: []nextReturn{
				{eof: true, block: NewBlockCommand(
					NewWord('G', 10),
					NewWord('L', 20),
					NewWord('P', 1),
					NewWord('X', 0),
					NewWord('Y', 0),
					NewWord('Z', 0),
				)},
			},
		},
		// G17 plane selection
		{
			lines: []string{"G17"},
			nextReturns: []nextReturn{
				{eof: true, block: NewBlockCommand(NewWord('G', 17))},
			},
		},
		// G18 plane selection
		{
			lines: []string{"G18"},
			nextReturns: []nextReturn{
				{eof: true, block: NewBlockCommand(NewWord('G', 18))},
			},
		},
		// G19 plane selection
		{
			lines: []string{"G19"},
			nextReturns: []nextReturn{
				{eof: true, block: NewBlockCommand(NewWord('G', 19))},
			},
		},
		// G20 units inches
		{
			lines: []string{"G20"},
			nextReturns: []nextReturn{
				{eof: true, block: NewBlockCommand(NewWord('G', 20))},
			},
		},
		// G21 units millimeters
		{
			lines: []string{"G21"},
			nextReturns: []nextReturn{
				{eof: true, block: NewBlockCommand(NewWord('G', 21))},
			},
		},
		// G28 go home
		{
			lines: []string{"G28"},
			nextReturns: []nextReturn{
				{eof: true, block: NewBlockCommand(NewWord('G', 28))},
			},
		},
		// G28.1 set home
		{
			lines: []string{"G28.1"},
			nextReturns: []nextReturn{
				{eof: true, block: NewBlockCommand(NewWord('G', 28.1))},
			},
		},
		// G30 go to position 1
		{
			lines: []string{"G30"},
			nextReturns: []nextReturn{
				{eof: true, block: NewBlockCommand(NewWord('G', 30))},
			},
		},
		// G30.1 set position 1
		{
			lines: []string{"G30.1"},
			nextReturns: []nextReturn{
				{eof: true, block: NewBlockCommand(NewWord('G', 30.1))},
			},
		},
		// G38.2 probe toward
		{
			lines: []string{"G38.2 X10 Y10 Z-5 F100"},
			nextReturns: []nextReturn{
				{eof: true, block: NewBlockCommand(
					NewWord('G', 38.2),
					NewWord('X', 10),
					NewWord('Y', 10),
					NewWord('Z', -5),
					NewWord('F', 100),
				)},
			},
		},
		// G38.3 probe toward no error
		{
			lines: []string{"G38.3 Z-10 F50"},
			nextReturns: []nextReturn{
				{eof: true, block: NewBlockCommand(
					NewWord('G', 38.3),
					NewWord('Z', -10),
					NewWord('F', 50),
				)},
			},
		},
		// G38.4 probe away
		{
			lines: []string{"G38.4 Z5 F50"},
			nextReturns: []nextReturn{
				{eof: true, block: NewBlockCommand(
					NewWord('G', 38.4),
					NewWord('Z', 5),
					NewWord('F', 50),
				)},
			},
		},
		// G38.5 probe away no error
		{
			lines: []string{"G38.5 X0 F100"},
			nextReturns: []nextReturn{
				{eof: true, block: NewBlockCommand(
					NewWord('G', 38.5),
					NewWord('X', 0),
					NewWord('F', 100),
				)},
			},
		},
		// G40 cutter compensation off
		{
			lines: []string{"G40"},
			nextReturns: []nextReturn{
				{eof: true, block: NewBlockCommand(NewWord('G', 40))},
			},
		},
		// G43.1 dynamic tool length offset
		{
			lines: []string{"G43.1 Z10"},
			nextReturns: []nextReturn{
				{eof: true, block: NewBlockCommand(
					NewWord('G', 43.1),
					NewWord('Z', 10),
				)},
			},
		},
		// G49 cancel tool length compensation
		{
			lines: []string{"G49"},
			nextReturns: []nextReturn{
				{eof: true, block: NewBlockCommand(NewWord('G', 49))},
			},
		},
		// G53 absolute override with G0
		{
			lines: []string{"G53 G0 X0 Y0 Z0"},
			nextReturns: []nextReturn{
				{eof: true, block: NewBlockCommand(
					NewWord('G', 53),
					NewWord('G', 0),
					NewWord('X', 0),
					NewWord('Y', 0),
					NewWord('Z', 0),
				)},
			},
		},
		// G54 coordinate system 1
		{
			lines: []string{"G54"},
			nextReturns: []nextReturn{
				{eof: true, block: NewBlockCommand(NewWord('G', 54))},
			},
		},
		// G55 coordinate system 2
		{
			lines: []string{"G55"},
			nextReturns: []nextReturn{
				{eof: true, block: NewBlockCommand(NewWord('G', 55))},
			},
		},
		// G56 coordinate system 3
		{
			lines: []string{"G56"},
			nextReturns: []nextReturn{
				{eof: true, block: NewBlockCommand(NewWord('G', 56))},
			},
		},
		// G57 coordinate system 4
		{
			lines: []string{"G57"},
			nextReturns: []nextReturn{
				{eof: true, block: NewBlockCommand(NewWord('G', 57))},
			},
		},
		// G58 coordinate system 5
		{
			lines: []string{"G58"},
			nextReturns: []nextReturn{
				{eof: true, block: NewBlockCommand(NewWord('G', 58))},
			},
		},
		// G59 coordinate system 6
		{
			lines: []string{"G59"},
			nextReturns: []nextReturn{
				{eof: true, block: NewBlockCommand(NewWord('G', 59))},
			},
		},
		// G80 cancel modal motion
		{
			lines: []string{"G80"},
			nextReturns: []nextReturn{
				{eof: true, block: NewBlockCommand(NewWord('G', 80))},
			},
		},
		// G90 absolute distance mode
		{
			lines: []string{"G90"},
			nextReturns: []nextReturn{
				{eof: true, block: NewBlockCommand(NewWord('G', 90))},
			},
		},
		// G91 incremental distance mode
		{
			lines: []string{"G91"},
			nextReturns: []nextReturn{
				{eof: true, block: NewBlockCommand(NewWord('G', 91))},
			},
		},
		// G92 set coordinate offset
		{
			lines: []string{"G92 X0 Y0 Z0"},
			nextReturns: []nextReturn{
				{eof: true, block: NewBlockCommand(
					NewWord('G', 92),
					NewWord('X', 0),
					NewWord('Y', 0),
					NewWord('Z', 0),
				)},
			},
		},
		// G92.1 reset coordinate offset
		{
			lines: []string{"G92.1"},
			nextReturns: []nextReturn{
				{eof: true, block: NewBlockCommand(NewWord('G', 92.1))},
			},
		},
		// G93 inverse time feed rate
		{
			lines: []string{"G93"},
			nextReturns: []nextReturn{
				{eof: true, block: NewBlockCommand(NewWord('G', 93))},
			},
		},
		// G94 units per minute feed rate
		{
			lines: []string{"G94"},
			nextReturns: []nextReturn{
				{eof: true, block: NewBlockCommand(NewWord('G', 94))},
			},
		},
		// M0 program stop
		{
			lines: []string{"M0"},
			nextReturns: []nextReturn{
				{eof: true, block: NewBlockCommand(NewWord('M', 0))},
			},
		},
		// M1 optional stop
		{
			lines: []string{"M1"},
			nextReturns: []nextReturn{
				{eof: true, block: NewBlockCommand(NewWord('M', 1))},
			},
		},
		// M2 program end
		{
			lines: []string{"M2"},
			nextReturns: []nextReturn{
				{eof: true, block: NewBlockCommand(NewWord('M', 2))},
			},
		},
		// M3 spindle on clockwise
		{
			lines: []string{"M3 S1000"},
			nextReturns: []nextReturn{
				{eof: true, block: NewBlockCommand(
					NewWord('M', 3),
					NewWord('S', 1000),
				)},
			},
		},
		// M4 spindle on counterclockwise
		{
			lines: []string{"M4 S1500"},
			nextReturns: []nextReturn{
				{eof: true, block: NewBlockCommand(
					NewWord('M', 4),
					NewWord('S', 1500),
				)},
			},
		},
		// M5 spindle stop
		{
			lines: []string{"M5"},
			nextReturns: []nextReturn{
				{eof: true, block: NewBlockCommand(NewWord('M', 5))},
			},
		},
		// M7 mist coolant on
		{
			lines: []string{"M7"},
			nextReturns: []nextReturn{
				{eof: true, block: NewBlockCommand(NewWord('M', 7))},
			},
		},
		// M8 flood coolant on
		{
			lines: []string{"M8"},
			nextReturns: []nextReturn{
				{eof: true, block: NewBlockCommand(NewWord('M', 8))},
			},
		},
		// M9 coolant off
		{
			lines: []string{"M9"},
			nextReturns: []nextReturn{
				{eof: true, block: NewBlockCommand(NewWord('M', 9))},
			},
		},
		// M30 program end and reset
		{
			lines: []string{"M30"},
			nextReturns: []nextReturn{
				{eof: true, block: NewBlockCommand(NewWord('M', 30))},
			},
		},
		// Complex command with multiple words
		{
			lines: []string{"G1 X100 Y50 Z10 F500 S2000"},
			nextReturns: []nextReturn{
				{eof: true, block: NewBlockCommand(
					NewWord('G', 1),
					NewWord('X', 100),
					NewWord('Y', 50),
					NewWord('Z', 10),
					NewWord('F', 500),
					NewWord('S', 2000),
				)},
			},
		},
		// Line number N
		{
			lines: []string{"N100 G1 X10"},
			nextReturns: []nextReturn{
				{eof: true, block: NewBlockCommand(
					NewWord('N', 100),
					NewWord('G', 1),
					NewWord('X', 10),
				)},
			},
		},
		// Tool selection T
		{
			lines: []string{"T1"},
			nextReturns: []nextReturn{
				{eof: true, block: NewBlockCommand(NewWord('T', 1))},
			},
		},
		// Negative coordinates
		{
			lines: []string{"G0 X-50 Y-25 Z-10"},
			nextReturns: []nextReturn{
				{eof: true, block: NewBlockCommand(
					NewWord('G', 0),
					NewWord('X', -50),
					NewWord('Y', -25),
					NewWord('Z', -10),
				)},
			},
		},
		// Decimal coordinates
		{
			lines: []string{"G1 X10.5 Y20.75 Z5.25"},
			nextReturns: []nextReturn{
				{eof: true, block: NewBlockCommand(
					NewWord('G', 1),
					NewWord('X', 10.5),
					NewWord('Y', 20.75),
					NewWord('Z', 5.25),
				)},
			},
		},
		// Arc with K offset
		{
			lines: []string{"G2 X10 Y10 K5"},
			nextReturns: []nextReturn{
				{eof: true, block: NewBlockCommand(
					NewWord('G', 2),
					NewWord('X', 10),
					NewWord('Y', 10),
					NewWord('K', 5),
				)},
			},
		},
		// Multiple spaces
		{
			lines: []string{"  G1   X10   Y20  "},
			nextReturns: []nextReturn{
				{eof: true, block: NewBlockCommand(
					NewWord('G', 1),
					NewWord('X', 10),
					NewWord('Y', 20),
				)},
			},
		},
		// Parentheses comment
		{
			lines: []string{"G1 X10 (move to X10) Y20"},
			nextReturns: []nextReturn{
				{eof: true, block: NewBlockCommand(
					NewWord('G', 1),
					NewWord('X', 10),
					NewWord('Y', 20),
				)},
			},
		},
		// Empty lines
		{
			lines: []string{
				"G1 X10",
				"",
				"G1 Y20",
			},
			nextReturns: []nextReturn{
				{eof: false, block: NewBlockCommand(
					NewWord('G', 1),
					NewWord('X', 10),
				)},
				{eof: false},
				{eof: true, block: NewBlockCommand(
					NewWord('G', 1),
					NewWord('Y', 20),
				)},
			},
		},
		// Semicolon comment on same line
		{
			lines: []string{"G1 X10 ; this is a comment"},
			nextReturns: []nextReturn{
				{eof: true, block: NewBlockCommand(
					NewWord('G', 1),
					NewWord('X', 10),
				)},
			},
		},
		// Only comment line
		{
			lines: []string{"; this is just a comment"},
			nextReturns: []nextReturn{
				{eof: true},
			},
		},
		// Letter without number error
		{
			lines: []string{"G1 X"},
			nextReturns: []nextReturn{
				{errorContains: "unexpected word letter at end"},
			},
		},
		// Number without letter error
		{
			lines: []string{"G1 10"},
			nextReturns: []nextReturn{
				{errorContains: "unexpected word number"},
			},
		},
		// Invalid number format
		{
			lines: []string{"G1 X1.2.3"},
			nextReturns: []nextReturn{
				{errorContains: "invalid number"},
			},
		},
		// Mixed system & command
		{
			lines: []string{"G0 $$"},
			nextReturns: []nextReturn{
				{errorContains: "system command cannot follow command words"},
			},
		},
		// Junk
		{
			lines: []string{"G0 #"},
			nextReturns: []nextReturn{
				{errorContains: "unexpected char"},
			},
		},
		// System command without newline
		{
			lines: []string{"$H"},
			nextReturns: []nextReturn{
				{eof: true, block: NewBlockSystem("$H")},
			},
		},
		// System command with spaces, no newline
		{
			lines: []string{" $H "},
			nextReturns: []nextReturn{
				{eof: true, block: NewBlockSystem("$H")},
			},
		},
		// System command with comment, no newline
		{
			lines: []string{" $H ; homing"},
			nextReturns: []nextReturn{
				{eof: true, block: NewBlockSystem("$H")},
			},
		},
		// System command $C without newline
		{
			lines: []string{"$C"},
			nextReturns: []nextReturn{
				{eof: true, block: NewBlockSystem("$C")},
			},
		},
		// System command $SLP without newline
		{
			lines: []string{"$SLP"},
			nextReturns: []nextReturn{
				{eof: true, block: NewBlockSystem("$SLP")},
			},
		},
	}

	for i, tc := range testCases {
		gcode := strings.Join(tc.lines, "\n")
		t.Run(fmt.Sprintf("#%d %s", i, gcode), func(t *testing.T) {
			parser := NewParser(strings.NewReader(gcode))
			for j, nextReturn := range tc.nextReturns {
				eof, block, tokens, err := parser.Next()
				if nextReturn.errorContains != "" {
					require.ErrorContains(t, err, nextReturn.errorContains)
				} else {
					for _, token := range tokens {
						t.Logf("%d> %s : %#v", j, token.Type, string(token.Value))
					}
					require.Equal(t, nextReturn.eof, eof)
					if nextReturn.block != nil {
						require.NotNil(t, block)
						require.Equal(t, nextReturn.block.NormalizedString(), block.NormalizedString())
					} else {
						require.Nil(t, block)
					}
					nl := ""
					if j < len(tc.lines)-1 {
						nl = "\n"
					}
					require.Equal(t, tc.lines[j]+nl, tokens.String())
				}
			}
		})
	}
}

func TestParserBlocks(t *testing.T) {
	testCases := []struct {
		name     string
		gcode    string
		expected []string // normalized strings of expected blocks
	}{
		{
			name:     "system command without newline",
			gcode:    "$H",
			expected: []string{"$H"},
		},
		{
			name:     "system command with spaces no newline",
			gcode:    " $H ",
			expected: []string{"$H"},
		},
		{
			name:     "system command with comment no newline",
			gcode:    " $H ; homing",
			expected: []string{"$H"},
		},
		{
			name:     "multiple commands with last without newline",
			gcode:    "G0 X10\n$H",
			expected: []string{"G0X10", "$H"},
		},
		{
			name:     "system command with newline",
			gcode:    "$H\n",
			expected: []string{"$H"},
		},
		{
			name:     "G-code command without newline",
			gcode:    "G0 X10",
			expected: []string{"G0X10"},
		},
		{
			name:     "multiple G-code commands",
			gcode:    "G0 X10\nG1 Y20\nM3",
			expected: []string{"G0X10", "G1Y20", "M3"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			parser := NewParser(strings.NewReader(tc.gcode))
			blocks, err := parser.Blocks()
			require.NoError(t, err)
			require.Equal(t, len(tc.expected), len(blocks), "block count mismatch")
			for i, expectedNorm := range tc.expected {
				require.Equal(t, expectedNorm, blocks[i].NormalizedString())
			}
		})
	}
}
