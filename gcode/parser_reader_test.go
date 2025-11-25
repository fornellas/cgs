package gcode

import (
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParserReader(t *testing.T) {
	parserReader := NewParserReader(
		NewParser(strings.NewReader("G1 X10\n G0X20 ; foo")),
	)
	buf := make([]byte, 4)
	var bytes []byte
	for {
		n, err := parserReader.Read(buf)
		if n > 0 {
			bytes = append(bytes, buf[:n]...)
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			require.NoError(t, err)
		}
	}
	require.Equal(t, "G1X10\nG0X20\n", string(bytes))
}
