package gcode

import "fmt"

// RotateXY rotates work coordinates at the XY plane. Machine coordinates are not affected.
type RotateXY struct {
	parser            *Parser
	cx                float64
	cy                float64
	radians           float64
	initialModalGroup *ModalGroup
}

// NewRotateXY creates a new RotateXY.
// cx and cy are the center coordinates for the rotation, radians is the angle (looking down at XY
// from Z positive to Z negative).
func NewRotateXY(parser *Parser, cx, cy, radians float64) *RotateXY {
	return &RotateXY{
		parser:            parser,
		cx:                cx,
		cy:                cy,
		radians:           radians,
		initialModalGroup: parser.ModalGroup.Copy(),
	}
}

// Next returns the next line of G-code with XY coordinates rotated by the specified angle and center point.
// It validates that units and distance modes haven't changed since RotateXY was created, as these changes
// are unsupported during rotation. System blocks are passed through unchanged. Returns nil when the end
// of input is reached.
func (r *RotateXY) Next() (*string, error) {
	eof, block, tokens, err := r.parser.Next()
	if err != nil {
		return nil, err
	}
	if eof {
		return nil, nil
	}

	if !r.parser.ModalGroup.Units.Equal(r.initialModalGroup.Units) {
		return nil, fmt.Errorf("line %d: %s: unit change unsupported", r.parser.Lexer.Line, block)
	}

	if !r.parser.ModalGroup.DistanceMode.Equal(r.initialModalGroup.DistanceMode) {
		return nil, fmt.Errorf("line %d: %s: distance mode incremental unsupported", r.parser.Lexer.Line, block)
	}

	if block.IsSystem() {
		line := tokens.String()
		return &line, nil
	}

	if err = block.RotateXY(r.cx, r.cy, r.radians); err != nil {
		return nil, fmt.Errorf("line %d: %s", r.parser.Lexer.Line, err)
	}
	line := block.String()
	return &line, nil
}
