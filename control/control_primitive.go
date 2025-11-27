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

	"github.com/fornellas/cgs/gcode"
	grblMod "github.com/fornellas/cgs/grbl"
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

type ControlPrimitive struct {
	*tview.Flex
	grbl                   *grblMod.Grbl
	pushMessageCh          chan grblMod.PushMessage
	app                    *tview.Application
	statusPrimitive        *StatusPrimitive
	quietStatusComms       bool
	sendStatusCommandCh    chan string
	sendCommandCh          chan string
	sendRealTimeCommandCh  chan grblMod.RealTimeCommand
	commandsTextView       *tview.TextView
	pushMessagesTextView   *tview.TextView
	gcodeParserTextView    *tview.TextView
	gcodeParamsTextView    *tview.TextView
	commandInputField      *tview.InputField
	commandInputHistory    []string
	commandInputHistoryIdx int
	mu                     sync.Mutex
	disableCommandInput    bool
	machineState           *string
}

//gocyclo:ignore
func NewControlPrimitive(
	ctx context.Context,
	grbl *grblMod.Grbl,
	pushMessageCh chan grblMod.PushMessage,
	app *tview.Application,
	statusPrimitive *StatusPrimitive,
	quietStatusComms bool,
) *ControlPrimitive {
	cp := &ControlPrimitive{
		grbl:                   grbl,
		pushMessageCh:          pushMessageCh,
		app:                    app,
		statusPrimitive:        statusPrimitive,
		quietStatusComms:       quietStatusComms,
		sendStatusCommandCh:    make(chan string, 10),
		sendCommandCh:          make(chan string, 10),
		sendRealTimeCommandCh:  make(chan grblMod.RealTimeCommand, 10),
		commandInputHistoryIdx: -1,
	}

	// Commands
	commandsTextView := tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true).
		SetWrap(true)
	commandsTextView.SetBorder(true).SetTitle("Commands")
	commandsTextView.SetChangedFunc(func() {
		cp.app.QueueUpdate(func() {
			text := commandsTextView.GetText(false)
			if len(text) > 0 && text[0] == '\n' {
				commandsTextView.SetText(text[1:])
			}
			commandsTextView.ScrollToEnd()
		})
	})
	cp.commandsTextView = commandsTextView

	// Push Messages
	pushMessagesTextView := tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true).
		SetWrap(true)
	pushMessagesTextView.SetBorder(true).SetTitle("Push Messages")
	pushMessagesTextView.SetChangedFunc(func() {
		cp.app.QueueUpdate(func() {
			text := pushMessagesTextView.GetText(false)
			if len(text) > 0 && text[0] == '\n' {
				pushMessagesTextView.SetText(text[1:])
			}
			pushMessagesTextView.ScrollToEnd()
		})
	})
	cp.pushMessagesTextView = pushMessagesTextView

	// G-Code Parser
	gcodeParserTextView := tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true).
		SetWrap(true)
	gcodeParserTextView.SetBorder(true).SetTitle("G-Code Parser")
	gcodeParserTextView.SetChangedFunc(func() {
		cp.app.QueueUpdate(func() {
			text := gcodeParserTextView.GetText(false)
			if len(text) > 0 && text[len(text)-1] == '\n' {
				gcodeParserTextView.SetText(text[:len(text)-1])
			}
		})
	})
	cp.gcodeParserTextView = gcodeParserTextView

	// G-Code Parameters
	gcodeParamsTextView := tview.NewTextView().
		SetDynamicColors(true).
		SetScrollable(true).
		SetWrap(true)
	gcodeParamsTextView.SetBorder(true).SetTitle("G-Code Parameters")
	gcodeParamsTextView.SetChangedFunc(func() {
		cp.app.QueueUpdate(func() {
			text := gcodeParamsTextView.GetText(false)
			if len(text) > 0 && text[len(text)-1] == '\n' {
				gcodeParamsTextView.SetText(text[:len(text)-1])
			}
		})
	})
	cp.gcodeParamsTextView = gcodeParamsTextView

	// Command
	commandInputField := tview.NewInputField()
	commandInputField.SetLabel("Command")
	commandInputField.SetDoneFunc(func(key tcell.Key) {
		switch key {
		case tcell.KeyEscape:
			commandInputField.SetText("")
		case tcell.KeyEnter:
			command := commandInputField.GetText()
			if command == "" {
				return
			}
			cp.QueueCommand(command)
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

	gcodeFlex := tview.NewFlex()
	gcodeFlex.SetDirection(tview.FlexColumn)
	gcodeFlex.AddItem(gcodeParserTextView, 0, 1, false)
	gcodeFlex.AddItem(gcodeParamsTextView, 0, 1, false)

	commsFlex := tview.NewFlex()
	commsFlex.SetDirection(tview.FlexColumn)
	commsFlex.AddItem(commandsTextView, 0, 1, false)
	commsFlex.AddItem(pushMessagesTextView, 0, 1, false)

	// Control
	controlFlex := tview.NewFlex()
	controlFlex.SetBorder(true)
	controlFlex.SetTitle("Contrtol")
	controlFlex.SetDirection(tview.FlexRow)
	controlFlex.AddItem(gcodeFlex, 17, 0, false)
	controlFlex.AddItem(commsFlex, 0, 1, false)
	controlFlex.AddItem(commandInputField, 1, 0, true)
	cp.Flex = controlFlex

	cp.sendStatusCommands()

	return cp
}

func (cp *ControlPrimitive) sendStatusCommands() {
	for command := range allStatusCommands {
		cp.queueStatusCommand(command)
	}
}

func (cp *ControlPrimitive) setDisabledState() {
	cp.app.QueueUpdate(func() {
		cp.mu.Lock()
		defer cp.mu.Unlock()
		if cp.disableCommandInput || cp.machineState == nil {
			cp.commandInputField.SetDisabled(true)
			return
		}
		switch *cp.machineState {
		case "Idle":
			cp.commandInputField.SetDisabled(false)
		case "Run":
			cp.commandInputField.SetDisabled(true)
		case "Hold":
			cp.commandInputField.SetDisabled(true)
		case "Jog":
			cp.commandInputField.SetDisabled(true)
		case "Alarm":
			cp.commandInputField.SetDisabled(true)
		case "Door":
			cp.commandInputField.SetDisabled(true)
		case "Check":
			cp.commandInputField.SetDisabled(false)
		case "Home":
			cp.commandInputField.SetDisabled(true)
		case "Sleep":
			cp.commandInputField.SetDisabled(true)
		default:
			panic(fmt.Errorf("unknown state: %s", *cp.machineState))
		}
	})
}

func (cp *ControlPrimitive) setMachineState(machineState string) {
	cp.mu.Lock()
	cp.machineState = &machineState
	cp.mu.Unlock()
	cp.setDisabledState()
}

func (cp *ControlPrimitive) DisableCommandInput(ctx context.Context, disabled bool) {
	cp.mu.Lock()
	cp.disableCommandInput = disabled
	cp.mu.Unlock()
	cp.setDisabledState()
}

func (cp *ControlPrimitive) sendCommand(
	ctx context.Context,
	commandParameter *commandParameterType,
) {
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
		return
	}

	if commandParameter.quiet {
		return
	}

	fmt.Fprintf(cp.commandsTextView, "\n[%s]ok[-]", tcell.ColorGreen)
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
func (cp *ControlPrimitive) getBlockStatusCmdsAndTimeout(block *gcode.Block) (map[string]bool, time.Duration) {
	statusCommands := map[string]bool{}
	var timeout time.Duration
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
			timeout = homeCommandTimeout
			// Grbl stops responding to status report queries while homing. Generating this
			// virtual status report enables subscribers to process the otherwise unreported
			//  state.
			cp.pushMessageCh <- &grblMod.StatusReportPushMessage{
				Message: "(virtual push message: status report: Home)",
				MachineState: grblMod.MachineState{
					State: "Home",
				},
			}
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
				timeout = 0
			case "G38.2", "G38.3", "G38.4", "G38.5":
				timeout = probeCommandTimeout
			}
		}
		statusCommands[grblMod.GrblCommandViewGcodeParserState] = true
		statusCommands[grblMod.GrblCommandViewGcodeParameters] = true
	} else {
		panic("bug: unknown block")
	}
	return statusCommands, timeout
}

