package tui

import (
	"context"
	"fmt"
	"strconv"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	iFmt "github.com/fornellas/cgs/internal/fmt"

	grblMod "github.com/fornellas/cgs/grbl"
)

type HeightMapPrimitive struct {
	*tview.Flex
	ctx              context.Context
	app              *tview.Application
	grbl             *grblMod.Grbl
	controlPrimitive *ControlPrimitive

	x0InputField            *tview.InputField
	y0InputField            *tview.InputField
	x1InputField            *tview.InputField
	y1InputField            *tview.InputField
	maxDistanceInputField   *tview.InputField
	zProbePlaneInputField   *tview.InputField
	maxZDeviationInputField *tview.InputField
	probeFeedRateInputField *tview.InputField
	probeButton             *tview.Button
	statusTextView          *tview.TextView
	heightMapTable          *tview.Table
	heightMapTableCellMap   map[float64]map[float64]*tview.TableCell

	heightMap *grblMod.HeightMap
}

func NewHeightMapPrimitive(
	ctx context.Context,
	app *tview.Application,
	grbl *grblMod.Grbl,
	controlPrimitive *ControlPrimitive,
) *HeightMapPrimitive {
	hm := &HeightMapPrimitive{
		ctx:              ctx,
		app:              app,
		grbl:             grbl,
		controlPrimitive: controlPrimitive,
	}

	// Parameters
	hm.x0InputField = tview.NewInputField()
	hm.x0InputField.SetLabel("X0")
	hm.x0InputField.SetFieldWidth(coordinateWidth)
	hm.x0InputField.SetAcceptanceFunc(acceptFloat)
	hm.x0InputField.SetChangedFunc(func(text string) { hm.updateHeightMap() })

	hm.y0InputField = tview.NewInputField()
	hm.y0InputField.SetLabel("Y0")
	hm.y0InputField.SetFieldWidth(coordinateWidth)
	hm.y0InputField.SetAcceptanceFunc(acceptFloat)
	hm.y0InputField.SetChangedFunc(func(text string) { hm.updateHeightMap() })

	hm.x1InputField = tview.NewInputField()
	hm.x1InputField.SetLabel("X1")
	hm.x1InputField.SetFieldWidth(coordinateWidth)
	hm.x1InputField.SetAcceptanceFunc(acceptFloat)
	hm.x1InputField.SetChangedFunc(func(text string) { hm.updateHeightMap() })

	hm.y1InputField = tview.NewInputField()
	hm.y1InputField.SetLabel("Y1")
	hm.y1InputField.SetFieldWidth(coordinateWidth)
	hm.y1InputField.SetAcceptanceFunc(acceptFloat)
	hm.y1InputField.SetChangedFunc(func(text string) { hm.updateHeightMap() })

	hm.maxDistanceInputField = tview.NewInputField()
	hm.maxDistanceInputField.SetLabel("Max probe point distance")
	hm.maxDistanceInputField.SetFieldWidth(coordinateWidth)
	hm.maxDistanceInputField.SetAcceptanceFunc(acceptUFloat)
	hm.maxDistanceInputField.SetChangedFunc(func(text string) { hm.updateHeightMap() })

	hm.zProbePlaneInputField = tview.NewInputField()
	hm.zProbePlaneInputField.SetLabel("Z Probe Plane")
	hm.zProbePlaneInputField.SetFieldWidth(coordinateWidth)
	hm.zProbePlaneInputField.SetAcceptanceFunc(acceptFloat)

	hm.maxZDeviationInputField = tview.NewInputField()
	hm.maxZDeviationInputField.SetLabel("Max Z deviation")
	hm.maxZDeviationInputField.SetFieldWidth(coordinateWidth)
	hm.maxZDeviationInputField.SetAcceptanceFunc(acceptUFloat)

	hm.probeFeedRateInputField = tview.NewInputField()
	hm.probeFeedRateInputField.SetLabel("Probe Feed Rate")
	hm.probeFeedRateInputField.SetFieldWidth(coordinateWidth)
	hm.probeFeedRateInputField.SetAcceptanceFunc(acceptUFloat)

	hm.probeButton = tview.NewButton("Probe")
	hm.probeButton.SetSelectedFunc(func() {
		go hm.probe()
	})

	hm.statusTextView = tview.NewTextView()
	hm.statusTextView.SetDynamicColors(true)

	hm.heightMapTable = tview.NewTable()

	rootFlex := tview.NewFlex()
	rootFlex.SetBorder(true)
	rootFlex.SetTitle("Height Map")
	rootFlex.SetDirection(tview.FlexRow)
	rootFlex.AddItem(hm.x0InputField, 1, 0, false)
	rootFlex.AddItem(hm.y0InputField, 1, 0, false)
	rootFlex.AddItem(hm.x1InputField, 1, 0, false)
	rootFlex.AddItem(hm.y1InputField, 1, 0, false)
	rootFlex.AddItem(hm.maxDistanceInputField, 1, 0, false)
	rootFlex.AddItem(hm.zProbePlaneInputField, 1, 0, false)
	rootFlex.AddItem(hm.maxZDeviationInputField, 1, 0, false)
	rootFlex.AddItem(hm.probeFeedRateInputField, 1, 0, false)
	rootFlex.AddItem(hm.probeButton, 3, 0, false)
	rootFlex.AddItem(hm.statusTextView, 1, 0, false)
	rootFlex.AddItem(hm.heightMapTable, 0, 1, false)

	hm.Flex = rootFlex

	// TODO set disabled state

	return hm
}

