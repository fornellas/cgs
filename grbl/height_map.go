package grbl

import "context"

type HeightMap struct {
	x0, y0      float64
	x1, y1      float64
	maxDistance float64

	xProbeCount int
	yProbeCount int
	z           []float64
}

// Creates a new HeightMap. x0, y0 , x1, y1 specify the rectangular area to be probed. maxDistance
// specify the horizontal/vertical distance limit between probes.
func NewHeightMap(
	x0, y0, x1, y1, maxDistance float64,
) *HeightMap {
	h := &HeightMap{}

	if x1 < x0 {
		x0, x1 = x1, x0
	}
	if y1 < y0 {
		y0, y1 = y1, y0
	}
	h.x0 = x0
	h.y0 = y0
	h.x1 = x1
	h.y1 = y1

	h.maxDistance = maxDistance

	h.xProbeCount = int((x1-x0)/maxDistance) + 1
	h.yProbeCount = int((y1-y0)/maxDistance) + 1

	h.z = make([]float64, (h.xProbeCount+1)*(h.yProbeCount+1))

	return h
}

// Probe the height map, using the currently active coordinate system.
func (h *HeightMap) Probe(
	ctx context.Context,
	probeFn func(ctx context.Context, x, y float64) (float64, error),
) error {
	xStep := (h.x1 - h.x0) / float64(h.xProbeCount)
	yStep := (h.y1 - h.y0) / float64(h.yProbeCount)
	for i := range h.xProbeCount {
		x := h.x0 + float64(i)*xStep
		if i%2 == 0 {
			for j := range h.yProbeCount {
				y := h.y0 + float64(j)*yStep
				z, err := probeFn(ctx, x, y)
				if err != nil {
					return err
				}
				h.z[j*h.xProbeCount+i] = z
			}
		} else {
			for j := h.yProbeCount - 1; j >= 0; j-- {
				y := h.y0 + float64(j)*yStep
				z, err := probeFn(ctx, x, y)
				if err != nil {
					return err
				}
				h.z[j*h.xProbeCount+i] = z
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
	i := (x - h.x0) / (h.x1 - h.x0) * float64(h.xProbeCount)
	j := (y - h.y0) / (h.y1 - h.y0) * float64(h.yProbeCount)

	// Get the integer indices of the lower-left corner
	i0 := int(i)
	j0 := int(j)

	i1 := i0 + 1
	j1 := j0 + 1

	// Get the 4 corner points
	z00 := h.z[j0*h.xProbeCount+i0]
	z10 := h.z[j0*h.xProbeCount+i1]
	z01 := h.z[j1*h.xProbeCount+i0]
	z11 := h.z[j1*h.xProbeCount+i1]

	// Bilinear interpolation
	fx := i - float64(i0)
	fy := j - float64(j0)

	z0 := z00*(1-fx) + z10*fx
	z1 := z01*(1-fx) + z11*fx
	z := z0*(1-fy) + z1*fy

	return &z
}