func (cp *ControlPrimitive) processCommand(ctx context.Context, command string) {
	realTimeCommands, command, err := cp.extractRealTimeCommands(command)
	if err != nil {
		fmt.Fprintf(cp.commandsTextView, "\n[%s]%s[-]", tcell.ColorRed, tview.Escape(err.Error()))
		return
	}
	for _, realTimeCommand := range realTimeCommands {
		cp.sendRealTimeCommand(realTimeCommand)
	}
	if len(command) == 0 {
		return
	}

	blocks, err := gcode.NewParser(strings.NewReader(command)).Blocks()
	if err != nil {
		fmt.Fprintf(cp.commandsTextView, "\n[%s]Failed to parse: %s[-]", tcell.ColorRed, tview.Escape(err.Error()))
		return
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
		var timeout time.Duration
		statusCommands, timeout = cp.getBlockStatusCmdsAndTimeout(block)
		if timeout > 0 {
			commandParameter.timeout = timeout
		}
	}

	cp.DisableCommandInput(ctx, true)
	defer cp.DisableCommandInput(ctx, false)
	cp.sendCommand(ctx, commandParameter)
	cp.sendCommand(ctx, syncCommand)
	for command := range statusCommands {
		cp.sendCommand(ctx, &commandParameterType{
			command: command,
			timeout: defaultCommandTimeout,
			quiet:   cp.quietStatusComms,
		})
	}
}

