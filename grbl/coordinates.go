package grbl

import (
	"fmt"
	"strconv"
	"strings"
)

type Coordinates struct {
	X float64
	Y float64
	Z float64
	A *float64
}

// NewCoordinates creates a Coordinates string values for X, Y, Z and A (optional).
func NewCoordinatesFromStrValues(dataValues []string) (*Coordinates, error) {
	coordinates := &Coordinates{}

	if len(dataValues) < 3 || len(dataValues) > 4 {
		return nil, fmt.Errorf("machine position field malformed: %#v", dataValues)
	}

	var err error

	coordinates.X, err = strconv.ParseFloat(dataValues[0], 64)
	if err != nil {
		return nil, fmt.Errorf("machine position X invalid: %#v", dataValues[0])
	}
	coordinates.Y, err = strconv.ParseFloat(dataValues[1], 64)
	if err != nil {
		return nil, fmt.Errorf("machine position Y invalid: %#v", dataValues[1])
	}
	coordinates.Z, err = strconv.ParseFloat(dataValues[2], 64)
	if err != nil {
		return nil, fmt.Errorf("machine position Z invalid: %#v", dataValues[2])
	}
	if len(dataValues) > 3 {
		a, err := strconv.ParseFloat(dataValues[3], 64)
		if err != nil {
			return nil, fmt.Errorf("machine position a invalid: %#v", dataValues[3])
		}
		coordinates.A = &a
	}
	return coordinates, nil
}

// NewCoordinates creates a Coordinates struct from a string CSV: X,Y,Z,A (A is optional)
func NewCoordinatesFromCSV(s string) (*Coordinates, error) {
	return NewCoordinatesFromStrValues(strings.Split(s, ","))
}

func (c *Coordinates) GetAxis(axis string) *float64 {
	switch axis {
	case "X":
		return &c.X
	case "Y":
		return &c.Y
	case "Z":
		return &c.Z
	case "A":
		return c.A
	}
	return nil
}
