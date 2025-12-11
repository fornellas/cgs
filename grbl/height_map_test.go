package grbl

import (
	"context"
	"fmt"
	"slices"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestNewHeightMap(t *testing.T) {
	for _, tc := range []struct {
		x0, y0, x1, y1, maxDistance float64
		errorContains               string
	}{
		{0, 0, 2, 2, 1, ""},
		{0, 0, 0, 2, 1, "x values must be different"},
		{0, 0, 2, 0, 1, "y values must be different"},
		{0, 0, 2, 4, 2, "not enough x probe points"},
		{0, 0, 4, 2, 2, "not enough y probe points"},
	} {
		t.Run(fmt.Sprintf("x0=%.1f,%.1f x1=%.1f,%.1f maxDistance=%.1f", tc.x0, tc.y0, tc.x1, tc.y1, tc.maxDistance), func(t *testing.T) {
			hm, err := NewHeightMap(tc.x0, tc.y0, tc.x1, tc.y1, tc.maxDistance)
			if tc.errorContains != "" {
				require.ErrorContains(t, err, tc.errorContains)
			} else {
				require.NoError(t, err)
				require.IsType(t, hm, &HeightMap{})
			}
		})
	}
}

func TestHeightMap(t *testing.T) {
	min := 0.0
	max := 2.0
	maxDistance := 1.0
	probeX := []float64{}
	for i := min; i <= max; i += maxDistance {
		probeX = append(probeX, min+maxDistance*i)
	}
	probeY := []float64{}
	for i := min; i <= max; i += maxDistance {
		probeY = append(probeY, min+maxDistance*i)
	}
	expectedProbeCount := len(probeX) * len(probeY)
	x0, y0 := min, min
	x1, y1 := max, max
	zt := 1.0
	zb := -1.0

	probeFn := func(ctx context.Context, x, y float64) (float64, error) {
		require.GreaterOrEqual(t, x, x0)
		require.GreaterOrEqual(t, y, y0)
		require.LessOrEqual(t, x, x1)
		require.LessOrEqual(t, y, y1)
		zRange := (zt - zb)
		z := zb + (((x + y) / (x1 + y1)) * zRange)
		return z, nil
	}

	hm, err := NewHeightMap(x0, y0, x1, y1, maxDistance)
	require.NoError(t, err)

	err = hm.Probe(t.Context(), probeFn)
	require.NoError(t, err)

	for _, ySlice := range hm.z {
		for _, z := range ySlice {
			fmt.Printf(" %.2f", z)
		}
		fmt.Println()
	}

	var probeCount int
	step := maxDistance / 4
	for x := x0 - step; x <= x1+step; x += step / 2 {
		for y := y0 - step; y <= y1+step; y += step / 2 {
			z := hm.GetInterpolatedValue(x, y)
			if x < x0 || x > x1 || y < y0 || y > y1 {
				require.Nilf(t, z, "%.2f,%.2f: z should have been nil: %#v", x, y, z)
			} else {
				isProbePoint := slices.Contains(probeX, x) && slices.Contains(probeY, y)
				if isProbePoint {
					probeCount++
					pz, err := probeFn(t.Context(), x, y)
					require.NoError(t, err)
					require.InDeltaf(t, pz, *z, 0.0001, "Probe point: %.2f %.2f: expected %.2f, got %.2f", x, y, pz, *z)
				}
				require.InDeltaf(t, (zt+zb)/2.0, *z, (zt-zb)/2.0, "%.2f,%.2f: z should have been %.2f and %.2f: %.2f", x, y, zb, zt, *z)
			}
		}
	}
	require.Equal(t, expectedProbeCount, probeCount)
}
