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
		eof   bool
		block *Block
		err   error
	}
	testCases := []struct {
		lines       []string
		nextReturns []nextReturn
	}{
		{
			lines: []string{" G0 ; foo"},
			nextReturns: []nextReturn{
				{eof: true, block: NewBlockCommand(NewWord('G', 0)), err: nil},
			},
		},
		{
			lines: []string{" $$ ; foo"},
			nextReturns: []nextReturn{
				{eof: true, block: NewBlockSystem("$$"), err: nil},
			},
		},
		{
			lines: []string{
				" G1 ; foo",
				"; bar",
			},
			nextReturns: []nextReturn{
				{eof: false, block: NewBlockCommand(NewWord('G', 1)), err: nil},
				{eof: true, err: nil},
			},
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprintf("Test Case #%d", i), func(t *testing.T) {
			parser := NewParser(strings.NewReader(strings.Join(tc.lines, "\n")))
			for j, nextReturn := range tc.nextReturns {
				eof, block, tokens, err := parser.Next()
				for _, token := range tokens {
					t.Logf("> %s : %#v", token.Type, string(token.Value))
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
				require.Equal(t, nextReturn.err, err)
			}
		})
	}
}
