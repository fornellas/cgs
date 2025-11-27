package grbl

import (
	"context"
	"fmt"
)

type HeightMap struct {
	x0, y0      float64
	x1, y1      float64
	maxDistance float64

	xSteps int
	ySteps int
	z      [][]float64
}

// Creates a new HeightMap. x0, y0 , x1, y1 specify the rectangular area to be probed. maxDistance
// specify the horizontal/vertical distance limit between probes.
func NewHeightMap(
	x0, y0, x1, y1, maxDistance float64,
) (*HeightMap, error) {
	h := &HeightMap{}

	if x1 < x0 {
		x0, x1 = x1, x0
	}
	if y1 < y0 {
		y0, y1 = y1, y0
	}
	h.x0 = x0
	h.x1 = x1
	if h.x0 == h.x1 {
		return nil, fmt.Errorf("x values must be different")
	}
	h.y0 = y0
	h.y1 = y1
	if h.y0 == h.y1 {
		return nil, fmt.Errorf("y values must be different")
	}

	h.maxDistance = maxDistance

	h.xSteps = int((x1 - x0) / maxDistance)
	if h.xSteps+1 < 3 {
		return nil, fmt.Errorf("not enough probe points")
	}
	h.ySteps = int((y1 - y0) / maxDistance)
	if h.ySteps+1 < 3 {
		return nil, fmt.Errorf("not enough probe points")
	}

	h.z = make([][]float64, h.xSteps+1)
	for i := range h.z {
		h.z[i] = make([]float64, h.ySteps+1)
	}

	return h, nil
}

// Probe the height map, using the currently active coordinate system.
// probeFn probes at given point, and return the measured z value.
func (h *HeightMap) Probe(
	ctx context.Context,
	probeFn func(ctx context.Context, x, y float64) (float64, error),
) error {
	xStep := (h.x1 - h.x0) / float64(h.xSteps)
	yStep := (h.y1 - h.y0) / float64(h.ySteps)
	for i := range h.xSteps + 1 {
		x := h.x0 + float64(i)*xStep
		if i%2 == 0 {
			for j := range h.ySteps + 1 {
				y := h.y0 + float64(j)*yStep
				z, err := probeFn(ctx, x, y)
				if err != nil {
					return err
				}
				h.z[i][j] = z
			}
		} else {
			for j := h.ySteps; j >= 0; j-- {
				y := h.y0 + float64(j)*yStep
				z, err := probeFn(ctx, x, y)
				if err != nil {
					return err
				}
				h.z[i][j] = z
			}
		}
	}
	return nil
}

// Return a corrected value for given x, y.
func (h *HeightMap) GetCorrectedValue(x, y float64) *float64 {
	if x < h.x0 || x > h.x1 || y < h.y0 || y > h.y1 {
		return nil
	}

	// Find the grid cell that contains (x, y)
	i := (x - h.x0) / (h.x1 - h.x0) * float64(h.xSteps)
	j := (y - h.y0) / (h.y1 - h.y0) * float64(h.ySteps)

	// Get the integer indices of the lower-left corner
	i0 := int(i)
	j0 := int(j)

	// Clamp to valid range
	if i0 > h.xSteps-1 {
		i0 = h.xSteps - 1
	}
	if j0 > h.ySteps-1 {
		j0 = h.ySteps - 1
	}

	i1 := i0 + 1
	j1 := j0 + 1

	// Get the 4 corner points
	z00 := h.z[i0][j0]
	z10 := h.z[i1][j0]
	z01 := h.z[i0][j1]
	z11 := h.z[i1][j1]

	// Bilinear interpolation
	fx := i - float64(i0)
	fy := j - float64(j0)

	z0 := z00*(1-fx) + z10*fx
	z1 := z01*(1-fx) + z11*fx
	z := z0*(1-fy) + z1*fy

	return &z
}
