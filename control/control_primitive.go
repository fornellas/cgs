package control

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"reflect"
	"regexp"
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

	gcodeParserModalGroupsMotionWords    []string
	gcodeParserModalGroupsMotionDropDown *tview.DropDown

	gcodeParserModalGroupsPlaneSelectionWords    []string
	gcodeParserModalGroupsPlaneSelectionDropDown *tview.DropDown

	gcodeParserModalGroupsDistanceModeWords    []string
	gcodeParserModalGroupsDistanceModeDropDown *tview.DropDown

	gcodeParserModalGroupsFeedRateModeWords    []string
	gcodeParserModalGroupsFeedRateModeDropDown *tview.DropDown

	gcodeParserModalGroupsUnitsWords    []string
	gcodeParserModalGroupsUnitsDropDown *tview.DropDown

	gcodeParserModalGroupsToolLengthOffsetInputField *tview.InputField

	gcodeParserModalGroupsCoordinateSystemSelectWords    []string
	gcodeParserModalGroupsCoordinateSystemSelectDropDown *tview.DropDown

	gcodeParserModalGroupsStoppingCheckbox *tview.Checkbox

	gcodeParserModalGroupsSpindleWords    []string
	gcodeParserModalGroupsSpindleDropDown *tview.DropDown

	gcodeParserModalGroupsCoolantMistCheckbok  *tview.Checkbox
	gcodeParserModalGroupsCoolantFloodCheckbok *tview.Checkbox
	gcodeParserModalGroupsCoolantOff           *tview.Button

	gcodeParserStateToolInputField         *tview.InputField
	gcodeParserStateSpindleSpeedInputField *tview.InputField
	gcodeParserStateFeedRateInputField     *tview.InputField

	gcodeParserFlex *tview.Flex

	coordinateSystemModeOptions  []string
	coordinateSystemModeDropdown *tview.DropDown

	coordinateSystem1xInputField *tview.InputField
	coordinateSystem1yInputField *tview.InputField
	coordinateSystem1zInputField *tview.InputField
	coordinateSystem1            *grblMod.Coordinates

	coordinateSystem2xInputField *tview.InputField
	coordinateSystem2yInputField *tview.InputField
	coordinateSystem2zInputField *tview.InputField
	coordinateSystem2            *grblMod.Coordinates

	coordinateSystem3xInputField *tview.InputField
	coordinateSystem3yInputField *tview.InputField
	coordinateSystem3zInputField *tview.InputField
	coordinateSystem3            *grblMod.Coordinates

	coordinateSystem4xInputField *tview.InputField
	coordinateSystem4yInputField *tview.InputField
	coordinateSystem4zInputField *tview.InputField
	coordinateSystem4            *grblMod.Coordinates

	coordinateSystem5xInputField *tview.InputField
	coordinateSystem5yInputField *tview.InputField
	coordinateSystem5zInputField *tview.InputField
	coordinateSystem5            *grblMod.Coordinates

	coordinateSystem6xInputField *tview.InputField
	coordinateSystem6yInputField *tview.InputField
	coordinateSystem6zInputField *tview.InputField
	coordinateSystem6            *grblMod.Coordinates

	gcodeParamsLegacyTextView *tview.TextView

	gcodeParamsScrollContainer *ScrollContainer

	commandsTextView *tview.TextView

	pushMessagesTextView *tview.TextView

	commandInputField      *tview.InputField
	commandInputHistory    []string
	commandInputHistoryIdx int

	disableCommandInput bool

	state grblMod.State

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
	cp.gcodeParserModalGroupsMotionWords = []string{"G0", "G1", "G2", "G3", "G38.2", "G38.3", "G38.4", "G38.5", "G80"}
	cp.gcodeParserModalGroupsMotionDropDown = newModalGroupDropDown(
		"Motion", cp.gcodeParserModalGroupsMotionWords,
	)

	// Plane selection
	cp.gcodeParserModalGroupsPlaneSelectionWords = []string{"G17", "G18", "G19"}
	cp.gcodeParserModalGroupsPlaneSelectionDropDown = newModalGroupDropDown(
		"Plane Selection", cp.gcodeParserModalGroupsPlaneSelectionWords,
	)

	// Distance Mode
	cp.gcodeParserModalGroupsDistanceModeWords = []string{"G90", "G91"}
	cp.gcodeParserModalGroupsDistanceModeDropDown = newModalGroupDropDown(
		"Distance Mode", cp.gcodeParserModalGroupsDistanceModeWords,
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
	cp.gcodeParserModalGroupsFeedRateModeWords = []string{"G93", "G94"}
	cp.gcodeParserModalGroupsFeedRateModeDropDown = newModalGroupDropDown(
		"Feed Rate Mode", cp.gcodeParserModalGroupsFeedRateModeWords,
	)

	// Units
	cp.gcodeParserModalGroupsUnitsWords = []string{"G20", "G21"}
	cp.gcodeParserModalGroupsUnitsDropDown = newModalGroupDropDown(
		"Units", cp.gcodeParserModalGroupsUnitsWords,
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
	cp.gcodeParserModalGroupsToolLengthOffsetInputField.SetAcceptanceFunc(acceptUFloat)
	cp.gcodeParserModalGroupsToolLengthOffsetInputField.SetFieldWidth(coordinateWidth)
	cp.gcodeParserModalGroupsToolLengthOffsetInputField.SetDoneFunc(func(key tcell.Key) {
		if cp.skipQueueCommand {
			return
		}
		cp.QueueCommand(fmt.Sprintf("G43.1 Z%s", cp.gcodeParserModalGroupsToolLengthOffsetInputField.GetText()))
	})

	// Coordinate System Select
	cp.gcodeParserModalGroupsCoordinateSystemSelectWords = []string{"G54", "G55", "G56", "G57", "G58", "G59"}
	cp.gcodeParserModalGroupsCoordinateSystemSelectDropDown = newModalGroupDropDown(
		"Coordinate System Select", cp.gcodeParserModalGroupsCoordinateSystemSelectWords,
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
	cp.gcodeParserModalGroupsSpindleWords = []string{"M3", "M4", "M5"}
	cp.gcodeParserModalGroupsSpindleDropDown = newModalGroupDropDown(
		"Spindle", cp.gcodeParserModalGroupsSpindleWords,
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

func (cp *ControlPrimitive) updateGcodeParams() {
	cp.mu.Lock()
	defer cp.mu.Unlock()
	if cp.coordinateSystem1 != nil {
		cp.coordinateSystem1xInputField.SetText(iFmt.SprintFloat(cp.coordinateSystem1.X, 4))
		cp.coordinateSystem1yInputField.SetText(iFmt.SprintFloat(cp.coordinateSystem1.Y, 4))
		cp.coordinateSystem1zInputField.SetText(iFmt.SprintFloat(cp.coordinateSystem1.Z, 4))
	} else {
		cp.coordinateSystem1xInputField.SetText("")
		cp.coordinateSystem1yInputField.SetText("")
		cp.coordinateSystem1zInputField.SetText("")
	}
	if cp.coordinateSystem2 != nil {
		cp.coordinateSystem2xInputField.SetText(iFmt.SprintFloat(cp.coordinateSystem2.X, 4))
		cp.coordinateSystem2yInputField.SetText(iFmt.SprintFloat(cp.coordinateSystem2.Y, 4))
		cp.coordinateSystem2zInputField.SetText(iFmt.SprintFloat(cp.coordinateSystem2.Z, 4))
	} else {
		cp.coordinateSystem2xInputField.SetText("")
		cp.coordinateSystem2yInputField.SetText("")
		cp.coordinateSystem2zInputField.SetText("")
	}
	if cp.coordinateSystem3 != nil {
		cp.coordinateSystem3xInputField.SetText(iFmt.SprintFloat(cp.coordinateSystem3.X, 4))
		cp.coordinateSystem3yInputField.SetText(iFmt.SprintFloat(cp.coordinateSystem3.Y, 4))
		cp.coordinateSystem3zInputField.SetText(iFmt.SprintFloat(cp.coordinateSystem3.Z, 4))
	} else {
		cp.coordinateSystem3xInputField.SetText("")
		cp.coordinateSystem3yInputField.SetText("")
		cp.coordinateSystem3zInputField.SetText("")
	}
	if cp.coordinateSystem4 != nil {
		cp.coordinateSystem4xInputField.SetText(iFmt.SprintFloat(cp.coordinateSystem4.X, 4))
		cp.coordinateSystem4yInputField.SetText(iFmt.SprintFloat(cp.coordinateSystem4.Y, 4))
		cp.coordinateSystem4zInputField.SetText(iFmt.SprintFloat(cp.coordinateSystem4.Z, 4))
	} else {
		cp.coordinateSystem4xInputField.SetText("")
		cp.coordinateSystem4yInputField.SetText("")
		cp.coordinateSystem4zInputField.SetText("")
	}
	if cp.coordinateSystem5 != nil {
		cp.coordinateSystem5xInputField.SetText(iFmt.SprintFloat(cp.coordinateSystem5.X, 4))
		cp.coordinateSystem5yInputField.SetText(iFmt.SprintFloat(cp.coordinateSystem5.Y, 4))
		cp.coordinateSystem5zInputField.SetText(iFmt.SprintFloat(cp.coordinateSystem5.Z, 4))
	} else {
		cp.coordinateSystem5xInputField.SetText("")
		cp.coordinateSystem5yInputField.SetText("")
		cp.coordinateSystem5zInputField.SetText("")
	}
	if cp.coordinateSystem6 != nil {
		cp.coordinateSystem6xInputField.SetText(iFmt.SprintFloat(cp.coordinateSystem6.X, 4))
		cp.coordinateSystem6yInputField.SetText(iFmt.SprintFloat(cp.coordinateSystem6.Y, 4))
		cp.coordinateSystem6zInputField.SetText(iFmt.SprintFloat(cp.coordinateSystem6.Z, 4))
	} else {
		cp.coordinateSystem6xInputField.SetText("")
		cp.coordinateSystem6yInputField.SetText("")
		cp.coordinateSystem6zInputField.SetText("")
	}
}

func (cp *ControlPrimitive) newGcodeParams() {
	// Coordinate system
	// select
	//   G54-G59
	// set
	//   G10 L2 P(1-9) - Set offset(s) to a value. Current position irrelevant (see G10 L2 for details).
	//   G10 L20 P(1-9) - Set offset(s) so current position becomes a value (see G10 L20 for details).
	// Interface Idea
	// Coordinate System [Offset|Value]
	// 1(G54): X:  Y:  Z:
	// 2(G55): X:  Y:  Z:
	coordinateSystemTextView := tview.NewTextView()
	coordinateSystemTextView.SetText("Coordinate System")

	cp.coordinateSystemModeDropdown = tview.NewDropDown()
	cp.coordinateSystemModeDropdown.SetLabel("Mode:")
	cp.coordinateSystemModeOptions = []string{
		fmt.Sprintf("Offset%s", sprintGcodeWord("G10L2")),
		fmt.Sprintf("Value%s", sprintGcodeWord("G10L20")),
	}
	cp.coordinateSystemModeDropdown.SetOptions(cp.coordinateSystemModeOptions, nil)
	cp.coordinateSystemModeDropdown.SetCurrentOption(0)
	cp.coordinateSystemModeDropdown.SetSelectedFunc(func(string, int) {
		// TODO update coordinate input fields
	})
	newCoordinatesInputFields := func(number, word string) (*tview.InputField, *tview.InputField, *tview.InputField, *tview.Flex) {
		labelTextView := tview.NewTextView()
		labelTextView.SetLabel(fmt.Sprintf("%s%s:", number, sprintGcodeWord(word)))

		x := tview.NewInputField()
		x.SetLabel("X:")
		// x.SetFieldWidth(coordinateWidth)
		y := tview.NewInputField()
		y.SetLabel("Y:")
		// y.SetFieldWidth(coordinateWidth)
		z := tview.NewInputField()
		z.SetLabel("Z:")
		// z.SetFieldWidth(coordinateWidth)
		changedFunc := func() {
			if cp.skipQueueCommand {
				return
			}
			// TODO
		}
		x.SetChangedFunc(func(string) { changedFunc() })
		y.SetChangedFunc(func(string) { changedFunc() })
		z.SetChangedFunc(func(string) { changedFunc() })

		flex := tview.NewFlex()
		flex.SetDirection(tview.FlexColumn)
		flex.AddItem(labelTextView, 6, 0, false)
		flex.AddItem(x, 0, 1, false)
		flex.AddItem(y, 0, 1, false)
		flex.AddItem(z, 0, 1, false)

		return x, y, z, flex
	}
	var coordinateSystem1flex, coordinateSystem2flex, coordinateSystem3flex, coordinateSystem4flex, coordinateSystem5flex, coordinateSystem6flex *tview.Flex
	cp.coordinateSystem1xInputField, cp.coordinateSystem1yInputField, cp.coordinateSystem1zInputField, coordinateSystem1flex = newCoordinatesInputFields("1", "G54")
	cp.coordinateSystem2xInputField, cp.coordinateSystem2yInputField, cp.coordinateSystem2zInputField, coordinateSystem2flex = newCoordinatesInputFields("2", "G55")
	cp.coordinateSystem3xInputField, cp.coordinateSystem3yInputField, cp.coordinateSystem3zInputField, coordinateSystem3flex = newCoordinatesInputFields("3", "G56")
	cp.coordinateSystem4xInputField, cp.coordinateSystem4yInputField, cp.coordinateSystem4zInputField, coordinateSystem4flex = newCoordinatesInputFields("4", "G57")
	cp.coordinateSystem5xInputField, cp.coordinateSystem5yInputField, cp.coordinateSystem5zInputField, coordinateSystem5flex = newCoordinatesInputFields("5", "G58")
	cp.coordinateSystem6xInputField, cp.coordinateSystem6yInputField, cp.coordinateSystem6zInputField, coordinateSystem6flex = newCoordinatesInputFields("6", "G59")

	// LEGACY
	cp.gcodeParamsLegacyTextView = tview.NewTextView()
	cp.gcodeParamsLegacyTextView.SetDynamicColors(true)
	cp.gcodeParamsLegacyTextView.SetScrollable(true)
	cp.gcodeParamsLegacyTextView.SetWrap(true)
	cp.gcodeParamsLegacyTextView.SetBorder(true).SetTitle("LEGACY")
	cp.gcodeParamsLegacyTextView.SetChangedFunc(func() {
		cp.app.QueueUpdateDraw(func() {
			text := cp.gcodeParamsLegacyTextView.GetText(false)
			if len(text) > 0 && text[len(text)-1] == '\n' {
				cp.gcodeParamsLegacyTextView.SetText(text[:len(text)-1])
			}
		})
	})

	// G-Code: Parameters
	cp.gcodeParamsScrollContainer = NewScrollContainer()
	cp.gcodeParamsScrollContainer.SetBorder(true)
	cp.gcodeParamsScrollContainer.SetTitle("G-Code: Parameters")
	cp.gcodeParamsScrollContainer.AddPrimitive(coordinateSystemTextView, 1)
	cp.gcodeParamsScrollContainer.AddPrimitive(cp.coordinateSystemModeDropdown, 1)
	cp.gcodeParamsScrollContainer.AddPrimitive(coordinateSystem1flex, 1)
	cp.gcodeParamsScrollContainer.AddPrimitive(coordinateSystem2flex, 1)
	cp.gcodeParamsScrollContainer.AddPrimitive(coordinateSystem3flex, 1)
	cp.gcodeParamsScrollContainer.AddPrimitive(coordinateSystem4flex, 1)
	cp.gcodeParamsScrollContainer.AddPrimitive(coordinateSystem5flex, 1)
	cp.gcodeParamsScrollContainer.AddPrimitive(coordinateSystem6flex, 1)
	cp.gcodeParamsScrollContainer.AddPrimitive(cp.gcodeParamsLegacyTextView, 16)
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
	// G-Code: Parameters
	cp.coordinateSystemModeDropdown.SetDisabled(disabled)
	cp.coordinateSystem1xInputField.SetDisabled(disabled)
	cp.coordinateSystem1yInputField.SetDisabled(disabled)
	cp.coordinateSystem1zInputField.SetDisabled(disabled)
	cp.coordinateSystem2xInputField.SetDisabled(disabled)
	cp.coordinateSystem2yInputField.SetDisabled(disabled)
	cp.coordinateSystem2zInputField.SetDisabled(disabled)
	cp.coordinateSystem3xInputField.SetDisabled(disabled)
	cp.coordinateSystem3yInputField.SetDisabled(disabled)
	cp.coordinateSystem3zInputField.SetDisabled(disabled)
	cp.coordinateSystem4xInputField.SetDisabled(disabled)
	cp.coordinateSystem4yInputField.SetDisabled(disabled)
	cp.coordinateSystem4zInputField.SetDisabled(disabled)
	cp.coordinateSystem5xInputField.SetDisabled(disabled)
	cp.coordinateSystem5yInputField.SetDisabled(disabled)
	cp.coordinateSystem5zInputField.SetDisabled(disabled)
	cp.coordinateSystem6xInputField.SetDisabled(disabled)
	cp.coordinateSystem6yInputField.SetDisabled(disabled)
	cp.coordinateSystem6zInputField.SetDisabled(disabled)
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
					cp.gcodeParserModalGroupsMotionWords,
					cp.gcodeParserModalGroupsMotionDropDown,
				)
			}
			if modalGroup.PlaneSelection != nil {
				setDropDownFn(
					modalGroup.PlaneSelection,
					cp.gcodeParserModalGroupsPlaneSelectionWords,
					cp.gcodeParserModalGroupsPlaneSelectionDropDown,
				)
			}
			if modalGroup.DistanceMode != nil {
				setDropDownFn(
					modalGroup.DistanceMode,
					cp.gcodeParserModalGroupsDistanceModeWords,
					cp.gcodeParserModalGroupsDistanceModeDropDown,
				)
			}
			if modalGroup.FeedRateMode != nil {
				setDropDownFn(
					modalGroup.FeedRateMode,
					cp.gcodeParserModalGroupsFeedRateModeWords,
					cp.gcodeParserModalGroupsFeedRateModeDropDown,
				)
			}
			if modalGroup.Units != nil {
				setDropDownFn(
					modalGroup.Units,
					cp.gcodeParserModalGroupsUnitsWords,
					cp.gcodeParserModalGroupsUnitsDropDown,
				)
			}
			if modalGroup.CoordinateSystemSelect != nil {
				setDropDownFn(
					modalGroup.CoordinateSystemSelect,
					cp.gcodeParserModalGroupsCoordinateSystemSelectWords,
					cp.gcodeParserModalGroupsCoordinateSystemSelectDropDown,
				)
			}
			if modalGroup.Spindle != nil {
				setDropDownFn(
					modalGroup.Spindle,
					cp.gcodeParserModalGroupsSpindleWords,
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
	})

	return tcell.ColorGreen
}

//gocyclo:ignore
func (cp *ControlPrimitive) processGcodeParamPushMessage() tcell.Color {
	color := tcell.ColorGreen

	params := cp.grbl.GetLastGcodeParameters()
	if params == nil {
		return color
	}

	// G-Code: Parameters
	cp.mu.Lock()
	cp.coordinateSystem1 = params.CoordinateSystem1
	cp.coordinateSystem2 = params.CoordinateSystem2
	cp.coordinateSystem3 = params.CoordinateSystem3
	cp.coordinateSystem4 = params.CoordinateSystem4
	cp.coordinateSystem5 = params.CoordinateSystem5
	cp.coordinateSystem6 = params.CoordinateSystem6
	cp.mu.Unlock()
	cp.app.QueueUpdateDraw(func() {
		cp.updateGcodeParams()
	})

	// LEGACY
	var buf bytes.Buffer
	if params.HasCoordinateSystem() {
		fmt.Fprint(&buf, "Coordinate System\n")
		if params.CoordinateSystem1 != nil {
			fmt.Fprintf(&buf, "  %s:", sprintGcodeWord("G54"))
			fmt.Fprintf(&buf, "%s\n", sprintCoordinatesSingleLine(params.CoordinateSystem1, " "))
		}
		if params.CoordinateSystem2 != nil {
			fmt.Fprintf(&buf, "  %s:", sprintGcodeWord("G55"))
			fmt.Fprintf(&buf, "%s\n", sprintCoordinatesSingleLine(params.CoordinateSystem2, " "))
		}
		if params.CoordinateSystem3 != nil {
			fmt.Fprintf(&buf, "  %s:", sprintGcodeWord("G56"))
			fmt.Fprintf(&buf, "%s\n", sprintCoordinatesSingleLine(params.CoordinateSystem3, " "))
		}
		if params.CoordinateSystem4 != nil {
			fmt.Fprintf(&buf, "  %s:", sprintGcodeWord("G57"))
			fmt.Fprintf(&buf, "%s\n", sprintCoordinatesSingleLine(params.CoordinateSystem4, " "))
		}
		if params.CoordinateSystem5 != nil {
			fmt.Fprintf(&buf, "  %s:", sprintGcodeWord("G58"))
			fmt.Fprintf(&buf, "%s\n", sprintCoordinatesSingleLine(params.CoordinateSystem5, " "))
		}
		if params.CoordinateSystem6 != nil {
			fmt.Fprintf(&buf, "  %s:", sprintGcodeWord("G59"))
			fmt.Fprintf(&buf, "%s\n", sprintCoordinatesSingleLine(params.CoordinateSystem6, " "))
		}
	}
	if params.HasPreDefinedPosition() {
		fmt.Fprintf(&buf, "Pre-Defined Position\n")
		if params.PrimaryPreDefinedPosition != nil {
			fmt.Fprintf(&buf, "  %s:%s\n", sprintGcodeWord("G28"), sprintCoordinatesSingleLine(params.PrimaryPreDefinedPosition, " "))
		}
		if params.SecondaryPreDefinedPosition != nil {
			fmt.Fprintf(&buf, "  %s:%s\n", sprintGcodeWord("G30"), sprintCoordinatesSingleLine(params.SecondaryPreDefinedPosition, " "))
		}
	}
	if params.CoordinateOffset != nil {
		fmt.Fprintf(&buf, "Coordinate Offset\n")
		fmt.Fprintf(&buf, "  %s:%s\n", sprintGcodeWord("G92"), sprintCoordinatesSingleLine(params.CoordinateOffset, " "))
	}
	if params.ToolLengthOffset != nil {
		cp.app.QueueUpdateDraw(func() {
			cp.gcodeParserModalGroupsToolLengthOffsetInputField.SetText(iFmt.SprintFloat(*params.ToolLengthOffset, 4))
		})
	}
	if params.Probe != nil {
		// TODO move to Probe tab
		fmt.Fprintf(&buf, "Last Probing Cycle\n")
		if params.Probe.Successful {
			fmt.Fprintf(&buf, "  %s\n", sprintCoordinatesSingleLine(&params.Probe.Coordinates, " "))
		} else {
			fmt.Fprintf(&buf, "  [%s]Failed[-]\n", tcell.ColorRed)
		}
	}
	cp.app.QueueUpdateDraw(func() {
		if buf.String() == cp.gcodeParamsLegacyTextView.GetText(false) {
			return
		}
		cp.gcodeParamsLegacyTextView.SetText(buf.String())
	})

	return color
}

func (cp *ControlPrimitive) processWelcomePushMessage() {
	cp.app.QueueUpdateDraw(func() {
		// G-Code Parameters
		cp.gcodeParamsLegacyTextView.Clear()
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
		//G-Code: Parameters
		cp.coordinateSystemModeDropdown.SetCurrentOption(0)
		cp.coordinateSystem1xInputField.SetText("")
		cp.coordinateSystem1yInputField.SetText("")
		cp.coordinateSystem1zInputField.SetText("")
		cp.coordinateSystem1 = nil
		cp.coordinateSystem2xInputField.SetText("")
		cp.coordinateSystem2yInputField.SetText("")
		cp.coordinateSystem2zInputField.SetText("")
		cp.coordinateSystem2 = nil
		cp.coordinateSystem3xInputField.SetText("")
		cp.coordinateSystem3yInputField.SetText("")
		cp.coordinateSystem3zInputField.SetText("")
		cp.coordinateSystem3 = nil
		cp.coordinateSystem4xInputField.SetText("")
		cp.coordinateSystem4yInputField.SetText("")
		cp.coordinateSystem4zInputField.SetText("")
		cp.coordinateSystem4 = nil
		cp.coordinateSystem5xInputField.SetText("")
		cp.coordinateSystem5yInputField.SetText("")
		cp.coordinateSystem5zInputField.SetText("")
		cp.coordinateSystem5 = nil
		cp.coordinateSystem6xInputField.SetText("")
		cp.coordinateSystem6yInputField.SetText("")
		cp.coordinateSystem6zInputField.SetText("")
		cp.coordinateSystem6 = nil
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

	if _, ok := pushMessage.(*grblMod.StatusReportPushMessage); ok {
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
