package control

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	iFmt "github.com/fornellas/cgs/internal/fmt"

	"github.com/fornellas/cgs/gcode"
	grblMod "github.com/fornellas/cgs/grbl"
	"github.com/fornellas/cgs/worker_manager"
)

var defaultCommandTimeout = 1 * time.Second
var homeCommandTimeout = 2 * time.Minute
var probeCommandTimeout = 2 * time.Minute

var gcodeParserModalGroupsMotionWords = []string{
	"G0", "G1", "G2", "G3", "G38.2", "G38.3", "G38.4", "G38.5", "G80",
}
var gcodeParserModalGroupsPlaneSelectionWords = []string{
	"G17", "G18", "G19",
}
var gcodeParserModalGroupsDistanceModeWords = []string{
	"G90", "G91",
}
var gcodeParserModalGroupsFeedRateModeWords = []string{
	"G93", "G94",
}
var gcodeParserModalGroupsUnitsWords = []string{
	"G20", "G21",
}
var gcodeParserModalGroupsCoordinateSystemSelectWords = []string{
	"G54", "G55", "G56", "G57", "G58", "G59",
}
var gcodeParserModalGroupsSpindleWords = []string{
	"M3", "M4", "M5",
}
var gcodeParamsCoordinateSystemModeOptions = []string{
	fmt.Sprintf("Offset%s", sprintGcodeWord("G10L2")),
	fmt.Sprintf("Work Coordinates%s", sprintGcodeWord("G10L20")),
}
var gcodeParamsCoordinateSystemModeOffsetIdx = 0
var gcodeParamsCoordinateSystemModeWorkCoordinatesIdx = 1

var gcodeParamsCoordinateOffsetModeOptions = []string{
	"Offset",
	"Work Coordinates",
}

var gcodeParamsCoordinateOffsetModeOffsetIdx = 0
var gcodeParamsCoordinateOffsetModeWorkCoordinatesIdx = 1

var allStatusCommands = map[string]bool{
	grblMod.GrblCommandViewGcodeParameters:  true,
	grblMod.GrblCommandViewGrblSettings:     true,
	grblMod.GrblCommandViewStartupBlocks:    true,
	grblMod.GrblCommandViewBuildInfo:        true,
	grblMod.GrblCommandViewGcodeParserState: true,
}

type commandParameterType struct {
	command string
	timeout time.Duration
	quiet   bool
}

var syncCommand = &commandParameterType{
	command: "G4 P0.01",
	quiet:   true,
}

type queuedCommandType struct {
	command string
	errCh   chan<- error
}

type ControlPrimitive struct {
	*tview.Flex
	grbl         *grblMod.Grbl
	app          *tview.Application
	stateTracker *StateTracker

	quietStatusComms bool

	skipQueueCommand bool

	sendStatusCommandCh   chan string
	sendCommandCh         chan *queuedCommandType
	sendRealTimeCommandCh chan grblMod.RealTimeCommand

	gcodeParserModalGroupsMotionDropDown *tview.DropDown

	gcodeParserModalGroupsPlaneSelectionDropDown *tview.DropDown

	gcodeParserModalGroupsDistanceModeDropDown *tview.DropDown

	gcodeParserModalGroupsFeedRateModeDropDown *tview.DropDown

	gcodeParserModalGroupsUnitsDropDown *tview.DropDown

	gcodeParserModalGroupsToolLengthOffsetInputField *tview.InputField

	gcodeParserModalGroupsCoordinateSystemSelectDropDown *tview.DropDown

	gcodeParserModalGroupsStoppingCheckbox *tview.Checkbox

	gcodeParserModalGroupsSpindleDropDown *tview.DropDown

	gcodeParserModalGroupsCoolantMistCheckbok  *tview.Checkbox
	gcodeParserModalGroupsCoolantFloodCheckbok *tview.Checkbox
	gcodeParserModalGroupsCoolantOff           *tview.Button

	gcodeParserStateToolInputField         *tview.InputField
	gcodeParserStateSpindleSpeedInputField *tview.InputField
	gcodeParserStateFeedRateInputField     *tview.InputField

	gcodeParserFlex *tview.Flex

	gcodeParamsCoordinateSystemModeDropdown *tview.DropDown

	gcodeParamsCoordinateSystem1XInputField *tview.InputField
	gcodeParamsCoordinateSystem1YInputField *tview.InputField
	gcodeParamsCoordinateSystem1ZInputField *tview.InputField

	gcodeParamsCoordinateSystem2XInputField *tview.InputField
	gcodeParamsCoordinateSystem2YInputField *tview.InputField
	gcodeParamsCoordinateSystem2ZInputField *tview.InputField

	gcodeParamsCoordinateSystem3XInputField *tview.InputField
	gcodeParamsCoordinateSystem3YInputField *tview.InputField
	gcodeParamsCoordinateSystem3ZInputField *tview.InputField

	gcodeParamsCoordinateSystem4XInputField *tview.InputField
	gcodeParamsCoordinateSystem4YInputField *tview.InputField
	gcodeParamsCoordinateSystem4ZInputField *tview.InputField

	gcodeParamsCoordinateSystem5XInputField *tview.InputField
	gcodeParamsCoordinateSystem5YInputField *tview.InputField
	gcodeParamsCoordinateSystem5ZInputField *tview.InputField

	gcodeParamsCoordinateSystem6XInputField *tview.InputField
	gcodeParamsCoordinateSystem6YInputField *tview.InputField
	gcodeParamsCoordinateSystem6ZInputField *tview.InputField

	gcodeParamsPreDefinedPosition1XInputField        *tview.InputField
	gcodeParamsPreDefinedPosition1YInputField        *tview.InputField
	gcodeParamsPreDefinedPosition1ZInputField        *tview.InputField
	gcodeParamsPreDefinedPosition1GoToButton         *tview.Button
	gcodeParamsPreDefinedPosition1SetToCurrentButton *tview.Button

	gcodeParamsPreDefinedPosition2XInputField        *tview.InputField
	gcodeParamsPreDefinedPosition2YInputField        *tview.InputField
	gcodeParamsPreDefinedPosition2ZInputField        *tview.InputField
	gcodeParamsPreDefinedPosition2GoToButton         *tview.Button
	gcodeParamsPreDefinedPosition2SetToCurrentButton *tview.Button

	gcodeParamsCoordinateOffsetModeDropdown *tview.DropDown

	gcodeParamsCoordinateOffsetXInputField *tview.InputField
	gcodeParamsCoordinateOffsetYInputField *tview.InputField
	gcodeParamsCoordinateOffsetZInputField *tview.InputField

	gcodeParamsScrollContainer *ScrollContainer

	commandsTextView *tview.TextView

	pushMessagesTextView *tview.TextView

	commandInputField      *tview.InputField
	commandInputHistory    []string
	commandInputHistoryIdx int

	disableCommandInput bool

	state grblMod.State

	machineCoordinates *grblMod.Coordinates
	gcodeParameters    *grblMod.GcodeParameters
	modalGroup         *gcode.ModalGroup

	mu sync.Mutex
}

//gocyclo:ignore
func NewControlPrimitive(
	ctx context.Context,
	grbl *grblMod.Grbl,
	app *tview.Application,
	stateTracker *StateTracker,
	quietStatusComms bool,
) *ControlPrimitive {
	cp := &ControlPrimitive{
		grbl:                   grbl,
		app:                    app,
		stateTracker:           stateTracker,
		quietStatusComms:       quietStatusComms,
		sendStatusCommandCh:    make(chan string, 10),
		sendCommandCh:          make(chan *queuedCommandType, 10),
		sendRealTimeCommandCh:  make(chan grblMod.RealTimeCommand, 10),
		commandInputHistoryIdx: -1,
		state:                  grblMod.StateUnknown,
	}

	cp.newGcodeParser()
	cp.newGcodeParams()
	cp.newCommands()
	cp.newPushMessages()
	cp.newCommand()

	cp.newControl()

	cp.setDisabledState()
	cp.sendStatusCommands()

	return cp
}

