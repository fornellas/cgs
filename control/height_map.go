package control

import (
	"context"
	"errors"
	"fmt"

	"github.com/rivo/tview"
)

type HeightMapPrimitive struct {
	*tview.Flex
	app              *tview.Application
	controlPrimitive *ControlPrimitive
}

func NewHeightMapPrimitive(
	ctx context.Context,
	app *tview.Application,
	controlPrimitive *ControlPrimitive,
) *HeightMapPrimitive {
	sp := &HeightMapPrimitive{
		app:              app,
		controlPrimitive: controlPrimitive,
	}

	// Parameters
	// x0
	// y0
	// x1
	// y1
	// maxDistance

	rootFlex := tview.NewFlex()
	rootFlex.SetBorder(true)
	rootFlex.SetTitle("Height Map")
	rootFlex.SetDirection(tview.FlexRow)
	sp.Flex = rootFlex

	// TODO set disabled state

	return sp
}

func (hm *HeightMapPrimitive) Worker(
	ctx context.Context, trackedStateCh <-chan *TrackedState,
) error {
	for {
		select {
		case <-ctx.Done():
			err := ctx.Err()
			if errors.Is(err, context.Canceled) {
				err = nil
			}
			return err
		case _, ok := <-trackedStateCh:
			if !ok {
				return fmt.Errorf("tracked state channel closed")
			}
			// TODO
		}
	}
}