func (cp *ControlPrimitive) RunSendCommandWorker(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			err := ctx.Err()
			if errors.Is(err, context.Canceled) {
				err = nil
			}
			return err
		case command := <-cp.sendStatusCommandCh:
			cp.sendCommand(ctx, &commandParameterType{
				command: command,
				timeout: defaultCommandTimeout,
				quiet:   cp.quietStatusComms,
			})
		case command := <-cp.sendCommandCh:
			cp.processCommand(ctx, command)
		}
	}
}

func (cp *ControlPrimitive) queueStatusCommand(
	command string,
) {
	cp.sendStatusCommandCh <- command
}

func (cp *ControlPrimitive) QueueCommand(
	command string,
) {
	cp.sendCommandCh <- command
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

func (cp *ControlPrimitive) RunSendRealTimeCommandWorker(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			err := ctx.Err()
			if errors.Is(err, context.Canceled) {
				err = nil
			}
			return err
		case realTimeCommand := <-cp.sendRealTimeCommandCh:
			cp.sendRealTimeCommand(realTimeCommand)
		}
	}
}

//gocyclo:ignore
func (cp *ControlPrimitive) processGcodeStatePushMessage(
	gcodeStatePushMessage *grblMod.GcodeStatePushMessage,
) tcell.Color {
	var buf bytes.Buffer

	if modalGroup := gcodeStatePushMessage.ModalGroup; modalGroup != nil {
		if modalGroup.Motion != nil {
			fmt.Fprintf(
				&buf, "%s:%s\n",
				sprintGcodeWord(modalGroup.Motion.NormalizedString()), modalGroup.Motion.Name(),
			)
		}
		if modalGroup.PlaneSelection != nil {
			fmt.Fprintf(
				&buf, "%s:%s\n",
				sprintGcodeWord(modalGroup.PlaneSelection.NormalizedString()), modalGroup.PlaneSelection.Name(),
			)
		}
		if modalGroup.DistanceMode != nil {
			fmt.Fprintf(&buf, "%s:%s\n", sprintGcodeWord(modalGroup.DistanceMode.NormalizedString()), modalGroup.DistanceMode.Name())
		}
		if modalGroup.FeedRateMode != nil {
			fmt.Fprintf(&buf, "%s:%s\n", sprintGcodeWord(modalGroup.FeedRateMode.NormalizedString()), modalGroup.FeedRateMode.Name())
		}
		if modalGroup.Units != nil {
			fmt.Fprintf(&buf, "%s:%s\n", sprintGcodeWord(gcodeStatePushMessage.ModalGroup.Units.NormalizedString()), gcodeStatePushMessage.ModalGroup.Units.Name())
		}
		if modalGroup.CutterRadiusCompensation != nil {
			fmt.Fprintf(&buf, "%s:%s\n", sprintGcodeWord(modalGroup.CutterRadiusCompensation.NormalizedString()), modalGroup.CutterRadiusCompensation.Name())
		}
		if modalGroup.ToolLengthOffset != nil {
			fmt.Fprintf(&buf, "%s:%s\n", sprintGcodeWord(modalGroup.ToolLengthOffset.NormalizedString()), modalGroup.ToolLengthOffset.Name())
		}
		if modalGroup.CoordinateSystemSelection != nil {
			fmt.Fprintf(&buf, "%s:%s\n", sprintGcodeWord(modalGroup.CoordinateSystemSelection.NormalizedString()), modalGroup.CoordinateSystemSelection.Name())
		}
		if modalGroup.Stopping != nil {
			fmt.Fprintf(&buf, "%s:%s\n", sprintGcodeWord(modalGroup.Stopping.NormalizedString()), modalGroup.Stopping.Name())
		}
		if modalGroup.SpindleTurning != nil {
			fmt.Fprintf(&buf, "%s:%s\n", sprintGcodeWord(modalGroup.SpindleTurning.NormalizedString()), modalGroup.SpindleTurning.Name())
		}
		for _, word := range modalGroup.Coolant {
			fmt.Fprintf(&buf, "%s:%s\n", sprintGcodeWord(word.NormalizedString()), word.Name())
		}
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

	cp.app.QueueUpdate(func() {
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

	cp.app.QueueUpdate(func() {
		if buf.String() == cp.gcodeParamsTextView.GetText(false) {
			return
		}
		cp.gcodeParamsTextView.SetText(buf.String())
	})

	return color
}

func (cp *ControlPrimitive) processWelcomePushMessage() {
	cp.app.QueueUpdate(func() {
		cp.gcodeParserTextView.Clear()
		cp.gcodeParamsTextView.Clear()
	})
	fmt.Fprintf(cp.pushMessagesTextView, "\n[%s]Soft-Reset detected[-]", tcell.ColorOrange)
	cp.sendStatusCommands()
}

func (cp *ControlPrimitive) processAlarmPushMessage(
	alarmPushMessage *grblMod.AlarmPushMessage,
) (string, tcell.Color) {
	return tview.Escape(alarmPushMessage.Error().Error()), tcell.ColorRed
}

func (cp *ControlPrimitive) processStatusReportPushMessage(
	statusReportPushMessage *grblMod.StatusReportPushMessage,
) tcell.Color {
	color := getMachineStateColor(statusReportPushMessage.MachineState.State)
	if color == tcell.ColorBlack {
		color = tcell.ColorWhite
	}
	cp.setMachineState(statusReportPushMessage.MachineState.State)
	return color
}

//gocyclo:ignore
func (cp *ControlPrimitive) ProcessPushMessage(ctx context.Context, pushMessage grblMod.PushMessage) {
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
		color = cp.processStatusReportPushMessage(statusReportPushMessage)
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