func (cp *ControlPrimitive) newGcodeParser() {
	newModalGroupDropDown := func(name string, words []string) *tview.DropDown {
		dropDown := tview.NewDropDown()
		dropDown.SetLabel(name + ":")
		texts := []string{}
		for _, word := range words {
			texts = append(texts, fmt.Sprintf("%s%s", tview.Escape(gcode.WordName(word)), sprintGcodeWord(word)))
		}
		dropDown.SetOptions(texts, func(text string, index int) {
			if cp.skipQueueCommand {
				return
			}
			cp.QueueCommand(words[index])
		})
		return dropDown
	}

	// Motion
	cp.gcodeParserModalGroupsMotionDropDown = newModalGroupDropDown(
		"Motion", gcodeParserModalGroupsMotionWords,
	)

	// Plane selection
	cp.gcodeParserModalGroupsPlaneSelectionDropDown = newModalGroupDropDown(
		"Plane Selection", gcodeParserModalGroupsPlaneSelectionWords,
	)

	// Distance Mode
	cp.gcodeParserModalGroupsDistanceModeDropDown = newModalGroupDropDown(
		"Distance Mode", gcodeParserModalGroupsDistanceModeWords,
	)

	// Arc IJK Distance Mode
	gcodeParserModalGroupsArcIjkDistanceModeDropDown := newModalGroupDropDown(
		"Arc IJK Distance Mode", []string{"G91.1"},
	)
	cp.skipQueueCommand = true
	gcodeParserModalGroupsArcIjkDistanceModeDropDown.SetCurrentOption(0)
	cp.skipQueueCommand = false
	gcodeParserModalGroupsArcIjkDistanceModeDropDown.SetDisabled(true)

	// Feed Rate Mode
	cp.gcodeParserModalGroupsFeedRateModeDropDown = newModalGroupDropDown(
		"Feed Rate Mode", gcodeParserModalGroupsFeedRateModeWords,
	)

	// Units
	cp.gcodeParserModalGroupsUnitsDropDown = newModalGroupDropDown(
		"Units", gcodeParserModalGroupsUnitsWords,
	)

	// Cutter Diameter Compensation
	gcodeParserModalGroupsCutterDiameterCompensationDropDown := newModalGroupDropDown(
		"Cutter Diameter Compensation", []string{"G40"},
	)
	cp.skipQueueCommand = true
	gcodeParserModalGroupsCutterDiameterCompensationDropDown.SetCurrentOption(0)
	cp.skipQueueCommand = false
	gcodeParserModalGroupsCutterDiameterCompensationDropDown.SetDisabled(true)

	// Tool Length Offset
	cp.gcodeParserModalGroupsToolLengthOffsetInputField = tview.NewInputField()
	cp.gcodeParserModalGroupsToolLengthOffsetInputField.SetLabel(fmt.Sprintf("Tool Length Offset%s:", sprintGcodeWord("G43.1")))
	cp.gcodeParserModalGroupsToolLengthOffsetInputField.SetAcceptanceFunc(acceptFloat)
	cp.gcodeParserModalGroupsToolLengthOffsetInputField.SetFieldWidth(coordinateWidth)
	cp.gcodeParserModalGroupsToolLengthOffsetInputField.SetDoneFunc(func(key tcell.Key) {
		if cp.skipQueueCommand {
			return
		}
		cp.QueueCommand(fmt.Sprintf("G43.1 Z%s", cp.gcodeParserModalGroupsToolLengthOffsetInputField.GetText()))
	})

	// Coordinate System Select
	cp.gcodeParserModalGroupsCoordinateSystemSelectDropDown = newModalGroupDropDown(
		"Coordinate System Select", gcodeParserModalGroupsCoordinateSystemSelectWords,
	)

	// Control Mode
	gcodeParserModalGroupsControlModeDropDown := newModalGroupDropDown(
		"Control Mode", []string{"G61"},
	)
	cp.skipQueueCommand = true
	gcodeParserModalGroupsControlModeDropDown.SetCurrentOption(0)
	cp.skipQueueCommand = false
	gcodeParserModalGroupsControlModeDropDown.SetDisabled(true)

	// Stopping
	cp.gcodeParserModalGroupsStoppingCheckbox = tview.NewCheckbox()
	cp.gcodeParserModalGroupsStoppingCheckbox.SetLabel(fmt.Sprintf("%s%s", tview.Escape(gcode.WordName("M0")), sprintGcodeWord("M0")))
	cp.gcodeParserModalGroupsStoppingCheckbox.SetChangedFunc(func(checked bool) {
		if cp.skipQueueCommand {
			return
		}
		if checked {
			cp.QueueCommand("M0")
		} else {
			cp.QueueRealTimeCommand(grblMod.RealTimeCommandCycleStartResume)
		}
	})

	// Spindle
	cp.gcodeParserModalGroupsSpindleDropDown = newModalGroupDropDown(
		"Spindle", gcodeParserModalGroupsSpindleWords,
	)

	// Coolant
	coolantTextView := tview.NewTextView()
	coolantTextView.SetLabel("Coolant:")
	cp.gcodeParserModalGroupsCoolantMistCheckbok = tview.NewCheckbox()
	cp.gcodeParserModalGroupsCoolantMistCheckbok.SetLabel(fmt.Sprintf("Mist%s", sprintGcodeWord("M7")))
	cp.gcodeParserModalGroupsCoolantMistCheckbok.SetChangedFunc(func(checked bool) {
		if cp.skipQueueCommand {
			return
		}
		cp.QueueCommand("M7")
	})
	cp.gcodeParserModalGroupsCoolantFloodCheckbok = tview.NewCheckbox()
	cp.gcodeParserModalGroupsCoolantFloodCheckbok.SetLabel(fmt.Sprintf("Flood%s", sprintGcodeWord("M8")))
	cp.gcodeParserModalGroupsCoolantFloodCheckbok.SetChangedFunc(func(checked bool) {
		if cp.skipQueueCommand {
			return
		}
		cp.QueueCommand("M8")
	})
	cp.gcodeParserModalGroupsCoolantOff = tview.NewButton(fmt.Sprintf("Off%s", sprintGcodeWord("M9")))
	cp.gcodeParserModalGroupsCoolantOff.SetSelectedFunc(func() {
		cp.QueueCommand("M9")
	})
	coolantFlex := tview.NewFlex()
	coolantFlex.SetDirection(tview.FlexColumn)
	coolantFlex.AddItem(coolantTextView, 0, 1, false)
	coolantFlex.AddItem(cp.gcodeParserModalGroupsCoolantMistCheckbok, 0, 1, false)
	coolantFlex.AddItem(cp.gcodeParserModalGroupsCoolantFloodCheckbok, 0, 1, false)
	coolantFlex.AddItem(cp.gcodeParserModalGroupsCoolantOff, 0, 1, false)

	// Modal Groups Flex
	gcodeParserModalGroupsFlex := NewScrollContainer()
	gcodeParserModalGroupsFlex.SetBorder(true)
	gcodeParserModalGroupsFlex.SetTitle("G-Code: Modal Groups")
	gcodeParserModalGroupsFlex.AddPrimitive(cp.gcodeParserModalGroupsMotionDropDown, 1)
	gcodeParserModalGroupsFlex.AddPrimitive(cp.gcodeParserModalGroupsPlaneSelectionDropDown, 1)
	gcodeParserModalGroupsFlex.AddPrimitive(cp.gcodeParserModalGroupsDistanceModeDropDown, 1)
	gcodeParserModalGroupsFlex.AddPrimitive(gcodeParserModalGroupsArcIjkDistanceModeDropDown, 1)
	gcodeParserModalGroupsFlex.AddPrimitive(cp.gcodeParserModalGroupsFeedRateModeDropDown, 1)
	gcodeParserModalGroupsFlex.AddPrimitive(cp.gcodeParserModalGroupsUnitsDropDown, 1)
	gcodeParserModalGroupsFlex.AddPrimitive(gcodeParserModalGroupsCutterDiameterCompensationDropDown, 1)
	gcodeParserModalGroupsFlex.AddPrimitive(cp.gcodeParserModalGroupsToolLengthOffsetInputField, 1)
	gcodeParserModalGroupsFlex.AddPrimitive(cp.gcodeParserModalGroupsCoordinateSystemSelectDropDown, 1)
	gcodeParserModalGroupsFlex.AddPrimitive(gcodeParserModalGroupsControlModeDropDown, 1)
	gcodeParserModalGroupsFlex.AddPrimitive(cp.gcodeParserModalGroupsStoppingCheckbox, 1)
	gcodeParserModalGroupsFlex.AddPrimitive(cp.gcodeParserModalGroupsSpindleDropDown, 1)
	gcodeParserModalGroupsFlex.AddPrimitive(coolantFlex, 1)

	// Parser State Flex
	cp.gcodeParserStateToolInputField = tview.NewInputField()
	cp.gcodeParserStateToolInputField.SetLabel("Tool:")
	cp.gcodeParserStateToolInputField.SetAcceptanceFunc(acceptUFloat)
	cp.gcodeParserStateToolInputField.SetFieldWidth(2)
	cp.gcodeParserStateToolInputField.SetDoneFunc(func(key tcell.Key) {
		if cp.skipQueueCommand {
			return
		}
		cp.QueueCommand(fmt.Sprintf("T%s", cp.gcodeParserStateToolInputField.GetText()))
	})
	cp.gcodeParserStateSpindleSpeedInputField = tview.NewInputField()
	cp.gcodeParserStateSpindleSpeedInputField.SetLabel("Spindle Speed:")
	cp.gcodeParserStateSpindleSpeedInputField.SetAcceptanceFunc(acceptUFloat)
	cp.gcodeParserStateSpindleSpeedInputField.SetFieldWidth(spindleSpeedWidth)
	cp.gcodeParserStateSpindleSpeedInputField.SetDoneFunc(func(key tcell.Key) {
		if cp.skipQueueCommand {
			return
		}
		cp.QueueCommand(fmt.Sprintf("S%s", cp.gcodeParserStateSpindleSpeedInputField.GetText()))
	})
	cp.gcodeParserStateFeedRateInputField = tview.NewInputField()
	cp.gcodeParserStateFeedRateInputField.SetLabel("Feed Rate:")
	cp.gcodeParserStateFeedRateInputField.SetAcceptanceFunc(acceptUFloat)
	cp.gcodeParserStateFeedRateInputField.SetFieldWidth(feedWidth)
	cp.gcodeParserStateFeedRateInputField.SetDoneFunc(func(key tcell.Key) {
		if cp.skipQueueCommand {
			return
		}
		cp.QueueCommand(fmt.Sprintf("F%s", cp.gcodeParserStateFeedRateInputField.GetText()))
	})
	gcodeParserStateFlex := NewScrollContainer()
	gcodeParserStateFlex.SetBorder(true)
	gcodeParserStateFlex.SetTitle("G-Code: Parser State")
	gcodeParserStateFlex.AddPrimitive(cp.gcodeParserStateToolInputField, 1)
	gcodeParserStateFlex.AddPrimitive(cp.gcodeParserStateSpindleSpeedInputField, 1)
	gcodeParserStateFlex.AddPrimitive(cp.gcodeParserStateFeedRateInputField, 1)

	// Flex
	gcodeParserFlex := tview.NewFlex()
	gcodeParserFlex.SetBorder(false)
	gcodeParserFlex.SetDirection(tview.FlexRow)
	gcodeParserFlex.AddItem(gcodeParserModalGroupsFlex, 15, 0, false)
	gcodeParserFlex.AddItem(gcodeParserStateFlex, 5, 0, false)

	cp.gcodeParserFlex = gcodeParserFlex
}

