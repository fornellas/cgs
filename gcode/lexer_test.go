package gcode

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLexer(t *testing.T) {
	matches, err := filepath.Glob("testdata/*.nc")
	require.NoError(t, err)
	require.NotEmpty(t, matches, "no .nc files found in gcode/testdata")

	for _, path := range matches {
		t.Run(path, func(t *testing.T) {
			f, err := os.Open(path)
			require.NoError(t, err, "failed to open %s", path)
			defer func() { require.NoError(t, f.Close()) }()

			var buf bytes.Buffer
			lx := NewLexer(f)
			for {
				token, err := lx.Next()
				require.NoError(t, err)
				if token.Type == TokenTypeEOF {
					break
				}
				buf.WriteString(token.Value)
			}

			orig, err := os.ReadFile(path)
			require.NoError(t, err)
			require.Equal(t, string(orig), buf.String())
		})
	}
}