func (hm *HeightMapPrimitive) updateHeightMap() {
	hm.heightMapTable.Clear()
	hm.heightMapTableCellMap = nil

	var err error
	var x0, y0, x1, y1, maxDistance float64

	if x0, err = strconv.ParseFloat(hm.x0InputField.GetText(), 64); err != nil {
		hm.statusTextView.SetText(fmt.Sprintf("[%s]Invalid X0: %s[-]", tcell.ColorRed, tview.Escape(err.Error())))
		return
	}
	if y0, err = strconv.ParseFloat(hm.y0InputField.GetText(), 64); err != nil {
		hm.statusTextView.SetText(fmt.Sprintf("[%s]Invalid Y0: %s[-]", tcell.ColorRed, tview.Escape(err.Error())))
		return
	}
	if x1, err = strconv.ParseFloat(hm.x1InputField.GetText(), 64); err != nil {
		hm.statusTextView.SetText(fmt.Sprintf("[%s]Invalid X1: %s[-]", tcell.ColorRed, tview.Escape(err.Error())))
		return
	}
	if y1, err = strconv.ParseFloat(hm.y1InputField.GetText(), 64); err != nil {
		hm.statusTextView.SetText(fmt.Sprintf("[%s]Invalid Y1: %s[-]", tcell.ColorRed, tview.Escape(err.Error())))
		return
	}
	if maxDistance, err = strconv.ParseFloat(hm.maxDistanceInputField.GetText(), 64); err != nil {
		hm.statusTextView.SetText(fmt.Sprintf("[%s]Invalid Max probe point distance: %s[-]", tcell.ColorRed, tview.Escape(err.Error())))
		return
	}

	hm.heightMap, err = grblMod.NewHeightMap(x0, y0, x1, y1, maxDistance)
	if err != nil {
		hm.statusTextView.SetText(fmt.Sprintf("[%s]%s[-]", tcell.ColorRed, tview.Escape(err.Error())))
		return
	}

	xSteps := hm.heightMap.GetXSteps()
	ySteps := hm.heightMap.GetYSteps()
	for i, x := range xSteps {
		tableCell := tview.NewTableCell(sprintColorCoordinate(x))
		tableCell.SetAlign(tview.AlignCenter)
		hm.heightMapTable.SetCell(len(ySteps), i+1, tableCell)
	}
	for j, y := range ySteps {
		tableCell := tview.NewTableCell(sprintColorCoordinate(y))
		tableCell.SetAlign(tview.AlignCenter)
		hm.heightMapTable.SetCell(len(ySteps)-j-1, 0, tableCell)
	}

	hm.heightMapTableCellMap = map[float64]map[float64]*tview.TableCell{}
	for i, x := range xSteps {
		for j, y := range ySteps {
			yMap, ok := hm.heightMapTableCellMap[x]
			if !ok {
				yMap = map[float64]*tview.TableCell{}
				hm.heightMapTableCellMap[x] = yMap
			}
			tableCell := tview.NewTableCell("N/A")
			tableCell.SetAlign(tview.AlignCenter)
			yMap[y] = tableCell
			hm.heightMapTable.SetCell(len(ySteps)-j-1, i+1, tableCell)
		}
	}

	hm.statusTextView.SetText("")
}