//gocyclo:ignore
func (cp *ControlPrimitive) updateGcodeParamsCoordinateSystem() {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	if n, _ := cp.gcodeParamsCoordinateSystemModeDropdown.GetCurrentOption(); n == gcodeParamsCoordinateSystemModeOffsetIdx && cp.gcodeParameters != nil {
		// Offset
		if cp.gcodeParameters.CoordinateSystem1 != nil {
			cp.gcodeParamsCoordinateSystem1XInputField.SetText(iFmt.SprintFloat(cp.gcodeParameters.CoordinateSystem1.X, 4))
			cp.gcodeParamsCoordinateSystem1YInputField.SetText(iFmt.SprintFloat(cp.gcodeParameters.CoordinateSystem1.Y, 4))
			cp.gcodeParamsCoordinateSystem1ZInputField.SetText(iFmt.SprintFloat(cp.gcodeParameters.CoordinateSystem1.Z, 4))
		} else {
			cp.gcodeParamsCoordinateSystem1XInputField.SetText("")
			cp.gcodeParamsCoordinateSystem1YInputField.SetText("")
			cp.gcodeParamsCoordinateSystem1ZInputField.SetText("")
		}
		if cp.gcodeParameters.CoordinateSystem2 != nil {
			cp.gcodeParamsCoordinateSystem2XInputField.SetText(iFmt.SprintFloat(cp.gcodeParameters.CoordinateSystem2.X, 4))
			cp.gcodeParamsCoordinateSystem2YInputField.SetText(iFmt.SprintFloat(cp.gcodeParameters.CoordinateSystem2.Y, 4))
			cp.gcodeParamsCoordinateSystem2ZInputField.SetText(iFmt.SprintFloat(cp.gcodeParameters.CoordinateSystem2.Z, 4))
		} else {
			cp.gcodeParamsCoordinateSystem2XInputField.SetText("")
			cp.gcodeParamsCoordinateSystem2YInputField.SetText("")
			cp.gcodeParamsCoordinateSystem2ZInputField.SetText("")
		}
		if cp.gcodeParameters.CoordinateSystem3 != nil {
			cp.gcodeParamsCoordinateSystem3XInputField.SetText(iFmt.SprintFloat(cp.gcodeParameters.CoordinateSystem3.X, 4))
			cp.gcodeParamsCoordinateSystem3YInputField.SetText(iFmt.SprintFloat(cp.gcodeParameters.CoordinateSystem3.Y, 4))
			cp.gcodeParamsCoordinateSystem3ZInputField.SetText(iFmt.SprintFloat(cp.gcodeParameters.CoordinateSystem3.Z, 4))
		} else {
			cp.gcodeParamsCoordinateSystem3XInputField.SetText("")
			cp.gcodeParamsCoordinateSystem3YInputField.SetText("")
			cp.gcodeParamsCoordinateSystem3ZInputField.SetText("")
		}
		if cp.gcodeParameters.CoordinateSystem4 != nil {
			cp.gcodeParamsCoordinateSystem4XInputField.SetText(iFmt.SprintFloat(cp.gcodeParameters.CoordinateSystem4.X, 4))
			cp.gcodeParamsCoordinateSystem4YInputField.SetText(iFmt.SprintFloat(cp.gcodeParameters.CoordinateSystem4.Y, 4))
			cp.gcodeParamsCoordinateSystem4ZInputField.SetText(iFmt.SprintFloat(cp.gcodeParameters.CoordinateSystem4.Z, 4))
		} else {
			cp.gcodeParamsCoordinateSystem4XInputField.SetText("")
			cp.gcodeParamsCoordinateSystem4YInputField.SetText("")
			cp.gcodeParamsCoordinateSystem4ZInputField.SetText("")
		}
		if cp.gcodeParameters.CoordinateSystem5 != nil {
			cp.gcodeParamsCoordinateSystem5XInputField.SetText(iFmt.SprintFloat(cp.gcodeParameters.CoordinateSystem5.X, 4))
			cp.gcodeParamsCoordinateSystem5YInputField.SetText(iFmt.SprintFloat(cp.gcodeParameters.CoordinateSystem5.Y, 4))
			cp.gcodeParamsCoordinateSystem5ZInputField.SetText(iFmt.SprintFloat(cp.gcodeParameters.CoordinateSystem5.Z, 4))
		} else {
			cp.gcodeParamsCoordinateSystem5XInputField.SetText("")
			cp.gcodeParamsCoordinateSystem5YInputField.SetText("")
			cp.gcodeParamsCoordinateSystem5ZInputField.SetText("")
		}
		if cp.gcodeParameters.CoordinateSystem6 != nil {
			cp.gcodeParamsCoordinateSystem6XInputField.SetText(iFmt.SprintFloat(cp.gcodeParameters.CoordinateSystem6.X, 4))
			cp.gcodeParamsCoordinateSystem6YInputField.SetText(iFmt.SprintFloat(cp.gcodeParameters.CoordinateSystem6.Y, 4))
			cp.gcodeParamsCoordinateSystem6ZInputField.SetText(iFmt.SprintFloat(cp.gcodeParameters.CoordinateSystem6.Z, 4))
		} else {
			cp.gcodeParamsCoordinateSystem6XInputField.SetText("")
			cp.gcodeParamsCoordinateSystem6YInputField.SetText("")
			cp.gcodeParamsCoordinateSystem6ZInputField.SetText("")
		}
	} else {
		// Machine Coordinates
		cp.gcodeParamsCoordinateSystem1XInputField.SetText("")
		cp.gcodeParamsCoordinateSystem1YInputField.SetText("")
		cp.gcodeParamsCoordinateSystem1ZInputField.SetText("")
		cp.gcodeParamsCoordinateSystem2XInputField.SetText("")
		cp.gcodeParamsCoordinateSystem2YInputField.SetText("")
		cp.gcodeParamsCoordinateSystem2ZInputField.SetText("")
		cp.gcodeParamsCoordinateSystem3XInputField.SetText("")
		cp.gcodeParamsCoordinateSystem3YInputField.SetText("")
		cp.gcodeParamsCoordinateSystem3ZInputField.SetText("")
		cp.gcodeParamsCoordinateSystem4XInputField.SetText("")
		cp.gcodeParamsCoordinateSystem4YInputField.SetText("")
		cp.gcodeParamsCoordinateSystem4ZInputField.SetText("")
		cp.gcodeParamsCoordinateSystem5XInputField.SetText("")
		cp.gcodeParamsCoordinateSystem5YInputField.SetText("")
		cp.gcodeParamsCoordinateSystem5ZInputField.SetText("")
		cp.gcodeParamsCoordinateSystem6XInputField.SetText("")
		cp.gcodeParamsCoordinateSystem6YInputField.SetText("")
		cp.gcodeParamsCoordinateSystem6ZInputField.SetText("")
		if cp.machineCoordinates != nil && cp.gcodeParameters != nil && cp.gcodeParameters.CoordinateOffset != nil && cp.gcodeParameters.ToolLengthOffset != nil {
			updateFunc := func(
				coordinateSystem *grblMod.Coordinates,
				xInputField *tview.InputField,
				yInputField *tview.InputField,
				zInputField *tview.InputField,
			) {
				xInputField.SetText(iFmt.SprintFloat(
					cp.machineCoordinates.X-
						coordinateSystem.X-
						cp.gcodeParameters.CoordinateOffset.X,
					4,
				))
				yInputField.SetText(iFmt.SprintFloat(
					cp.machineCoordinates.Y-
						coordinateSystem.Y-
						cp.gcodeParameters.CoordinateOffset.Y,
					4,
				))
				if cp.gcodeParameters.ToolLengthOffset != nil {
					zInputField.SetText(iFmt.SprintFloat(
						cp.machineCoordinates.Z-
							coordinateSystem.Z-
							cp.gcodeParameters.CoordinateOffset.Z-
							*cp.gcodeParameters.ToolLengthOffset,
						4,
					))
				}
			}
			if cp.gcodeParameters.CoordinateSystem1 != nil {
				updateFunc(
					cp.gcodeParameters.CoordinateSystem1,
					cp.gcodeParamsCoordinateSystem1XInputField,
					cp.gcodeParamsCoordinateSystem1YInputField,
					cp.gcodeParamsCoordinateSystem1ZInputField,
				)
			}
			if cp.gcodeParameters.CoordinateSystem2 != nil {
				updateFunc(
					cp.gcodeParameters.CoordinateSystem2,
					cp.gcodeParamsCoordinateSystem2XInputField,
					cp.gcodeParamsCoordinateSystem2YInputField,
					cp.gcodeParamsCoordinateSystem2ZInputField,
				)
			}
			if cp.gcodeParameters.CoordinateSystem3 != nil {
				updateFunc(
					cp.gcodeParameters.CoordinateSystem3,
					cp.gcodeParamsCoordinateSystem3XInputField,
					cp.gcodeParamsCoordinateSystem3YInputField,
					cp.gcodeParamsCoordinateSystem3ZInputField,
				)
			}
			if cp.gcodeParameters.CoordinateSystem4 != nil {
				updateFunc(
					cp.gcodeParameters.CoordinateSystem4,
					cp.gcodeParamsCoordinateSystem4XInputField,
					cp.gcodeParamsCoordinateSystem4YInputField,
					cp.gcodeParamsCoordinateSystem4ZInputField,
				)
			}
			if cp.gcodeParameters.CoordinateSystem5 != nil {
				updateFunc(
					cp.gcodeParameters.CoordinateSystem5,
					cp.gcodeParamsCoordinateSystem5XInputField,
					cp.gcodeParamsCoordinateSystem5YInputField,
					cp.gcodeParamsCoordinateSystem5ZInputField,
				)
			}
			if cp.gcodeParameters.CoordinateSystem6 != nil {
				updateFunc(
					cp.gcodeParameters.CoordinateSystem6,
					cp.gcodeParamsCoordinateSystem6XInputField,
					cp.gcodeParamsCoordinateSystem6YInputField,
					cp.gcodeParamsCoordinateSystem6ZInputField,
				)
			}
		}
	}
}

func (cp *ControlPrimitive) getCoordinateSystemCoordinates() *grblMod.Coordinates {
	var coordinateSystem *grblMod.Coordinates
	if cp.modalGroup != nil && cp.modalGroup.CoordinateSystemSelect != nil {
		switch cp.modalGroup.CoordinateSystemSelect.NormalizedString() {
		case "G54":
			coordinateSystem = cp.gcodeParameters.CoordinateSystem1
		case "G55":
			coordinateSystem = cp.gcodeParameters.CoordinateSystem2
		case "G56":
			coordinateSystem = cp.gcodeParameters.CoordinateSystem3
		case "G57":
			coordinateSystem = cp.gcodeParameters.CoordinateSystem4
		case "G58":
			coordinateSystem = cp.gcodeParameters.CoordinateSystem5
		case "G59":
			coordinateSystem = cp.gcodeParameters.CoordinateSystem6
		default:
			panic(fmt.Sprintf("bug: unexpected coordinate system: %#v", cp.modalGroup.CoordinateSystemSelect.NormalizedString()))
		}
	}
	return coordinateSystem
}

