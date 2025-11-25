package gcode

import (
	"io"
	"sync"
)

// ParserReader wraps a Parser and implements the [io.Reader] interface.
type ParserReader struct {
	parser *Parser
	mu     sync.Mutex
	buffer []byte
	eof    bool
}

func NewParserReader(parser *Parser) *ParserReader {
	return &ParserReader{
		parser: parser,
	}
}

// Read calls Parser.Next and yields reads for each Block returned. Block.NormalizedString is used.
func (pr *ParserReader) Read(p []byte) (int, error) {
	pr.mu.Lock()
	defer pr.mu.Unlock()

	if len(pr.buffer) > 0 {
		n := copy(p, pr.buffer)
		pr.buffer = pr.buffer[n:]
		if len(pr.buffer) == 0 && pr.eof {
			return n, io.EOF
		}
		return n, nil
	}

	if pr.eof {
		return 0, io.EOF
	}

	for {
		eof, block, _, err := pr.parser.Next()
		if err != nil {
			return 0, err
		}

		if eof {
			pr.eof = true
		}

		if block == nil {
			if pr.eof {
				return 0, io.EOF
			}
			continue
		}

		data := []byte(block.String() + "\n")
		n := copy(p, data)

		if n < len(data) {
			pr.buffer = data[n:]
		}

		return n, nil
	}
}