func (hm *HeightMapPrimitive) getTableCellColor(minValue, maxValue, value float64) tcell.Color {
	if maxValue <= minValue {
		return tcell.NewRGBColor(0, 255, 0)
	}

	t := (value - minValue) / (maxValue - minValue)
	t = max(0.0, min(1.0, t))

	var r, g, b int32
	const maxChannelValue = float64(255)

	switch {
	case t <= 0.25:
		// blue -> cyan
		tPrime := t / 0.25
		r = 0
		g = int32(tPrime * maxChannelValue)
		b = 255
	case t <= 0.5:
		// cyan -> green
		tPrime := (t - 0.25) / 0.25
		r = 0
		g = 255
		b = int32((1.0 - tPrime) * maxChannelValue)
	case t <= 0.75:
		// green -> yellow
		tPrime := (t - 0.5) / 0.25
		r = int32(tPrime * maxChannelValue)
		g = 255
		b = 0
	default:
		// yellow -> red
		tPrime := (t - 0.75) / 0.25
		r = 255
		g = int32((1.0 - tPrime) * maxChannelValue)
		b = 0
	}

	return tcell.NewRGBColor(r, g, b)
}

//gocyclo:ignore
func (hm *HeightMapPrimitive) updateTableCellColors() {
	var minZ, maxZ *float64
	for _, yMap := range hm.heightMapTableCellMap {
		for _, tableCell := range yMap {
			if tableCell.Reference == nil {
				continue
			}
			z := tableCell.Reference.(float64)
			if minZ == nil {
				if maxZ == nil {
					minZ = &z
					maxZ = &z
				} else {
					if z > *maxZ {
						minZ = maxZ
						maxZ = &z
					} else {
						minZ = &z
					}
				}
			} else {
				if maxZ == nil {
					if z < *minZ {
						maxZ = minZ
						minZ = &z
					} else {
						maxZ = &z
					}
				} else {
					if z < *minZ {
						minZ = &z
					}
					if z > *maxZ {
						maxZ = &z
					}
				}
			}
		}
	}

	if minZ == nil || maxZ == nil {
		return
	}
	for _, yMap := range hm.heightMapTableCellMap {
		for _, tableCell := range yMap {
			if tableCell.Reference == nil {
				continue
			}
			z := tableCell.Reference.(float64)
			tableCell.SetBackgroundColor(hm.getTableCellColor(*minZ, *maxZ, z))
		}
	}
}