//gocyclo:ignore
func (cp *ControlPrimitive) updateGcodeParamsCoordinateOffset() {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	if n, _ := cp.gcodeParamsCoordinateOffsetModeDropdown.GetCurrentOption(); n == gcodeParamsCoordinateOffsetModeOffsetIdx && cp.gcodeParameters != nil {
		// Offset
		if cp.gcodeParameters.CoordinateOffset != nil {
			cp.gcodeParamsCoordinateOffsetXInputField.SetText(iFmt.SprintFloat(cp.gcodeParameters.CoordinateOffset.X, 4))
			cp.gcodeParamsCoordinateOffsetYInputField.SetText(iFmt.SprintFloat(cp.gcodeParameters.CoordinateOffset.Y, 4))
			cp.gcodeParamsCoordinateOffsetZInputField.SetText(iFmt.SprintFloat(cp.gcodeParameters.CoordinateOffset.Z, 4))
		} else {
			cp.gcodeParamsCoordinateOffsetXInputField.SetText("")
			cp.gcodeParamsCoordinateOffsetYInputField.SetText("")
			cp.gcodeParamsCoordinateOffsetZInputField.SetText("")
		}
	} else {
		// Work Coordinates
		coordinateSystem := cp.getCoordinateSystemCoordinates()
		if cp.machineCoordinates != nil && coordinateSystem != nil && cp.gcodeParameters.CoordinateOffset != nil {
			cp.gcodeParamsCoordinateOffsetXInputField.SetText(
				iFmt.SprintFloat(
					cp.machineCoordinates.X-
						coordinateSystem.X-
						cp.gcodeParameters.CoordinateOffset.X,
					4,
				),
			)
			cp.gcodeParamsCoordinateOffsetYInputField.SetText(
				iFmt.SprintFloat(
					cp.machineCoordinates.Y-
						coordinateSystem.Y-
						cp.gcodeParameters.CoordinateOffset.Y,
					4,
				),
			)
			if cp.gcodeParameters.ToolLengthOffset != nil {
				cp.gcodeParamsCoordinateOffsetZInputField.SetText(
					iFmt.SprintFloat(
						cp.machineCoordinates.Z-
							coordinateSystem.Z-
							cp.gcodeParameters.CoordinateOffset.Z-
							*cp.gcodeParameters.ToolLengthOffset,
						4,
					),
				)
			} else {
				cp.gcodeParamsCoordinateOffsetZInputField.SetText("")
			}
		} else {
			cp.gcodeParamsCoordinateOffsetXInputField.SetText("")
			cp.gcodeParamsCoordinateOffsetYInputField.SetText("")
			cp.gcodeParamsCoordinateOffsetZInputField.SetText("")
		}
	}
}

func (cp *ControlPrimitive) newGcodeParams() {
	// Coordinate System
	coordinateSystemTitleTextView := tview.NewTextView()
	coordinateSystemTitleTextView.SetText("Coordinate System")
	cp.gcodeParamsCoordinateSystemModeDropdown = tview.NewDropDown()
	cp.gcodeParamsCoordinateSystemModeDropdown.SetLabel("Mode:")
	cp.gcodeParamsCoordinateSystemModeDropdown.SetOptions(gcodeParamsCoordinateSystemModeOptions, nil)
	cp.gcodeParamsCoordinateSystemModeDropdown.SetCurrentOption(gcodeParamsCoordinateSystemModeWorkCoordinatesIdx)
	cp.gcodeParamsCoordinateSystemModeDropdown.SetSelectedFunc(func(string, int) {
		cp.updateGcodeParamsCoordinateSystem()
	})
	newCoordinateSystemPrimitives := func(number, word string) (*tview.InputField, *tview.InputField, *tview.InputField, *tview.Flex) {
		labelTextView := tview.NewTextView()
		labelTextView.SetLabel(fmt.Sprintf("%s%s:", number, sprintGcodeWord(word)))

		getChangedFunc := func(letter string, inputField *tview.InputField) func(tcell.Key) {
			return func(tcell.Key) {
				if cp.skipQueueCommand {
					return
				}
				if n, _ := cp.gcodeParamsCoordinateSystemModeDropdown.GetCurrentOption(); n == gcodeParamsCoordinateSystemModeOffsetIdx {
					cp.QueueCommandIgnoreResponse(fmt.Sprintf("G10L2P%s%s%s", number, letter, inputField.GetText()))
				} else {
					cp.QueueCommandIgnoreResponse(fmt.Sprintf("G10L20P%s%s%s", number, letter, inputField.GetText()))
				}
			}
		}

		getCoordinateInputField := func(letter string) *tview.InputField {
			coordinateInputField := tview.NewInputField()
			coordinateInputField.SetLabel(fmt.Sprintf("%s:", letter))
			coordinateInputField.SetDoneFunc(getChangedFunc(letter, coordinateInputField))
			coordinateInputField.SetAcceptanceFunc(acceptFloat)
			coordinateInputField.SetBorderPadding(0, 0, 1, 0)
			return coordinateInputField
		}

		x := getCoordinateInputField("X")
		y := getCoordinateInputField("Y")
		z := getCoordinateInputField("Z")

		flex := tview.NewFlex()
		flex.SetDirection(tview.FlexColumn)
		flex.AddItem(labelTextView, 5, 0, false)
		flex.AddItem(x, 0, 1, false)
		flex.AddItem(y, 0, 1, false)
		flex.AddItem(z, 0, 1, false)

		return x, y, z, flex
	}
	var coordinateSystem1Flex, coordinateSystem2Flex, coordinateSystem3Flex, coordinateSystem4Flex, coordinateSystem5Flex, coordinateSystem6Flex *tview.Flex
	cp.gcodeParamsCoordinateSystem1XInputField,
		cp.gcodeParamsCoordinateSystem1YInputField,
		cp.gcodeParamsCoordinateSystem1ZInputField,
		coordinateSystem1Flex = newCoordinateSystemPrimitives("1", "G54")
	cp.gcodeParamsCoordinateSystem2XInputField,
		cp.gcodeParamsCoordinateSystem2YInputField,
		cp.gcodeParamsCoordinateSystem2ZInputField,
		coordinateSystem2Flex = newCoordinateSystemPrimitives("2", "G55")
	cp.gcodeParamsCoordinateSystem3XInputField,
		cp.gcodeParamsCoordinateSystem3YInputField,
		cp.gcodeParamsCoordinateSystem3ZInputField,
		coordinateSystem3Flex = newCoordinateSystemPrimitives("3", "G56")
	cp.gcodeParamsCoordinateSystem4XInputField,
		cp.gcodeParamsCoordinateSystem4YInputField,
		cp.gcodeParamsCoordinateSystem4ZInputField,
		coordinateSystem4Flex = newCoordinateSystemPrimitives("4", "G57")
	cp.gcodeParamsCoordinateSystem5XInputField,
		cp.gcodeParamsCoordinateSystem5YInputField,
		cp.gcodeParamsCoordinateSystem5ZInputField,
		coordinateSystem5Flex = newCoordinateSystemPrimitives("5", "G58")
	cp.gcodeParamsCoordinateSystem6XInputField,
		cp.gcodeParamsCoordinateSystem6YInputField,
		cp.gcodeParamsCoordinateSystem6ZInputField,
		coordinateSystem6Flex = newCoordinateSystemPrimitives("6", "G59")

	// Pre-Defined Position
	preDefinedPositionTitleTextView := tview.NewTextView()
	preDefinedPositionTitleTextView.SetText("Pre-Defined Position")
	newPreDefinedPositionPrimitives := func(number, word string) (
		*tview.InputField, *tview.InputField, *tview.InputField,
		*tview.Button, *tview.Button,
		*tview.Flex,
	) {
		labelTextView := tview.NewTextView()
		labelTextView.SetLabel(fmt.Sprintf("%s%s:", number, sprintGcodeWord(word)))

		getDoneFunc := func(letter string, inputField *tview.InputField) func(tcell.Key) {
			return func(tcell.Key) {
				if cp.skipQueueCommand {
					return
				}
				cp.QueueCommandIgnoreResponse(fmt.Sprintf("%s.1%s%s", word, letter, inputField.GetText()))
			}
		}

		getCoordinateInputField := func(letter string) *tview.InputField {
			coordinateInputField := tview.NewInputField()
			coordinateInputField.SetLabel(fmt.Sprintf("%s:", letter))
			coordinateInputField.SetDoneFunc(getDoneFunc(letter, coordinateInputField))
			coordinateInputField.SetAcceptanceFunc(acceptFloat)
			coordinateInputField.SetBorderPadding(0, 0, 0, 1)
			return coordinateInputField
		}

		x := getCoordinateInputField("X")
		y := getCoordinateInputField("Y")
		z := getCoordinateInputField("Z")

		paramsFlex := tview.NewFlex()
		paramsFlex.SetDirection(tview.FlexColumn)
		paramsFlex.AddItem(labelTextView, 6, 0, false)
		paramsFlex.AddItem(x, 0, 1, false)
		paramsFlex.AddItem(y, 0, 1, false)
		paramsFlex.AddItem(z, 0, 1, false)

		goButton := tview.NewButton("Go To")
		goButton.SetSelectedFunc(func() {
			cp.QueueCommandIgnoreResponse(word)
		})

		spacerTextView := tview.NewTextView()

		setToCurrentButton := tview.NewButton("Set to Current")
		setToCurrentButton.SetSelectedFunc(func() {
			cp.QueueCommandIgnoreResponse(fmt.Sprintf("%s.1", word))
		})

		buttonsFlex := tview.NewFlex()
		buttonsFlex.SetDirection(tview.FlexColumn)
		buttonsFlex.AddItem(goButton, 0, 1, false)
		buttonsFlex.AddItem(spacerTextView, 2, 0, false)
		buttonsFlex.AddItem(setToCurrentButton, 0, 1, false)
		buttonsFlex.SetBorderPadding(0, 0, 2, 2)

		flex := tview.NewFlex()
		flex.SetDirection(tview.FlexRow)
		flex.AddItem(paramsFlex, 1, 0, false)
		flex.AddItem(buttonsFlex, 1, 0, false)

		return x, y, z, goButton, setToCurrentButton, flex
	}
	var preDefinedPosition1Flex, preDefinedPosition2Flex *tview.Flex
	cp.gcodeParamsPreDefinedPosition1XInputField,
		cp.gcodeParamsPreDefinedPosition1YInputField,
		cp.gcodeParamsPreDefinedPosition1ZInputField,
		cp.gcodeParamsPreDefinedPosition1GoToButton,
		cp.gcodeParamsPreDefinedPosition1SetToCurrentButton,
		preDefinedPosition1Flex = newPreDefinedPositionPrimitives("1", "G28")
	cp.gcodeParamsPreDefinedPosition2XInputField,
		cp.gcodeParamsPreDefinedPosition2YInputField,
		cp.gcodeParamsPreDefinedPosition2ZInputField,
		cp.gcodeParamsPreDefinedPosition2GoToButton,
		cp.gcodeParamsPreDefinedPosition2SetToCurrentButton,
		preDefinedPosition2Flex = newPreDefinedPositionPrimitives("1", "G30")

	// Coordinate Offset
	coordinateOffsetTitleTextView := tview.NewTextView()
	coordinateOffsetTitleTextView.SetText("Coordinate Offset")
	cp.gcodeParamsCoordinateOffsetModeDropdown = tview.NewDropDown()
	cp.gcodeParamsCoordinateOffsetModeDropdown.SetLabel("Mode:")
	cp.gcodeParamsCoordinateOffsetModeDropdown.SetOptions(gcodeParamsCoordinateOffsetModeOptions, nil)
	cp.gcodeParamsCoordinateOffsetModeDropdown.SetCurrentOption(gcodeParamsCoordinateOffsetModeWorkCoordinatesIdx)
	cp.gcodeParamsCoordinateOffsetModeDropdown.SetSelectedFunc(func(string, int) {
		cp.updateGcodeParamsCoordinateOffset()
	})
	coordinateOffsetLabelTextView := tview.NewTextView()
	coordinateOffsetLabelTextView.SetLabel(fmt.Sprintf("%s:", sprintGcodeWord("G92")))
	getCoordinateOffsetCoordinateInputField := func(letter string) *tview.InputField {
		coordinateInputField := tview.NewInputField()
		coordinateInputField.SetLabel(fmt.Sprintf("%s:", letter))
		coordinateInputField.SetDoneFunc(func(tcell.Key) {
			if cp.skipQueueCommand {
				return
			}
			var coordinateOffsetStr string
			if n, _ := cp.gcodeParamsCoordinateOffsetModeDropdown.GetCurrentOption(); n == gcodeParamsCoordinateOffsetModeOffsetIdx {
				coordinateOffsetStr = coordinateInputField.GetText()
			} else {
				var machineCoordinate, coordinateSystemCoordinate *float64
				machineCoordinate = cp.machineCoordinates.GetAxis(letter)
				coordinateSystemCoordinates := cp.getCoordinateSystemCoordinates()
				coordinateSystemCoordinate = coordinateSystemCoordinates.GetAxis(letter)
				workCoordinate, err := strconv.ParseFloat(coordinateInputField.GetText(), 64)
				if err != nil {
					panic(fmt.Errorf("bug: parsing not expected to fail: %w", err))
				}
				if machineCoordinate == nil || coordinateSystemCoordinate == nil || cp.gcodeParameters.ToolLengthOffset == nil {
					return
				}
				coordinateOffset := *machineCoordinate - *coordinateSystemCoordinate - *cp.gcodeParameters.ToolLengthOffset - workCoordinate
				coordinateOffsetStr = iFmt.SprintFloat(coordinateOffset, 4)
			}
			cp.QueueCommandIgnoreResponse(fmt.Sprintf("G92%s%s", letter, coordinateOffsetStr))
		})
		coordinateInputField.SetAcceptanceFunc(acceptFloat)
		coordinateInputField.SetBorderPadding(0, 0, 1, 0)
		return coordinateInputField
	}
	cp.gcodeParamsCoordinateOffsetXInputField = getCoordinateOffsetCoordinateInputField("X")
	cp.gcodeParamsCoordinateOffsetYInputField = getCoordinateOffsetCoordinateInputField("Y")
	cp.gcodeParamsCoordinateOffsetZInputField = getCoordinateOffsetCoordinateInputField("Z")
	coordinateOffsetFlex := tview.NewFlex()
	coordinateOffsetFlex.SetDirection(tview.FlexColumn)
	coordinateOffsetFlex.AddItem(coordinateOffsetLabelTextView, 5, 0, false)
	coordinateOffsetFlex.AddItem(cp.gcodeParamsCoordinateOffsetXInputField, 0, 1, false)
	coordinateOffsetFlex.AddItem(cp.gcodeParamsCoordinateOffsetYInputField, 0, 1, false)
	coordinateOffsetFlex.AddItem(cp.gcodeParamsCoordinateOffsetZInputField, 0, 1, false)

	// G-Code: Parameters
	cp.gcodeParamsScrollContainer = NewScrollContainer()
	cp.gcodeParamsScrollContainer.SetBorder(true)
	cp.gcodeParamsScrollContainer.SetTitle("G-Code: Parameters")
	cp.gcodeParamsScrollContainer.AddPrimitive(coordinateSystemTitleTextView, 1)
	cp.gcodeParamsScrollContainer.AddPrimitive(cp.gcodeParamsCoordinateSystemModeDropdown, 1)
	cp.gcodeParamsScrollContainer.AddPrimitive(coordinateSystem1Flex, 1)
	cp.gcodeParamsScrollContainer.AddPrimitive(coordinateSystem2Flex, 1)
	cp.gcodeParamsScrollContainer.AddPrimitive(coordinateSystem3Flex, 1)
	cp.gcodeParamsScrollContainer.AddPrimitive(coordinateSystem4Flex, 1)
	cp.gcodeParamsScrollContainer.AddPrimitive(coordinateSystem5Flex, 1)
	cp.gcodeParamsScrollContainer.AddPrimitive(coordinateSystem6Flex, 1)
	cp.gcodeParamsScrollContainer.AddPrimitive(preDefinedPositionTitleTextView, 1)
	cp.gcodeParamsScrollContainer.AddPrimitive(preDefinedPosition1Flex, 2)
	cp.gcodeParamsScrollContainer.AddPrimitive(preDefinedPosition2Flex, 2)
	cp.gcodeParamsScrollContainer.AddPrimitive(coordinateOffsetTitleTextView, 1)
	cp.gcodeParamsScrollContainer.AddPrimitive(cp.gcodeParamsCoordinateOffsetModeDropdown, 1)
	cp.gcodeParamsScrollContainer.AddPrimitive(coordinateOffsetFlex, 1)
}

