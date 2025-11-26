package grbl

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/fornellas/slogxt/log"

	"github.com/fornellas/cgs/gcode"
)

var ErrEEPROMCommandNotSupported = errors.New("EEPROM related commands can not be streamed")

type ProgramStreamer struct {
	port                   io.Writer
	responseMessageCh      chan *ResponseMessage
	maxSerialRxBufferBytes int

	availableBufferBytes int
	sentChunkBytes       []int
}

func NewProgramStreamer(
	port io.Writer,
	responseMessageCh chan *ResponseMessage,
	maxSerialRxBufferBytes int,
) *ProgramStreamer {
	return &ProgramStreamer{
		port:                   port,
		responseMessageCh:      responseMessageCh,
		maxSerialRxBufferBytes: maxSerialRxBufferBytes,
	}
}

func (s *ProgramStreamer) writeChunk(chunk []byte) error {
	n, err := s.port.Write(chunk)
	if err != nil {
		return fmt.Errorf("write to serial port error: %w", err)
	}
	if n != len(chunk) {
		panic(fmt.Errorf("bug: write to serial port error: wrote %d bytes, expected %d", n, len(chunk)))
	}
	return nil
}

func (s *ProgramStreamer) waitForResponseMessage(ctx context.Context) error {
	var ok bool
	select {
	case _, ok = <-s.responseMessageCh:
		if !ok {
			return fmt.Errorf("stream program: response message channel is closed")
		}
	case <-ctx.Done():
		return fmt.Errorf("stream program: %w", ctx.Err())
	}
	s.availableBufferBytes += s.sentChunkBytes[0]
	s.sentChunkBytes = s.sentChunkBytes[1:]
	return nil
}

func (s *ProgramStreamer) writeLine(ctx context.Context, line []byte) error {
	sent := 0
	for sent < len(line) {
		for s.availableBufferBytes == 0 {
			if err := s.waitForResponseMessage(ctx); err != nil {
				return err
			}
		}

		end := min(sent+s.availableBufferBytes, len(line))
		chunk := line[sent:end]

		err := s.writeChunk(chunk)
		if err != nil {
			return err
		}

		sent += len(chunk)
		s.availableBufferBytes -= len(chunk)
	}

	s.sentChunkBytes = append(s.sentChunkBytes, len(line))
	return nil
}

func (s *ProgramStreamer) Run(ctx context.Context, programReader io.Reader) error {
	ctx, logger := log.MustWithGroup(ctx, "Program Streamer")

	s.availableBufferBytes = s.maxSerialRxBufferBytes
	s.sentChunkBytes = []int{}

	parser := gcode.NewParser(programReader)

	for {
		eof, block, _, err := parser.Next()
		if err != nil {
			return fmt.Errorf("gcode parse error: %w", err)
		}

		if block == nil || block.Empty() {
			if eof {
				break
			}
			continue
		}

		if block.IsEEPROM() {
			return fmt.Errorf("%w: %s", ErrEEPROMCommandNotSupported, block.NormalizedString())
		}

		line := []byte(block.NormalizedString() + "\n")

		if err := s.writeLine(ctx, line); err != nil {
			return err
		}

		logger.Debug("Sent", "line", strings.TrimSuffix(string(line), "\n"))

		if eof {
			break
		}
	}

	for len(s.sentChunkBytes) > 0 {
		if err := s.waitForResponseMessage(ctx); err != nil {
			return err
		}
	}

	if s.availableBufferBytes != s.maxSerialRxBufferBytes {
		panic(fmt.Errorf("bug: final availableBufferBytes %d differs from maxSerialRxBufferBytes %d", s.availableBufferBytes, s.maxSerialRxBufferBytes))
	}
	if len(s.sentChunkBytes) > 0 {
		panic(fmt.Errorf("bug: sentChunkBytes not empty: %#v", s.sentChunkBytes))
	}

	return nil
}