func (hm *HeightMapPrimitive) getProbeFunc(zProbePlane, maxZDeviation, probeFeedRate float64) func(ctx context.Context, x, y float64) (float64, error) {
	return func(ctx context.Context, x, y float64) (probedZ float64, err error) {
		yMap, ok := hm.heightMapTableCellMap[x]
		if !ok {
			panic(fmt.Errorf("bug: can't find table cell: X%f", x))
		}
		tableCell, ok := yMap[y]
		if !ok {
			panic(fmt.Errorf("bug: can't find table cell: Y%f", y))
		}

		hm.app.QueueUpdateDraw(func() {
			tableCell.SetText("Probing")
		})

		if err := hm.controlPrimitive.SendCommand(ctx,
			fmt.Sprintf("G0 Z%s", iFmt.SprintFloat(zProbePlane+maxZDeviation, 4)),
		); err != nil {
			return 0, err
		}

		if err := hm.controlPrimitive.SendCommand(ctx,
			fmt.Sprintf("G0 X%s Y%s", iFmt.SprintFloat(x, 4), iFmt.SprintFloat(y, 4)),
		); err != nil {
			return 0, err
		}

		if err := hm.controlPrimitive.SendCommand(ctx,
			fmt.Sprintf("G38.2 Z%s F%s", iFmt.SprintFloat(zProbePlane-maxZDeviation, 4), iFmt.SprintFloat(probeFeedRate, 4)),
		); err != nil {
			return 0, err
		}

		if err := hm.controlPrimitive.SendCommand(ctx, grblMod.GrblCommandViewGcodeParameters); err != nil {
			return 0, err
		}

		gcodeParameters := hm.controlPrimitive.grbl.GetLastGcodeParameters()
		if gcodeParameters.Probe == nil {
			return 0, fmt.Errorf("no probe result at G-Code Parameters push message")
		}

		if !gcodeParameters.Probe.Successful {
			return 0, fmt.Errorf("probe failed")
		}

		workCoordinatesOffset := hm.grbl.GetLastWorkCoordinateOffset()
		if workCoordinatesOffset == nil {
			panic("bug: Grbl.GetLastWorkCoordinateOffset not expected to be nil")
		}

		probedZ = gcodeParameters.Probe.Coordinates.Z - workCoordinatesOffset.Z

		hm.app.QueueUpdateDraw(func() {
			tableCell.SetText(iFmt.SprintFloat(probedZ, 4))
			tableCell.Reference = probedZ
		})

		hm.updateTableCellColors()

		return probedZ, nil
	}
}

func (hm *HeightMapPrimitive) probe() {
	var err error
	var zProbePlane, maxZDeviation, probeFeedRate float64

	hm.app.QueueUpdateDraw(func() {
		if zProbePlane, err = strconv.ParseFloat(hm.zProbePlaneInputField.GetText(), 64); err != nil {
			hm.statusTextView.SetText(fmt.Sprintf("[%s]Invalid Z Probe Plane: %s[-]", tcell.ColorRed, tview.Escape(err.Error())))
			return
		}

		if maxZDeviation, err = strconv.ParseFloat(hm.maxZDeviationInputField.GetText(), 64); err != nil {
			hm.statusTextView.SetText(fmt.Sprintf("[%s]Invalid Max Z Deviation: %s[-]", tcell.ColorRed, tview.Escape(err.Error())))
			return
		}

		if probeFeedRate, err = strconv.ParseFloat(hm.probeFeedRateInputField.GetText(), 64); err != nil {
			hm.statusTextView.SetText(fmt.Sprintf("[%s]Invalid Probe Feed Rate: %s[-]", tcell.ColorRed, tview.Escape(err.Error())))
			return
		}

		for _, yMap := range hm.heightMapTableCellMap {
			for _, tableCell := range yMap {
				tableCell.SetText("N/A")
				tableCell.SetBackgroundColor(tcell.ColorNone)
				tableCell.Reference = nil
			}
		}

		hm.statusTextView.SetText("Probing...")
	})

	// This primes data for Grbl.GetLastWorkCoordinateOffset() which we'll need
	if err := hm.controlPrimitive.grbl.SendRealTimeCommand(grblMod.RealTimeCommandStatusReportQuery); err != nil {
		hm.app.QueueUpdateDraw(func() {
			hm.statusTextView.SetText(fmt.Sprintf("[%s]Failed to query Status Report: %s[-]", tcell.ColorRed, tview.Escape(err.Error())))
		})
	}

	if err := hm.heightMap.Probe(hm.ctx, hm.getProbeFunc(zProbePlane, maxZDeviation, probeFeedRate)); err != nil {
		hm.app.QueueUpdateDraw(func() {
			hm.statusTextView.SetText(fmt.Sprintf("[%s]%s[-]", tcell.ColorRed, tview.Escape(err.Error())))
		})
	}

	hm.app.QueueUpdateDraw(func() {
		hm.statusTextView.SetText(fmt.Sprintf("[%s]Success[-]", tcell.ColorGreen))
	})
}

func (hm *HeightMapPrimitive) Worker(
	ctx context.Context,
	trackedStateCh <-chan *TrackedState,
) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case _, ok := <-trackedStateCh:
			if !ok {
				return fmt.Errorf("tracked state channel closed")
			}
			// TODO
		}
	}
}