func (cp *ControlPrimitive) newCommands() {
	commandsTextView := tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true).
		SetWrap(true)
	commandsTextView.SetBorder(true).SetTitle("Commands")
	commandsTextView.SetChangedFunc(func() {
		cp.app.QueueUpdateDraw(func() {
			text := commandsTextView.GetText(false)
			if len(text) > 0 && text[0] == '\n' {
				commandsTextView.SetText(text[1:])
			}
			commandsTextView.ScrollToEnd()
		})
	})
	cp.commandsTextView = commandsTextView
}

func (cp *ControlPrimitive) newPushMessages() {
	pushMessagesTextView := tview.NewTextView()
	pushMessagesTextView.SetDynamicColors(true)
	pushMessagesTextView.SetScrollable(true)
	pushMessagesTextView.SetWrap(true)
	pushMessagesTextView.SetBorder(true).SetTitle("Push Messages")
	pushMessagesTextView.SetChangedFunc(func() {
		cp.app.QueueUpdateDraw(func() {
			text := pushMessagesTextView.GetText(false)
			if len(text) > 0 && text[0] == '\n' {
				pushMessagesTextView.SetText(text[1:])
			}
			pushMessagesTextView.ScrollToEnd()
		})
	})
	cp.pushMessagesTextView = pushMessagesTextView
}

func (cp *ControlPrimitive) newCommand() {
	commandInputField := tview.NewInputField()
	commandInputField.SetLabel("Command:")
	commandInputField.SetDoneFunc(func(key tcell.Key) {
		switch key {
		case tcell.KeyEscape:
			commandInputField.SetText("")
		case tcell.KeyEnter:
			command := commandInputField.GetText()
			if command == "" {
				return
			}
			cp.QueueCommandIgnoreResponse(command)
			cp.commandInputHistory = append([]string{command}, cp.commandInputHistory...)
			cp.commandInputHistoryIdx = -1
			commandInputField.SetText("")
		}
	})
	commandInputField.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyUp:
			if cp.commandInputHistoryIdx == len(cp.commandInputHistory)-1 {
				return nil
			}
			cp.commandInputHistoryIdx++
			cp.commandInputField.SetText(cp.commandInputHistory[cp.commandInputHistoryIdx])
			return nil
		case tcell.KeyDown:
			if cp.commandInputHistoryIdx == -1 {
				return nil
			}
			cp.commandInputHistoryIdx--
			if cp.commandInputHistoryIdx < 0 {
				return nil
			}
			cp.commandInputField.SetText(cp.commandInputHistory[cp.commandInputHistoryIdx])
			return nil
		}
		return event
	})
	cp.commandInputField = commandInputField
}

func (cp *ControlPrimitive) newControl() {
	gcodeFlex := tview.NewFlex()
	gcodeFlex.SetDirection(tview.FlexColumn)
	gcodeFlex.AddItem(cp.gcodeParserFlex, 0, 1, false)
	gcodeFlex.AddItem(cp.gcodeParamsScrollContainer, 0, 1, false)

	commsFlex := tview.NewFlex()
	commsFlex.SetDirection(tview.FlexColumn)
	commsFlex.AddItem(cp.commandsTextView, 0, 1, false)
	commsFlex.AddItem(cp.pushMessagesTextView, 0, 1, false)

	controlFlex := tview.NewFlex()
	controlFlex.SetBorder(true)
	controlFlex.SetTitle("Contrtol")
	controlFlex.SetDirection(tview.FlexRow)
	controlFlex.AddItem(gcodeFlex, 20, 0, false)
	controlFlex.AddItem(commsFlex, 0, 1, false)
	controlFlex.AddItem(cp.commandInputField, 1, 0, true)
	cp.Flex = controlFlex
}

func (cp *ControlPrimitive) sendStatusCommands() {
	for command := range allStatusCommands {
		cp.queueStatusCommand(command)
	}
}

