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

	gcodeParserTextView *tview.TextView
	gcodeParserFlex     *tview.Flex

	gcodeParamsTextView *tview.TextView

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

	cp.newGcodeParserFlex()
	cp.newGcodeParamsTextView()
	cp.newCommandsTextView()
	cp.newPushMessagesTextView()
	cp.newCommandInputField()

	cp.newControlFlex()

	cp.setDisabledState()
	cp.sendStatusCommands()

	return cp
}

func (cp *ControlPrimitive) newGcodeParserFlex() {
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
	gcodeParserModalGroupsArcIjkDistanceModeDropDown.SetCurrentOption(0)
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
	gcodeParserModalGroupsCutterDiameterCompensationDropDown.SetCurrentOption(0)
	gcodeParserModalGroupsCutterDiameterCompensationDropDown.SetDisabled(true)

	// Tool Length Offset
	cp.gcodeParserModalGroupsToolLengthOffsetInputField = tview.NewInputField()
	cp.gcodeParserModalGroupsToolLengthOffsetInputField.SetLabel(fmt.Sprintf("Tool Length Offset%s:", sprintGcodeWord("G43.1")))
	cp.gcodeParserModalGroupsToolLengthOffsetInputField.SetAcceptanceFunc(acceptUFloat)
	cp.gcodeParserModalGroupsToolLengthOffsetInputField.SetChangedFunc(func(text string) {
		if cp.skipQueueCommand {
			return
		}
		z, err := strconv.ParseFloat(text, 64)
		if err != nil {
			panic(fmt.Sprintf("bug: SetAcceptanceFunc should have prevented bad floats: %#v", text))
		}
		cp.QueueCommand(fmt.Sprintf("G43.1 Z%s", sprintCoordinate(z)))
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
	gcodeParserModalGroupsControlModeDropDown.SetCurrentOption(0)
	gcodeParserModalGroupsControlModeDropDown.SetDisabled(true)

	// Stopping
	cp.gcodeParserModalGroupsStoppingCheckbox = tview.NewCheckbox()
	cp.gcodeParserModalGroupsStoppingCheckbox.SetLabel(fmt.Sprintf("%s%s", tview.Escape(gcode.WordName("M0")), sprintGcodeWord("M0")))
	cp.gcodeParserModalGroupsStoppingCheckbox.SetChangedFunc(func(checked bool) {
		if cp.skipQueueCommand {
			return
		}
		// TODO check state
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
	cp.gcodeParserModalGroupsCoolantMistCheckbok = tview.NewCheckbox()
	cp.gcodeParserModalGroupsCoolantMistCheckbok.SetLabel("Mist")
	cp.gcodeParserModalGroupsCoolantMistCheckbok.SetChangedFunc(func(checked bool) {
		if cp.skipQueueCommand {
			return
		}
		cp.QueueCommand("M7")
	})
	cp.gcodeParserModalGroupsCoolantFloodCheckbok = tview.NewCheckbox()
	cp.gcodeParserModalGroupsCoolantFloodCheckbok.SetLabel("Flood")
	cp.gcodeParserModalGroupsCoolantFloodCheckbok.SetChangedFunc(func(checked bool) {
		if cp.skipQueueCommand {
			return
		}
		cp.QueueCommand("M8")
	})
	cp.gcodeParserModalGroupsCoolantOff = tview.NewButton("Off")
	cp.gcodeParserModalGroupsCoolantOff.SetSelectedFunc(func() {
		cp.QueueCommand("M9")
	})

	// Modal Groups Flex
	gcodeParserModalGroupsFlex := NewScrollContainer()
	gcodeParserModalGroupsFlex.SetBorder(true)
	gcodeParserModalGroupsFlex.SetTitle("Modal Groups")
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
	// TODO add flex for Coolant all in the same line

	// TextView
	gcodeParserTextView := tview.NewTextView()
	gcodeParserTextView.SetDynamicColors(true)
	gcodeParserTextView.SetScrollable(true)
	gcodeParserTextView.SetWrap(true)
	gcodeParserTextView.SetBorder(true)
	gcodeParserTextView.SetTitle("LEGACY")
	gcodeParserTextView.SetChangedFunc(func() {
		cp.app.QueueUpdateDraw(func() {
			text := gcodeParserTextView.GetText(false)
			if len(text) > 0 && text[len(text)-1] == '\n' {
				gcodeParserTextView.SetText(text[:len(text)-1])
			}
		})
	})
	cp.gcodeParserTextView = gcodeParserTextView

	// Flex
	gcodeParserFlex := tview.NewFlex()
	gcodeParserFlex.SetBorder(true)
	gcodeParserFlex.SetTitle("G-Code Parser")
	gcodeParserFlex.SetDirection(tview.FlexRow)
	gcodeParserFlex.AddItem(gcodeParserModalGroupsFlex, 0, 1, false)
	gcodeParserFlex.AddItem(gcodeParserTextView, 0, 1, false)

	cp.gcodeParserFlex = gcodeParserFlex
}

func (cp *ControlPrimitive) newGcodeParamsTextView() {
	gcodeParamsTextView := tview.NewTextView()
	gcodeParamsTextView.SetDynamicColors(true)
	gcodeParamsTextView.SetScrollable(true)
	gcodeParamsTextView.SetWrap(true)
	gcodeParamsTextView.SetBorder(true).SetTitle("G-Code Parameters")
	gcodeParamsTextView.SetChangedFunc(func() {
		cp.app.QueueUpdateDraw(func() {
			text := gcodeParamsTextView.GetText(false)
			if len(text) > 0 && text[len(text)-1] == '\n' {
				gcodeParamsTextView.SetText(text[:len(text)-1])
			}
		})
	})
	cp.gcodeParamsTextView = gcodeParamsTextView
}

func (cp *ControlPrimitive) newCommandsTextView() {
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

func (cp *ControlPrimitive) newPushMessagesTextView() {
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

func (cp *ControlPrimitive) newCommandInputField() {
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

func (cp *ControlPrimitive) newControlFlex() {
	gcodeFlex := tview.NewFlex()
	gcodeFlex.SetDirection(tview.FlexColumn)
	gcodeFlex.AddItem(cp.gcodeParserFlex, 0, 1, false)
	gcodeFlex.AddItem(cp.gcodeParamsTextView, 0, 1, false)

	commsFlex := tview.NewFlex()
	commsFlex.SetDirection(tview.FlexColumn)
	commsFlex.AddItem(cp.commandsTextView, 0, 1, false)
	commsFlex.AddItem(cp.pushMessagesTextView, 0, 1, false)

	controlFlex := tview.NewFlex()
	controlFlex.SetBorder(true)
	controlFlex.SetTitle("Contrtol")
	controlFlex.SetDirection(tview.FlexRow)
	controlFlex.AddItem(gcodeFlex, 0, 1, false)
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

	cp.commandInputField.SetDisabled(disabled)
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
}

func (cp *ControlPrimitive) setState(state grblMod.State) {
	cp.mu.Lock()
	cp.state = state
	cp.mu.Unlock()
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
	var buf bytes.Buffer

	cp.skipQueueCommand = true
	defer func() { cp.skipQueueCommand = false }()

	if modalGroup := gcodeStatePushMessage.ModalGroup; modalGroup != nil {
		if modalGroup.Motion != nil {
			index := -1
			for i, word := range cp.gcodeParserModalGroupsMotionWords {
				if word == modalGroup.Motion.NormalizedString() {
					index = i
				}
			}
			cp.gcodeParserModalGroupsMotionDropDown.SetCurrentOption(index)
		}
		if modalGroup.PlaneSelection != nil {
			index := -1
			for i, word := range cp.gcodeParserModalGroupsPlaneSelectionWords {
				if word == modalGroup.PlaneSelection.NormalizedString() {
					index = i
				}
			}
			cp.gcodeParserModalGroupsPlaneSelectionDropDown.SetCurrentOption(index)
		}
		if modalGroup.DistanceMode != nil {
			index := -1
			for i, word := range cp.gcodeParserModalGroupsDistanceModeWords {
				if word == modalGroup.DistanceMode.NormalizedString() {
					index = i
				}
			}
			cp.gcodeParserModalGroupsDistanceModeDropDown.SetCurrentOption(index)
		}
		if modalGroup.FeedRateMode != nil {
			index := -1
			for i, word := range cp.gcodeParserModalGroupsFeedRateModeWords {
				if word == modalGroup.FeedRateMode.NormalizedString() {
					index = i
				}
			}
			cp.gcodeParserModalGroupsFeedRateModeDropDown.SetCurrentOption(index)
		}
		if modalGroup.Units != nil {
			index := -1
			for i, word := range cp.gcodeParserModalGroupsUnitsWords {
				if word == modalGroup.Units.NormalizedString() {
					index = i
				}
			}
			cp.gcodeParserModalGroupsUnitsDropDown.SetCurrentOption(index)
		}
		if modalGroup.ToolLengthOffset != nil {
			// FIXME not reported here, needs to come from $#
		}
		if modalGroup.CoordinateSystemSelect != nil {
			index := -1
			for i, word := range cp.gcodeParserModalGroupsCoordinateSystemSelectWords {
				if word == modalGroup.CoordinateSystemSelect.NormalizedString() {
					index = i
				}
			}
			cp.gcodeParserModalGroupsCoordinateSystemSelectDropDown.SetCurrentOption(index)
		}
		if modalGroup.Stopping != nil {
			// FIXME never reported back, need to manually track sent gcodes
		}
		if modalGroup.Spindle != nil {
			index := -1
			for i, word := range cp.gcodeParserModalGroupsSpindleWords {
				if word == modalGroup.Spindle.NormalizedString() {
					index = i
				}
			}
			cp.gcodeParserModalGroupsSpindleDropDown.SetCurrentOption(index)
		}
		// for _, word := range modalGroup.Coolant {
		// TODO
		// cp.gcodeParserModalGroupsCoolantMistCheckbok
		// cp.gcodeParserModalGroupsCoolantFloodCheckbok
		// cp.gcodeParserModalGroupsCoolantOff
		// }
	}

	if gcodeStatePushMessage.Tool != nil {
		fmt.Fprintf(&buf, "Tool: %s\n", sprintTool(*gcodeStatePushMessage.Tool))
	}
	if gcodeStatePushMessage.FeedRate != nil && *gcodeStatePushMessage.FeedRate != 0 {
		fmt.Fprintf(&buf, "Feed Rate: %s\n", sprintFeed(*gcodeStatePushMessage.FeedRate))
	}
	if gcodeStatePushMessage.SpindleSpeed != nil && *gcodeStatePushMessage.SpindleSpeed != 0 {
		fmt.Fprintf(&buf, "Speed: %s\n", sprintSpeed(*gcodeStatePushMessage.SpindleSpeed))
	}

	cp.app.QueueUpdateDraw(func() {
		if buf.String() == cp.gcodeParserTextView.GetText(false) {
			return
		}
		cp.gcodeParserTextView.SetText(buf.String())
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
		fmt.Fprintf(&buf, "Tool Length Offset:%s\n", sprintCoordinate(*params.ToolLengthOffset))
	}
	if params.Probe != nil {
		fmt.Fprintf(&buf, "Last Probing Cycle\n")
		if params.Probe.Successful {
			fmt.Fprintf(&buf, "  %s\n", sprintCoordinatesSingleLine(&params.Probe.Coordinates, " "))
		} else {
			fmt.Fprintf(&buf, "  [%s]Failed[-]\n", tcell.ColorRed)
		}
	}

	cp.app.QueueUpdateDraw(func() {
		if buf.String() == cp.gcodeParamsTextView.GetText(false) {
			return
		}
		cp.gcodeParamsTextView.SetText(buf.String())
	})

	return color
}

func (cp *ControlPrimitive) processWelcomePushMessage() {
	cp.app.QueueUpdateDraw(func() {
		cp.gcodeParserTextView.Clear()
		cp.gcodeParamsTextView.Clear()
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
		cp.skipQueueCommand = false
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