func (cp *ControlPrimitive) setDisabledState() {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	var disabled bool
	if cp.disableCommandInput {
		disabled = true
	} else {
		switch cp.state {
		case grblMod.StateIdle:
			disabled = false
		case grblMod.StateRun:
			disabled = true
		case grblMod.StateHold:
			disabled = true
		case grblMod.StateJog:
			disabled = true
		case grblMod.StateAlarm:
			disabled = true
		case grblMod.StateDoor:
			disabled = true
		case grblMod.StateCheck:
			disabled = false
		case grblMod.StateHome:
			disabled = true
		case grblMod.StateSleep:
			disabled = true
		case grblMod.StateUnknown:
			disabled = true
		default:
			panic(fmt.Errorf("unknown state: %s", cp.state))
		}
	}
	// G-Code: Modal Groups
	cp.gcodeParserModalGroupsMotionDropDown.SetDisabled(disabled)
	cp.gcodeParserModalGroupsPlaneSelectionDropDown.SetDisabled(disabled)
	cp.gcodeParserModalGroupsDistanceModeDropDown.SetDisabled(disabled)
	cp.gcodeParserModalGroupsFeedRateModeDropDown.SetDisabled(disabled)
	cp.gcodeParserModalGroupsUnitsDropDown.SetDisabled(disabled)
	cp.gcodeParserModalGroupsToolLengthOffsetInputField.SetDisabled(disabled)
	cp.gcodeParserModalGroupsCoordinateSystemSelectDropDown.SetDisabled(disabled)
	cp.gcodeParserModalGroupsStoppingCheckbox.SetDisabled(disabled)
	cp.gcodeParserModalGroupsSpindleDropDown.SetDisabled(disabled)
	cp.gcodeParserModalGroupsCoolantMistCheckbok.SetDisabled(disabled)
	cp.gcodeParserModalGroupsCoolantFloodCheckbok.SetDisabled(disabled)
	cp.gcodeParserModalGroupsCoolantOff.SetDisabled(disabled)
	// G-Code: Parser State
	cp.gcodeParserStateToolInputField.SetDisabled(disabled)
	cp.gcodeParserStateSpindleSpeedInputField.SetDisabled(disabled)
	cp.gcodeParserStateFeedRateInputField.SetDisabled(disabled)
	// G-Code: Parameters: Coordinate System
	cp.gcodeParamsCoordinateSystemModeDropdown.SetDisabled(disabled)
	cp.gcodeParamsCoordinateSystem1XInputField.SetDisabled(disabled)
	cp.gcodeParamsCoordinateSystem1YInputField.SetDisabled(disabled)
	cp.gcodeParamsCoordinateSystem1ZInputField.SetDisabled(disabled)
	cp.gcodeParamsCoordinateSystem2XInputField.SetDisabled(disabled)
	cp.gcodeParamsCoordinateSystem2YInputField.SetDisabled(disabled)
	cp.gcodeParamsCoordinateSystem2ZInputField.SetDisabled(disabled)
	cp.gcodeParamsCoordinateSystem3XInputField.SetDisabled(disabled)
	cp.gcodeParamsCoordinateSystem3YInputField.SetDisabled(disabled)
	cp.gcodeParamsCoordinateSystem3ZInputField.SetDisabled(disabled)
	cp.gcodeParamsCoordinateSystem4XInputField.SetDisabled(disabled)
	cp.gcodeParamsCoordinateSystem4YInputField.SetDisabled(disabled)
	cp.gcodeParamsCoordinateSystem4ZInputField.SetDisabled(disabled)
	cp.gcodeParamsCoordinateSystem5XInputField.SetDisabled(disabled)
	cp.gcodeParamsCoordinateSystem5YInputField.SetDisabled(disabled)
	cp.gcodeParamsCoordinateSystem5ZInputField.SetDisabled(disabled)
	cp.gcodeParamsCoordinateSystem6XInputField.SetDisabled(disabled)
	cp.gcodeParamsCoordinateSystem6YInputField.SetDisabled(disabled)
	cp.gcodeParamsCoordinateSystem6ZInputField.SetDisabled(disabled)
	// G-Code: Parameters: Pre-Defined Position
	cp.gcodeParamsPreDefinedPosition1XInputField.SetDisabled(disabled)
	cp.gcodeParamsPreDefinedPosition1YInputField.SetDisabled(disabled)
	cp.gcodeParamsPreDefinedPosition1ZInputField.SetDisabled(disabled)
	cp.gcodeParamsPreDefinedPosition1GoToButton.SetDisabled(disabled)
	cp.gcodeParamsPreDefinedPosition1SetToCurrentButton.SetDisabled(disabled)
	cp.gcodeParamsPreDefinedPosition2XInputField.SetDisabled(disabled)
	cp.gcodeParamsPreDefinedPosition2YInputField.SetDisabled(disabled)
	cp.gcodeParamsPreDefinedPosition2ZInputField.SetDisabled(disabled)
	cp.gcodeParamsPreDefinedPosition2GoToButton.SetDisabled(disabled)
	cp.gcodeParamsPreDefinedPosition2SetToCurrentButton.SetDisabled(disabled)
	// G-Code: Parameters: Coordinate Offset
	cp.gcodeParamsCoordinateOffsetModeDropdown.SetDisabled(disabled)
	cp.gcodeParamsCoordinateOffsetXInputField.SetDisabled(disabled)
	cp.gcodeParamsCoordinateOffsetYInputField.SetDisabled(disabled)
	cp.gcodeParamsCoordinateOffsetZInputField.SetDisabled(disabled)
	// Command
	cp.commandInputField.SetDisabled(disabled)
}

func (cp *ControlPrimitive) setState(state grblMod.State) {
	cp.mu.Lock()
	cp.state = state
	cp.mu.Unlock()

	var stopping bool
	if state == grblMod.StateHold {
		stopping = true
	}
	cp.gcodeParserModalGroupsStoppingCheckbox.SetChecked(stopping)

	cp.setDisabledState()
}

func (cp *ControlPrimitive) DisableCommandInput(disabled bool) {
	cp.mu.Lock()
	cp.disableCommandInput = disabled
	cp.mu.Unlock()
	cp.app.QueueUpdateDraw(func() { cp.setDisabledState() })
}

func (cp *ControlPrimitive) sendCommand(ctx context.Context, commandParameter *commandParameterType) error {
	if !commandParameter.quiet {
		fmt.Fprintf(cp.commandsTextView, "\n[%s]%s[-]", tcell.ColorWhite, tview.Escape(commandParameter.command))
	}

	cmdCtx := ctx
	if commandParameter.timeout > 0 {
		var cancel context.CancelFunc
		cmdCtx, cancel = context.WithDeadline(ctx, time.Now().Add(commandParameter.timeout))
		defer cancel()
	}

	err := cp.grbl.SendCommand(cmdCtx, commandParameter.command)
	if err != nil {
		var errResponseMessage *grblMod.ErrResponseMessage
		if errors.As(err, &errResponseMessage) {
			responseMessage := errResponseMessage.ResponseMessage()
			fmt.Fprintf(cp.commandsTextView, "\n[%s]%s[-]", tcell.ColorRed, tview.Escape(responseMessage.String()))
			fmt.Fprintf(cp.commandsTextView, "\n[%s]%s[-]", tcell.ColorRed, tview.Escape(errResponseMessage.Error()))
		} else {
			fmt.Fprintf(cp.commandsTextView, "\n[%s]Send command failed: %#v: %s[-]", tcell.ColorRed, tview.Escape(commandParameter.command), tview.Escape(err.Error()))
		}
		return err
	}

	if commandParameter.quiet {
		return nil
	}

	fmt.Fprintf(cp.commandsTextView, "\n[%s]ok[-]", tcell.ColorGreen)

	return nil
}

func (cp *ControlPrimitive) extractRealTimeCommands(command string) ([]grblMod.RealTimeCommand, string, error) {
	var cmdBuffer bytes.Buffer
	realTimeCommands := []grblMod.RealTimeCommand{}
	for _, c := range []byte(command) {
		realTimeCommand, err := grblMod.NewRealTimeCommand(c)
		if err != nil {
			if !errors.Is(err, grblMod.ErrNotRealTimeCommand) {
				return nil, "", fmt.Errorf("real time command parsing fail: %s", err.Error())
			}
			cmdBuffer.WriteByte(c)
		} else {
			realTimeCommands = append(realTimeCommands, realTimeCommand)
			if realTimeCommand == grblMod.RealTimeCommandStatusReportQuery && cp.quietStatusComms {
				fmt.Fprintf(cp.pushMessagesTextView, "\n[%s](push messages from ? omitted, results at Control panel)[-]", tcell.ColorYellow)
			}
		}
	}
	return realTimeCommands, cmdBuffer.String(), nil
}

//gocyclo:ignore
func (cp *ControlPrimitive) getStatusCmdsAndTimeout(block *gcode.Block) (map[string]bool, *time.Duration) {
	statusCommands := map[string]bool{}
	var timeout *time.Duration
	if block.IsSystem() {
		if _, ok := allStatusCommands[block.String()]; ok && cp.quietStatusComms {
			fmt.Fprintf(cp.pushMessagesTextView, "\n[%s](push messages from %s omitted, results at Control panel)[-]", tcell.ColorYellow, block.String())
		}
		switch block.String() {
		case grblMod.GrblCommandRestoreGcodeParametersToDefaults:
			statusCommands[grblMod.GrblCommandViewGcodeParameters] = true
		case grblMod.GrblCommandRestoreAllToDefaults:
			statusCommands[grblMod.GrblCommandViewGcodeParameters] = true
			statusCommands[grblMod.GrblCommandViewGrblSettings] = true
			statusCommands[grblMod.GrblCommandViewStartupBlocks] = true
			statusCommands[grblMod.GrblCommandViewBuildInfo] = true
		case grblMod.GrblCommandRestoreGrblSettingsToDefaults:
			statusCommands[grblMod.GrblCommandViewStartupBlocks] = true
		}
		if strings.HasPrefix(block.String(), grblMod.GrblCommandRunHomingCyclePrefix) {
			timeout = &homeCommandTimeout
			cp.stateTracker.HomeOverride(true)
		}
		matched, err := regexp.MatchString(`^\$[0-9]+=`, block.String())
		if err != nil {
			panic(err)
		}
		if matched {
			statusCommands[grblMod.GrblCommandViewGrblSettings] = true
		}
		if strings.HasPrefix(block.String(), grblMod.GrblCommandWriteBuildInfoPrefix) {
			statusCommands[grblMod.GrblCommandViewBuildInfo] = true
		}
		if strings.HasPrefix(block.String(), grblMod.GrblCommandSaveStartupBlockPrefix) {
			statusCommands[grblMod.GrblCommandViewStartupBlocks] = true
		}
	} else if block.IsCommand() {
		for _, word := range block.Words() {
			switch word.NormalizedString() {
			case "M0":
				var zeroDuration time.Duration
				timeout = &zeroDuration
			case "G38.2", "G38.3", "G38.4", "G38.5":
				timeout = &probeCommandTimeout
			}
		}
		statusCommands[grblMod.GrblCommandViewGcodeParserState] = true
		statusCommands[grblMod.GrblCommandViewGcodeParameters] = true
	} else {
		panic("bug: unknown block")
	}
	return statusCommands, timeout
}

func (cp *ControlPrimitive) processCommand(ctx context.Context, command string) error {
	var err error
	var realTimeCommands []grblMod.RealTimeCommand
	realTimeCommands, command, err = cp.extractRealTimeCommands(command)
	if err != nil {
		fmt.Fprintf(cp.commandsTextView, "\n[%s]%s[-]", tcell.ColorRed, tview.Escape(err.Error()))
		return err
	}
	for _, realTimeCommand := range realTimeCommands {
		cp.sendRealTimeCommand(realTimeCommand)
	}
	if len(command) == 0 {
		return nil
	}

	blocks, err := gcode.NewParser(strings.NewReader(command)).Blocks()
	if err != nil {
		fmt.Fprintf(cp.commandsTextView, "\n[%s]Failed to parse: %s[-]", tcell.ColorRed, tview.Escape(err.Error()))
		return err
	}
	if len(blocks) > 1 {
		panic("bug: expected single block")
	}

	commandParameter := &commandParameterType{
		command: command,
		timeout: defaultCommandTimeout,
	}
	var statusCommands map[string]bool
	if len(blocks) > 0 {
		block := blocks[0]
		var timeout *time.Duration
		statusCommands, timeout = cp.getStatusCmdsAndTimeout(block)
		if timeout != nil {
			commandParameter.timeout = *timeout
		}
	}

	cp.DisableCommandInput(true)
	defer cp.DisableCommandInput(false)

	err = cp.sendCommand(ctx, commandParameter)
	if len(statusCommands) > 0 {
		cp.sendCommand(ctx, syncCommand)
	}
	for command := range statusCommands {
		cp.sendCommand(ctx, &commandParameterType{
			command: command,
			timeout: defaultCommandTimeout,
			quiet:   cp.quietStatusComms,
		})
	}
	cp.stateTracker.HomeOverride(false)
	return err
}

func (cp *ControlPrimitive) queueStatusCommand(command string) {
	cp.sendStatusCommandCh <- command
}

func (cp *ControlPrimitive) QueueCommandIgnoreResponse(command string) {
	cp.sendCommandCh <- &queuedCommandType{
		command: command,
	}
}

func (cp *ControlPrimitive) QueueCommand(command string) <-chan error {
	errCh := make(chan error, 1)
	cp.sendCommandCh <- &queuedCommandType{
		command: command,
		errCh:   errCh,
	}
	return errCh
}

func (cp *ControlPrimitive) sendRealTimeCommand(
	cmd grblMod.RealTimeCommand,
) {
	textView := cp.commandsTextView

	if cp.quietStatusComms || cmd != grblMod.RealTimeCommandStatusReportQuery {
		fmt.Fprintf(textView, "\n[%s]%s[-]", tcell.ColorBlue, tview.Escape(cmd.String()))
	}
	if err := cp.grbl.SendRealTimeCommand(cmd); err != nil {
		fmt.Fprintf(textView, "\n[%s]Failed to send real-time command: %s[-]", tcell.ColorRed, err)
	}
}

func (cp *ControlPrimitive) QueueRealTimeCommand(rtc grblMod.RealTimeCommand) {
	cp.sendRealTimeCommandCh <- rtc
}

//gocyclo:ignore
func (cp *ControlPrimitive) processGcodeStatePushMessage(
	gcodeStatePushMessage *grblMod.GcodeStatePushMessage,
) tcell.Color {
	cp.mu.Lock()
	newModalGroup := *gcodeStatePushMessage.ModalGroup
	cp.modalGroup = &newModalGroup
	cp.mu.Unlock()

	cp.app.QueueUpdateDraw(func() {
		cp.skipQueueCommand = true
		defer func() { cp.skipQueueCommand = false }()

		// G-Code: Modal Groups
		if modalGroup := gcodeStatePushMessage.ModalGroup; modalGroup != nil {
			setDropDownFn := func(modalWord *gcode.Word, words []string, dropDown *tview.DropDown) {
				index := -1
				for i, word := range words {
					if word == modalWord.NormalizedString() {
						index = i
					}
				}
				dropDown.SetCurrentOption(index)
			}
			if modalGroup.Motion != nil {
				setDropDownFn(
					modalGroup.Motion,
					gcodeParserModalGroupsMotionWords,
					cp.gcodeParserModalGroupsMotionDropDown,
				)
			}
			if modalGroup.PlaneSelection != nil {
				setDropDownFn(
					modalGroup.PlaneSelection,
					gcodeParserModalGroupsPlaneSelectionWords,
					cp.gcodeParserModalGroupsPlaneSelectionDropDown,
				)
			}
			if modalGroup.DistanceMode != nil {
				setDropDownFn(
					modalGroup.DistanceMode,
					gcodeParserModalGroupsDistanceModeWords,
					cp.gcodeParserModalGroupsDistanceModeDropDown,
				)
			}
			if modalGroup.FeedRateMode != nil {
				setDropDownFn(
					modalGroup.FeedRateMode,
					gcodeParserModalGroupsFeedRateModeWords,
					cp.gcodeParserModalGroupsFeedRateModeDropDown,
				)
			}
			if modalGroup.Units != nil {
				setDropDownFn(
					modalGroup.Units,
					gcodeParserModalGroupsUnitsWords,
					cp.gcodeParserModalGroupsUnitsDropDown,
				)
			}
			if modalGroup.CoordinateSystemSelect != nil {
				setDropDownFn(
					modalGroup.CoordinateSystemSelect,
					gcodeParserModalGroupsCoordinateSystemSelectWords,
					cp.gcodeParserModalGroupsCoordinateSystemSelectDropDown,
				)
			}
			if modalGroup.Spindle != nil {
				setDropDownFn(
					modalGroup.Spindle,
					gcodeParserModalGroupsSpindleWords,
					cp.gcodeParserModalGroupsSpindleDropDown,
				)
			}
			cp.gcodeParserModalGroupsCoolantMistCheckbok.SetChecked(false)
			cp.gcodeParserModalGroupsCoolantFloodCheckbok.SetChecked(false)
			for _, word := range modalGroup.Coolant {
				switch word.NormalizedString() {
				case "M7":
					cp.gcodeParserModalGroupsCoolantMistCheckbok.SetChecked(true)
				case "M8":
					cp.gcodeParserModalGroupsCoolantFloodCheckbok.SetChecked(true)
				case "M9":
				default:
					panic(fmt.Sprintf("bug: unexpected word: %#v", word.NormalizedString()))
				}
			}
		}

		// G-Code: Parser State
		if gcodeStatePushMessage.Tool != nil {
			cp.gcodeParserStateToolInputField.SetText(
				iFmt.SprintFloat(*gcodeStatePushMessage.Tool, 4),
			)
		}
		if gcodeStatePushMessage.SpindleSpeed != nil {
			cp.gcodeParserStateSpindleSpeedInputField.SetText(
				iFmt.SprintFloat(*gcodeStatePushMessage.SpindleSpeed, 4),
			)
		}
		if gcodeStatePushMessage.FeedRate != nil {
			cp.gcodeParserStateFeedRateInputField.SetText(
				iFmt.SprintFloat(*gcodeStatePushMessage.FeedRate, 4),
			)
		}

		// G-Code: Coordinate Offset
		cp.updateGcodeParamsCoordinateOffset()
	})

	return tcell.ColorGreen
}

//gocyclo:ignore
func (cp *ControlPrimitive) processGcodeParamPushMessage() tcell.Color {
	color := tcell.ColorGreen

	var update bool

	gcodeParameters := cp.grbl.GetLastGcodeParameters()
	cp.mu.Lock()
	if !reflect.DeepEqual(gcodeParameters, cp.gcodeParameters) {
		newGcodeParameters := *gcodeParameters
		cp.gcodeParameters = &newGcodeParameters
		update = true
	}
	cp.mu.Unlock()

	if !update {
		return color
	}

	// G-Code: Parameters
	cp.app.QueueUpdateDraw(func() {
		// Coordinate System
		cp.updateGcodeParamsCoordinateSystem()
		// Pre-Defined Position
		if gcodeParameters.PrimaryPreDefinedPosition != nil {
			cp.gcodeParamsPreDefinedPosition1XInputField.SetText(iFmt.SprintFloat(gcodeParameters.PrimaryPreDefinedPosition.X, 4))
			cp.gcodeParamsPreDefinedPosition1YInputField.SetText(iFmt.SprintFloat(gcodeParameters.PrimaryPreDefinedPosition.Y, 4))
			cp.gcodeParamsPreDefinedPosition1ZInputField.SetText(iFmt.SprintFloat(gcodeParameters.PrimaryPreDefinedPosition.Z, 4))
		}
		if gcodeParameters.SecondaryPreDefinedPosition != nil {
			cp.gcodeParamsPreDefinedPosition2XInputField.SetText(iFmt.SprintFloat(gcodeParameters.SecondaryPreDefinedPosition.X, 4))
			cp.gcodeParamsPreDefinedPosition2YInputField.SetText(iFmt.SprintFloat(gcodeParameters.SecondaryPreDefinedPosition.Y, 4))
			cp.gcodeParamsPreDefinedPosition2ZInputField.SetText(iFmt.SprintFloat(gcodeParameters.SecondaryPreDefinedPosition.Z, 4))
		}
		// Coordinate Offset
		cp.updateGcodeParamsCoordinateOffset()
		// Tool Length Offset
		if gcodeParameters.ToolLengthOffset != nil {
			cp.gcodeParserModalGroupsToolLengthOffsetInputField.SetText(iFmt.SprintFloat(*gcodeParameters.ToolLengthOffset, 4))
		} else {
			cp.gcodeParserModalGroupsToolLengthOffsetInputField.SetText("")
		}
	})
	return color
}

func (cp *ControlPrimitive) processWelcomePushMessage() {
	cp.app.QueueUpdateDraw(func() {
		// G-Code: Modal Groups
		cp.skipQueueCommand = true
		cp.gcodeParserModalGroupsMotionDropDown.SetCurrentOption(-1)
		cp.gcodeParserModalGroupsPlaneSelectionDropDown.SetCurrentOption(-1)
		cp.gcodeParserModalGroupsDistanceModeDropDown.SetCurrentOption(-1)
		cp.gcodeParserModalGroupsFeedRateModeDropDown.SetCurrentOption(-1)
		cp.gcodeParserModalGroupsUnitsDropDown.SetCurrentOption(-1)
		cp.gcodeParserModalGroupsToolLengthOffsetInputField.SetText("")
		cp.gcodeParserModalGroupsCoordinateSystemSelectDropDown.SetCurrentOption(-1)
		cp.gcodeParserModalGroupsStoppingCheckbox.SetChecked(false)
		cp.gcodeParserModalGroupsSpindleDropDown.SetCurrentOption(-1)
		cp.gcodeParserModalGroupsCoolantMistCheckbok.SetChecked(false)
		cp.gcodeParserModalGroupsCoolantFloodCheckbok.SetChecked(false)
		// G-Code: Parser State
		cp.gcodeParserStateToolInputField.SetText("")
		cp.gcodeParserStateSpindleSpeedInputField.SetText("")
		cp.gcodeParserStateFeedRateInputField.SetText("")
		cp.skipQueueCommand = false
		// G-Code: Parameters: Coordinate System
		cp.gcodeParamsCoordinateSystemModeDropdown.SetCurrentOption(gcodeParamsCoordinateSystemModeWorkCoordinatesIdx)
		cp.gcodeParamsCoordinateSystem1XInputField.SetText("")
		cp.gcodeParamsCoordinateSystem1YInputField.SetText("")
		cp.gcodeParamsCoordinateSystem1ZInputField.SetText("")
		cp.gcodeParamsCoordinateSystem2XInputField.SetText("")
		cp.gcodeParamsCoordinateSystem2YInputField.SetText("")
		cp.gcodeParamsCoordinateSystem2ZInputField.SetText("")
		cp.gcodeParamsCoordinateSystem3XInputField.SetText("")
		cp.gcodeParamsCoordinateSystem3YInputField.SetText("")
		cp.gcodeParamsCoordinateSystem3ZInputField.SetText("")
		cp.gcodeParamsCoordinateSystem4XInputField.SetText("")
		cp.gcodeParamsCoordinateSystem4YInputField.SetText("")
		cp.gcodeParamsCoordinateSystem4ZInputField.SetText("")
		cp.gcodeParamsCoordinateSystem5XInputField.SetText("")
		cp.gcodeParamsCoordinateSystem5YInputField.SetText("")
		cp.gcodeParamsCoordinateSystem5ZInputField.SetText("")
		cp.gcodeParamsCoordinateSystem6XInputField.SetText("")
		cp.gcodeParamsCoordinateSystem6YInputField.SetText("")
		cp.gcodeParamsCoordinateSystem6ZInputField.SetText("")
		// G-Code: Parameters: Pre-Defined Position
		cp.gcodeParamsPreDefinedPosition1XInputField.SetText("")
		cp.gcodeParamsPreDefinedPosition1YInputField.SetText("")
		cp.gcodeParamsPreDefinedPosition1ZInputField.SetText("")
		cp.gcodeParamsPreDefinedPosition2XInputField.SetText("")
		cp.gcodeParamsPreDefinedPosition2YInputField.SetText("")
		cp.gcodeParamsPreDefinedPosition2ZInputField.SetText("")
		// G-Code: Parameters: Coordinate Offset
		cp.gcodeParamsCoordinateOffsetModeDropdown.SetCurrentOption(gcodeParamsCoordinateOffsetModeWorkCoordinatesIdx)
		cp.gcodeParamsCoordinateOffsetXInputField.SetText("")
		cp.gcodeParamsCoordinateOffsetYInputField.SetText("")
		cp.gcodeParamsCoordinateOffsetZInputField.SetText("")

		cp.machineCoordinates = nil
		cp.gcodeParameters = nil
		cp.modalGroup = nil
	})
	fmt.Fprintf(cp.pushMessagesTextView, "\n[%s]Soft-Reset detected[-]", tcell.ColorOrange)
	cp.sendStatusCommands()
}

func (cp *ControlPrimitive) processAlarmPushMessage(
	alarmPushMessage *grblMod.AlarmPushMessage,
) (string, tcell.Color) {
	return tview.Escape(alarmPushMessage.Error().Error()), tcell.ColorRed
}

//gocyclo:ignore
func (cp *ControlPrimitive) processPushMessage(pushMessage grblMod.PushMessage) {
	var color = tcell.ColorGreen
	var extraInfo string

	if gcodeStatePushMessage, ok := pushMessage.(*grblMod.GcodeStatePushMessage); ok {
		color = cp.processGcodeStatePushMessage(gcodeStatePushMessage)
		if cp.quietStatusComms {
			return
		}
	}

	if _, ok := pushMessage.(*grblMod.GcodeParamPushMessage); ok {
		color = cp.processGcodeParamPushMessage()

		if cp.quietStatusComms {
			return
		}
	}

	if _, ok := pushMessage.(*grblMod.WelcomePushMessage); ok {
		cp.processWelcomePushMessage()
	}

	if alarmPushMessage, ok := pushMessage.(*grblMod.AlarmPushMessage); ok {
		extraInfo, color = cp.processAlarmPushMessage(alarmPushMessage)
	}

	if statusReportPushMessage, ok := pushMessage.(*grblMod.StatusReportPushMessage); ok {
		machineCoordinates := statusReportPushMessage.GetMachineCoordinates(cp.grbl)
		if !reflect.DeepEqual(cp.machineCoordinates, machineCoordinates) {
			cp.mu.Lock()
			newMachineCoordinates := *machineCoordinates
			cp.machineCoordinates = &newMachineCoordinates
			cp.mu.Unlock()
			cp.app.QueueUpdateDraw(func() {
				cp.updateGcodeParamsCoordinateSystem()
				cp.updateGcodeParamsCoordinateOffset()
			})
		}
		if cp.quietStatusComms {
			return
		}
	}

	if _, ok := pushMessage.(*grblMod.SettingPushMessage); ok {
		if cp.quietStatusComms {
			return
		}
	}

	if _, ok := pushMessage.(*grblMod.VersionPushMessage); ok {
		if cp.quietStatusComms {
			return
		}
	}

	if _, ok := pushMessage.(*grblMod.CompileTimeOptionsPushMessage); ok {
		if cp.quietStatusComms {
			return
		}
	}

	text := pushMessage.String()
	if len(text) == 0 {
		fmt.Fprintf(cp.pushMessagesTextView, "\n[%s](%#v)[-]", color, tview.Escape(reflect.TypeOf(pushMessage).String()))
	} else {
		fmt.Fprintf(cp.pushMessagesTextView, "\n[%s]%s[-]", color, tview.Escape(text))
	}
	if len(extraInfo) > 0 {
		fmt.Fprintf(cp.pushMessagesTextView, "\n[%s]%s[-]", tcell.ColorWhite, tview.Escape(extraInfo))
	}
}

func (cp *ControlPrimitive) pushMessageWorker(
	ctx context.Context,
	pushMessageCh <-chan grblMod.PushMessage,
) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case pushMessage, ok := <-pushMessageCh:
			if !ok {
				return fmt.Errorf("push message channel closed")
			}
			cp.processPushMessage(pushMessage)
		}
	}
}

func (cp *ControlPrimitive) trackedStateWorker(
	ctx context.Context,
	trackedStateCh <-chan *TrackedState,
) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case trackedState, ok := <-trackedStateCh:
			if !ok {
				return fmt.Errorf("tracked state channel closed")
			}
			cp.app.QueueUpdateDraw(func() { cp.setState(trackedState.State) })
		}
	}
}

func (cp *ControlPrimitive) sendStatusCommandWorker(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case command := <-cp.sendStatusCommandCh:
			cp.sendCommand(ctx, &commandParameterType{
				command: command,
				timeout: defaultCommandTimeout,
				quiet:   cp.quietStatusComms,
			})
		}
	}
}

func (cp *ControlPrimitive) sendCommandWorker(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case queuedCommand := <-cp.sendCommandCh:
			err := cp.processCommand(ctx, queuedCommand.command)
			if queuedCommand.errCh != nil {
				queuedCommand.errCh <- err
			}
		}
	}
}

func (cp *ControlPrimitive) sendRealTimeCommandWorker(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case realTimeCommand := <-cp.sendRealTimeCommandCh:
			cp.sendRealTimeCommand(realTimeCommand)
		}
	}
}

func (cp *ControlPrimitive) Worker(
	ctx context.Context,
	pushMessageCh <-chan grblMod.PushMessage,
	trackedStateCh <-chan *TrackedState,
) error {
	workerManager := worker_manager.NewWorkerManager()

	workerManager.AddWorker("Control.pushMessageWorker", func(ctx context.Context) error {
		return cp.pushMessageWorker(ctx, pushMessageCh)
	})
	workerManager.AddWorker("Control.trackedStateWorker", func(ctx context.Context) error {
		return cp.trackedStateWorker(ctx, trackedStateCh)
	})
	workerManager.AddWorker("Control.sendStatusCommandWorker", cp.sendStatusCommandWorker)
	workerManager.AddWorker("Control.sendCommandWorker", cp.sendCommandWorker)
	workerManager.AddWorker("Control.sendRealTimeCommandWorker", cp.sendRealTimeCommandWorker)

	workerManager.Start(ctx)

	var err error
	for name, workerErr := range workerManager.Wait(ctx) {
		if errors.Is(workerErr, context.Canceled) {
			workerErr = nil
		}
		if workerErr != nil {
			workerErr = fmt.Errorf("%s: %w", name, workerErr)
		}
		err = errors.Join(err, workerErr)
	}
	return err
}
