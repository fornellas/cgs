package grbl

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/fornellas/cgs/gcode"
)

////////////////////////////////////////////////////////////////////////////////////////////////////
// Welcome
////////////////////////////////////////////////////////////////////////////////////////////////////

type WelcomePushMessage struct {
	Message string
}

func (m *WelcomePushMessage) String() string {
	return m.Message
}

////////////////////////////////////////////////////////////////////////////////////////////////////
// Alarm
////////////////////////////////////////////////////////////////////////////////////////////////////

var alarmPushMessagePrefix = "ALARM:"

type AlarmPushMessage struct {
	Message string
}

func (m *AlarmPushMessage) String() string {
	return m.Message
}

//gocyclo:ignore
func (m *AlarmPushMessage) Error() error {
	n, err := strconv.Atoi(m.Message[len(alarmPushMessagePrefix):])
	if err != nil {
		return fmt.Errorf("unable to parse alarm number (%s)", m.Message)
	}
	switch n {
	case 1:
		//lint:ignore ST1005 vanilla Grbl message
		return errors.New("Hard limit triggered. Machine position is likely lost due to sudden and immediate halt. Re-homing is highly recommended.")
	case 2:
		//lint:ignore ST1005 vanilla Grbl message
		return errors.New("G-code motion target exceeds machine travel. Machine position safely retained. Alarm may be unlocked.")
	case 3:
		//lint:ignore ST1005 vanilla Grbl message
		return errors.New("Reset while in motion. Grbl cannot guarantee position. Lost steps are likely. Re-homing is highly recommended.")
	case 4:
		//lint:ignore ST1005 vanilla Grbl message
		return errors.New("Probe fail. The probe is not in the expected initial state before starting probe cycle, where G38.2 and G38.3 is not triggered and G38.4 and G38.5 is triggered.")
	case 5:
		//lint:ignore ST1005 vanilla Grbl message
		return errors.New("Probe fail. Probe did not contact the workpiece within the programmed travel for G38.2 and G38.4.")
	case 6:
		//lint:ignore ST1005 vanilla Grbl message
		return errors.New("Homing fail. Reset during active homing cycle.")
	case 7:
		//lint:ignore ST1005 vanilla Grbl message
		return errors.New("Homing fail. Safety door was opened during active homing cycle.")
	case 8:
		//lint:ignore ST1005 vanilla Grbl message
		return errors.New("Homing fail. Cycle failed to clear limit switch when pulling off. Try increasing pull-off setting or check wiring.")
	case 9:
		//lint:ignore ST1005 vanilla Grbl message
		return errors.New("Homing fail. Could not find limit switch within search distance. Defined as 1.5 * max_travel on search and 5 * pulloff on locate phases.")
	case 10:
		//lint:ignore ST1005 vanilla Grbl message
		return errors.New("Homing fail. On dual axis machines, could not find the second limit switch for self-squaring.")
	default:
		return fmt.Errorf("unknown (%s)", m.Message)
	}
}

////////////////////////////////////////////////////////////////////////////////////////////////////
// Setting
////////////////////////////////////////////////////////////////////////////////////////////////////

type SettingPushMessage struct {
	Message string
	Key     string
	Value   string
}

func NewSettingPushMessage(message string) (*SettingPushMessage, error) {
	if !strings.HasPrefix(message, "$") {
		return nil, fmt.Errorf("setting message does not start with $: %s", message)
	}
	parts := strings.Split(message[1:], "=")
	if len(parts) != 2 {
		return nil, fmt.Errorf("setting message does not contain exactly one =: %s", message)
	}
	return &SettingPushMessage{
		Message: message,
		Key:     parts[0],
		Value:   parts[1],
	}, nil
}

func (m *SettingPushMessage) String() string {
	return m.Message
}

////////////////////////////////////////////////////////////////////////////////////////////////////
// Feedback
////////////////////////////////////////////////////////////////////////////////////////////////////

type FeedbackPushMessage struct {
	Message string
}

func (m *FeedbackPushMessage) String() string {
	return m.Message
}

func (m *FeedbackPushMessage) Text() string {
	return strings.TrimSuffix(strings.TrimPrefix(m.Message, "[MSG:"), "]")
}

////////////////////////////////////////////////////////////////////////////////////////////////////
// GcodeState
////////////////////////////////////////////////////////////////////////////////////////////////////

type GcodeStatePushMessage struct {
	Message      string
	ModalGroup   *gcode.ModalGroup
	Tool         *float64
	SpindleSpeed *float64
	FeedRate     *float64
}

func NewGcodeStatePushMessage(message string) (*GcodeStatePushMessage, error) {
	m := &GcodeStatePushMessage{
		Message:    message,
		ModalGroup: gcode.DefaultModalGroup.Copy(),
	}

	block := strings.TrimSuffix(strings.TrimPrefix(message, "[GC:"), "]")
	for wordStr := range strings.SplitSeq(block, " ") {
		word, err := gcode.NewWordFromString(wordStr)
		if err != nil {
			return nil, err
		}
		m.ModalGroup.Update(word)
		switch word.Letter() {
		case 'T':
			n := word.Number()
			m.Tool = &n
		case 'F':
			n := word.Number()
			m.FeedRate = &n
		case 'S':
			n := word.Number()
			m.SpindleSpeed = &n
		}
	}

	return m, nil
}

func (m *GcodeStatePushMessage) String() string {
	return m.Message
}

////////////////////////////////////////////////////////////////////////////////////////////////////
// Help
////////////////////////////////////////////////////////////////////////////////////////////////////

type HelpPushMessage struct {
	Message string
}

func (m *HelpPushMessage) String() string {
	return m.Message
}

////////////////////////////////////////////////////////////////////////////////////////////////////
// GcodeParam
////////////////////////////////////////////////////////////////////////////////////////////////////

type Probe struct {
	Coordinates Coordinates
	Successful  bool
}

func NewProbe(message string) (*Probe, error) {
	if !strings.HasPrefix(message, "[PRB:") || !strings.HasSuffix(message, "]") {
		return nil, fmt.Errorf("probe message malformed: %#v", message)
	}

	content := message[5 : len(message)-1]

	lastColonIdx := strings.LastIndex(content, ":")
	if lastColonIdx == -1 {
		return nil, fmt.Errorf("probe message missing success flag: %#v", message)
	}

	coordStr := content[:lastColonIdx]
	successStr := content[lastColonIdx+1:]

	coordinates, err := NewCoordinatesFromCSV(coordStr)
	if err != nil {
		return nil, fmt.Errorf("probe message coordinates invalid: %#v: %w", message, err)
	}

	if successStr != "0" && successStr != "1" {
		return nil, fmt.Errorf("probe message success flag invalid: %#v", message)
	}

	return &Probe{
		Coordinates: *coordinates,
		Successful:  successStr == "1",
	}, nil
}

var gcodeParamPrefixes = map[string]bool{
	"[G54:": true,
	"[G55:": true,
	"[G56:": true,
	"[G57:": true,
	"[G58:": true,
	"[G59:": true,
	"[G28:": true,
	"[G30:": true,
	"[G92:": true,
	"[TLO:": true,
	"[PRB:": true,
}

// G-Code parameters, as reported by Grbl via $#.
type GcodeParameters struct {
	// Coordinate system 1 (G54)
	CoordinateSystem1 *Coordinates
	// Coordinate system 2 (G55)
	CoordinateSystem2 *Coordinates
	// Coordinate system 3 (G56)
	CoordinateSystem3 *Coordinates
	// Coordinate system 4 (G57)
	CoordinateSystem4 *Coordinates
	// Coordinate system 5 (G58)
	CoordinateSystem5 *Coordinates
	// Coordinate system 6 (G59)
	CoordinateSystem6 *Coordinates
	// Primary Pre-Defined Position (G28)
	PrimaryPreDefinedPosition *Coordinates
	// Secondary Pre-Defined Position (G30)
	SecondaryPreDefinedPosition *Coordinates
	// Coordinate Offset (G92)
	CoordinateOffset *Coordinates
	// Tool length offset (for the default z-axis)
	ToolLengthOffset *float64
	// Last probing cycle
	Probe *Probe
}

// Updates GcodeParameters from given GcodeParamPushMessage.
func (g *GcodeParameters) Update(gcodeParamPushMessage *GcodeParamPushMessage) {
	if gcodeParamPushMessage.GcodeParameters.CoordinateSystem1 != nil {
		g.CoordinateSystem1 = gcodeParamPushMessage.GcodeParameters.CoordinateSystem1
	}
	if gcodeParamPushMessage.GcodeParameters.CoordinateSystem2 != nil {
		g.CoordinateSystem2 = gcodeParamPushMessage.GcodeParameters.CoordinateSystem2
	}
	if gcodeParamPushMessage.GcodeParameters.CoordinateSystem3 != nil {
		g.CoordinateSystem3 = gcodeParamPushMessage.GcodeParameters.CoordinateSystem3
	}
	if gcodeParamPushMessage.GcodeParameters.CoordinateSystem4 != nil {
		g.CoordinateSystem4 = gcodeParamPushMessage.GcodeParameters.CoordinateSystem4
	}
	if gcodeParamPushMessage.GcodeParameters.CoordinateSystem5 != nil {
		g.CoordinateSystem5 = gcodeParamPushMessage.GcodeParameters.CoordinateSystem5
	}
	if gcodeParamPushMessage.GcodeParameters.CoordinateSystem6 != nil {
		g.CoordinateSystem6 = gcodeParamPushMessage.GcodeParameters.CoordinateSystem6
	}
	if gcodeParamPushMessage.GcodeParameters.PrimaryPreDefinedPosition != nil {
		g.PrimaryPreDefinedPosition = gcodeParamPushMessage.GcodeParameters.PrimaryPreDefinedPosition
	}
	if gcodeParamPushMessage.GcodeParameters.SecondaryPreDefinedPosition != nil {
		g.SecondaryPreDefinedPosition = gcodeParamPushMessage.GcodeParameters.SecondaryPreDefinedPosition
	}
	if gcodeParamPushMessage.GcodeParameters.CoordinateOffset != nil {
		g.CoordinateOffset = gcodeParamPushMessage.GcodeParameters.CoordinateOffset
	}
	if gcodeParamPushMessage.GcodeParameters.ToolLengthOffset != nil {
		g.ToolLengthOffset = gcodeParamPushMessage.GcodeParameters.ToolLengthOffset
	}
	if gcodeParamPushMessage.GcodeParameters.Probe != nil {
		g.Probe = gcodeParamPushMessage.GcodeParameters.Probe
	}
}

func (g *GcodeParameters) HasCoordinateSystem() bool {
	if g.CoordinateSystem1 != nil {
		return true
	}
	if g.CoordinateSystem2 != nil {
		return true
	}
	if g.CoordinateSystem3 != nil {
		return true
	}
	if g.CoordinateSystem4 != nil {
		return true
	}
	if g.CoordinateSystem5 != nil {
		return true
	}
	if g.CoordinateSystem6 != nil {
		return true
	}
	return false
}

func (g *GcodeParameters) HasPreDefinedPosition() bool {
	if g.PrimaryPreDefinedPosition != nil {
		return true
	}
	if g.SecondaryPreDefinedPosition != nil {
		return true
	}
	return false
}

type GcodeParamPushMessage struct {
	Message         string
	GcodeParameters GcodeParameters
}

//gocyclo:ignore
func NewGcodeParamPushMessage(message string) (*GcodeParamPushMessage, error) {
	m := &GcodeParamPushMessage{
		Message: message,
	}

	if !strings.HasPrefix(message, "[") || !strings.HasSuffix(message, "]") {
		return nil, fmt.Errorf("gcode param message malformed: not surrounded by []: %#v", message)
	}

	content := message[1 : len(message)-1]
	colonIdx := strings.Index(content, ":")
	if colonIdx == -1 {
		return nil, fmt.Errorf("gcode param message malformed: missing colon: %#v", message)
	}

	paramType := content[:colonIdx]
	paramValue := content[colonIdx+1:]

	switch paramType {
	case "G54":
		coordinates, err := NewCoordinatesFromCSV(paramValue)
		if err != nil {
			return nil, fmt.Errorf("gcode param G54 invalid: %#v: %w", message, err)
		}
		m.GcodeParameters.CoordinateSystem1 = coordinates
	case "G55":
		coordinates, err := NewCoordinatesFromCSV(paramValue)
		if err != nil {
			return nil, fmt.Errorf("gcode param G55 invalid: %#v: %w", message, err)
		}
		m.GcodeParameters.CoordinateSystem2 = coordinates
	case "G56":
		coordinates, err := NewCoordinatesFromCSV(paramValue)
		if err != nil {
			return nil, fmt.Errorf("gcode param G56 invalid: %#v: %w", message, err)
		}
		m.GcodeParameters.CoordinateSystem3 = coordinates
	case "G57":
		coordinates, err := NewCoordinatesFromCSV(paramValue)
		if err != nil {
			return nil, fmt.Errorf("gcode param G57 invalid: %#v: %w", message, err)
		}
		m.GcodeParameters.CoordinateSystem4 = coordinates
	case "G58":
		coordinates, err := NewCoordinatesFromCSV(paramValue)
		if err != nil {
			return nil, fmt.Errorf("gcode param G58 invalid: %#v: %w", message, err)
		}
		m.GcodeParameters.CoordinateSystem5 = coordinates
	case "G59":
		coordinates, err := NewCoordinatesFromCSV(paramValue)
		if err != nil {
			return nil, fmt.Errorf("gcode param G59 invalid: %#v: %w", message, err)
		}
		m.GcodeParameters.CoordinateSystem6 = coordinates
	case "G28":
		coordinates, err := NewCoordinatesFromCSV(paramValue)
		if err != nil {
			return nil, fmt.Errorf("gcode param G28 invalid: %#v: %w", message, err)
		}
		m.GcodeParameters.PrimaryPreDefinedPosition = coordinates
	case "G30":
		coordinates, err := NewCoordinatesFromCSV(paramValue)
		if err != nil {
			return nil, fmt.Errorf("gcode param G30 invalid: %#v: %w", message, err)
		}
		m.GcodeParameters.SecondaryPreDefinedPosition = coordinates
	case "G92":
		coordinates, err := NewCoordinatesFromCSV(paramValue)
		if err != nil {
			return nil, fmt.Errorf("gcode param G92 invalid: %#v: %w", message, err)
		}
		m.GcodeParameters.CoordinateOffset = coordinates
	case "TLO":
		offset, err := strconv.ParseFloat(paramValue, 64)
		if err != nil {
			return nil, fmt.Errorf("gcode param TLO invalid: %#v: %w", message, err)
		}
		m.GcodeParameters.ToolLengthOffset = &offset
	case "PRB":
		probe, err := NewProbe(message)
		if err != nil {
			return nil, fmt.Errorf("gcode param PRB invalid: %#v: %w", message, err)
		}
		m.GcodeParameters.Probe = probe
	default:
		return nil, fmt.Errorf("gcode param message unknown type: %#v", message)
	}

	return m, nil
}

func (m *GcodeParamPushMessage) String() string {
	return m.Message
}

////////////////////////////////////////////////////////////////////////////////////////////////////
// Version
////////////////////////////////////////////////////////////////////////////////////////////////////

type VersionPushMessage struct {
	Message string
	Version string
	Info    string
}

func NewVersionPushMessage(message string) (*VersionPushMessage, error) {
	const prefix = "[VER:"
	const suffix = "]"
	const sep = ":"
	if !strings.HasPrefix(message, prefix) {
		return nil, fmt.Errorf("message does not contain prefix %#v: %#v", prefix, message)
	}
	if !strings.HasSuffix(message, suffix) {
		return nil, fmt.Errorf("message does not contain suffix %#v: %#v", suffix, message)
	}
	text := strings.TrimSuffix(strings.TrimPrefix(message, prefix), suffix)
	parts := strings.Split(text, sep)
	if len(parts) != 2 {
		return nil, fmt.Errorf("message format unknown: %#v", message)
	}
	return &VersionPushMessage{
		Message: message,
		Version: parts[0],
		Info:    parts[1],
	}, nil
}

func (m *VersionPushMessage) String() string {
	return m.Message
}

////////////////////////////////////////////////////////////////////////////////////////////////////
// CompileTimeOptions
////////////////////////////////////////////////////////////////////////////////////////////////////

type CompileTimeOptionsPushMessage struct {
	Message             string
	CompileTimeOptions  []string
	PlannerBlocks       uint64
	SerialRxBufferBytes uint64
}

var buildOptionDescription = map[rune]string{
	'V': "Variable spindle",
	'N': "Line numbers",
	'M': "Mist coolant M7",
	'C': "CoreXY",
	'P': "Parking motion",
	'Z': "Homing force origin",
	'H': "Homing single axis commands",
	'T': "Two limit switches on axis",
	'A': "Allow feed rate overrides in probe cycles",
	'D': "Use spindle direction as enable pin",
	'0': "Spindle enable off when speed is zero",
	'S': "Software limit pin debouncing",
	'R': "Parking override control",
	'+': "Safety door input pin",
	'*': "Restore all EEPROM command",
	'$': "Restore EEPROM `$` settings command",
	'#': "Restore EEPROM parameter data command",
	'I': "Build info write user string command",
	'E': "Force sync upon EEPROM write",
	'W': "Force sync upon work coordinate offset change",
	'L': "Homing initialization auto-lock",
	'2': "Dual axis motors",
}

func NewCompileTimeOptionsPushMessage(message string) (*CompileTimeOptionsPushMessage, error) {
	const prefix = "[OPT:"
	const suffix = "]"
	const sep = ","
	if !strings.HasPrefix(message, prefix) {
		return nil, fmt.Errorf("message does not contain prefix %#v: %#v", prefix, message)
	}
	if !strings.HasSuffix(message, suffix) {
		return nil, fmt.Errorf("message does not contain suffix %#v: %#v", suffix, message)
	}
	text := strings.TrimSuffix(strings.TrimPrefix(message, prefix), suffix)
	parts := strings.Split(text, sep)
	if len(parts) != 3 {
		return nil, fmt.Errorf("message format unknown: %#v", message)
	}
	compileTimeOptions := []string{}
	for _, code := range parts[0] {
		var opt string
		var ok bool
		if opt, ok = buildOptionDescription[code]; !ok {
			opt = fmt.Sprintf("unknown (%c)", code)
		}
		compileTimeOptions = append(compileTimeOptions, opt)
	}
	var plannerBlocks uint64
	var err error
	if plannerBlocks, err = strconv.ParseUint(parts[1], 10, 64); err != nil {
		return nil, fmt.Errorf("unable to parse planner blocks: %#v: %w", message, err)
	}
	var serialRxBufferBytes uint64
	if serialRxBufferBytes, err = strconv.ParseUint(parts[2], 10, 64); err != nil {
		return nil, fmt.Errorf("unable to parse serial RX buffer bytes: %#v: %w", message, err)
	}
	return &CompileTimeOptionsPushMessage{
		Message:             message,
		CompileTimeOptions:  compileTimeOptions,
		PlannerBlocks:       plannerBlocks,
		SerialRxBufferBytes: serialRxBufferBytes,
	}, nil
}

func (m *CompileTimeOptionsPushMessage) String() string {
	return m.Message
}

////////////////////////////////////////////////////////////////////////////////////////////////////
// StartupLineExecution
////////////////////////////////////////////////////////////////////////////////////////////////////

type StartupLineExecutionPushMessage struct {
	Message string
}

func (m *StartupLineExecutionPushMessage) String() string {
	return m.Message
}

////////////////////////////////////////////////////////////////////////////////////////////////////
// StatusReport
////////////////////////////////////////////////////////////////////////////////////////////////////

type State string

var StateIdle State = "Idle"
var StateRun State = "Run"
var StateHold State = "Hold"
var StateJog State = "Jog"
var StateAlarm State = "Alarm"
var StateDoor State = "Door"
var StateCheck State = "Check"
var StateHome State = "Home"
var StateSleep State = "Sleep"
var StateUnknown State = ""

var knownStates = map[State]bool{
	StateIdle:  true,
	StateRun:   true,
	StateHold:  true,
	StateJog:   true,
	StateAlarm: true,
	StateDoor:  true,
	StateCheck: true,
	StateHome:  true,
	StateSleep: true,
}

type MachineState struct {
	// Valid states types:  `Idle, Run, Hold, Jog, Alarm, Door, Check, Home, Sleep`
	State State
	// Current sub-states are:
	// - `Hold:0` Hold complete. Ready to resume.
	// - `Hold:1` Hold in-progress. Reset will throw an alarm.
	// - `Door:0` Door closed. Ready to resume.
	// - `Door:1` Machine stopped. Door still ajar. Can't resume until closed.
	// - `Door:2` Door opened. Hold (or parking retract) in-progress. Reset will throw an alarm.
	// - `Door:3` Door closed and resuming. Restoring from park, if applicable. Reset will throw an alarm.
	SubState *int
}

func NewMachineState(dataField string) (*MachineState, error) {
	parts := strings.Split(dataField, ":")
	if len(parts) < 1 {
		return nil, fmt.Errorf("machine state field empty: %#v", dataField)
	}
	if len(parts) > 2 {
		return nil, fmt.Errorf("machine state field malformed: %#v", dataField)
	}
	state := State(parts[0])
	var subStatePtr *int
	if len(parts) == 2 {
		subState, err := strconv.Atoi(parts[1])
		if err != nil {
			return nil, fmt.Errorf("machine state substate invalid: %#v", dataField)
		}
		subStatePtr = &subState
	}
	if _, ok := knownStates[state]; !ok {
		return nil, fmt.Errorf("unknown machine state: %#v", state)
	}

	return &MachineState{
		State:    state,
		SubState: subStatePtr,
	}, nil
}

func (m *MachineState) SubStateString() string {
	if m.SubState == nil {
		return ""
	}
	switch m.State {
	case "Hold":
		switch *m.SubState {
		case 0:
			return "complete"
		case 1:
			return "in-progress"
		}
	case "Door":
		switch *m.SubState {
		case 0:
			return "closed"
		case 1:
			return "ajar"
		case 2:
			return "opened"
		case 3:
			return "resuming"
		}
	}
	return fmt.Sprintf("unknown (%d)", *m.SubState)
}

type MachinePosition Coordinates

func NewMachinePosition(dataValues []string) (*MachinePosition, error) {
	coordinates, err := NewCoordinatesFromStrValues(dataValues)
	if err != nil {
		return nil, err
	}
	return (*MachinePosition)(coordinates), nil
}

type WorkPosition Coordinates

func NewWorkPosition(dataValues []string) (*WorkPosition, error) {
	coordinates, err := NewCoordinatesFromStrValues(dataValues)
	if err != nil {
		return nil, err
	}
	return (*WorkPosition)(coordinates), nil
}

// Work coordinate offset is the current work coordinate offset of the g-code parser, which is the
// sum of the current work coordinate system, G92 offsets, and G43.1 tool length offset.
type WorkCoordinateOffset Coordinates

func NewWorkCoordinateOffset(dataValues []string) (*WorkCoordinateOffset, error) {
	coordinates, err := NewCoordinatesFromStrValues(dataValues)
	if err != nil {
		return nil, err
	}
	return (*WorkCoordinateOffset)(coordinates), nil
}

type BufferState struct {
	// Number of available blocks in the planner buffer
	AvailableBlocks int
	// Number of available bytes in the serial RX buffer
	AvailableBytes int
}

func NewBufferState(dataValues []string) (*BufferState, error) {
	if len(dataValues) != 2 {
		return nil, fmt.Errorf("buffer state field malformed: %#v", dataValues)
	}

	availableBlocks, err := strconv.Atoi(dataValues[0])
	if err != nil {
		return nil, fmt.Errorf("buffer state available blocks invalid: %#v", dataValues[0])
	}

	availableBytes, err := strconv.Atoi(dataValues[1])
	if err != nil {
		return nil, fmt.Errorf("buffer state available bytes invalid: %#v", dataValues[1])
	}

	return &BufferState{
		AvailableBlocks: availableBlocks,
		AvailableBytes:  availableBytes,
	}, nil
}

// Line currently being executed
type LineNumber int

func NewLineNumber(dataValues []string) (*LineNumber, error) {
	if len(dataValues) != 1 {
		return nil, fmt.Errorf("line number field malformed: %#v", dataValues)
	}

	lineNumber, err := strconv.Atoi(dataValues[0])
	if err != nil {
		return nil, fmt.Errorf("line number invalid: %#v", dataValues[0])
	}

	result := LineNumber(lineNumber)
	return &result, nil
}

// Current Feed
type Feed float64

func NewFeed(dataValues []string) (*Feed, error) {
	if len(dataValues) != 1 {
		return nil, fmt.Errorf("feed field malformed: %#v", dataValues)
	}

	feed, err := strconv.ParseFloat(dataValues[0], 64)
	if err != nil {
		return nil, fmt.Errorf("feed invalid: %#v", dataValues[0])
	}

	result := Feed(feed)
	return &result, nil
}

// Current Feed and Speed
type FeedSpindle struct {
	Feed  float64
	Speed float64
}

func NewStatusReportFeedSpindle(dataValues []string) (*FeedSpindle, error) {
	if len(dataValues) != 2 {
		return nil, fmt.Errorf("feed spindle field malformed: %#v", dataValues)
	}

	feed, err := strconv.ParseFloat(dataValues[0], 64)
	if err != nil {
		return nil, fmt.Errorf("feed spindle feed invalid: %#v", dataValues[0])
	}

	speed, err := strconv.ParseFloat(dataValues[1], 64)
	if err != nil {
		return nil, fmt.Errorf("feed spindle speed invalid: %#v", dataValues[1])
	}

	return &FeedSpindle{
		Feed:  feed,
		Speed: speed,
	}, nil
}

// Input pins Grbl has detected as 'triggered'.
type PinState struct {
	XLimit     *bool
	YLimit     *bool
	ZLimit     *bool
	ALimit     *bool
	Probe      *bool
	Door       *bool
	Hold       *bool
	SoftReset  *bool
	CycleStart *bool
}

func NewPinState(dataValues []string) (*PinState, error) {
	if len(dataValues) != 1 {
		return nil, fmt.Errorf("pin state field malformed: %#v", dataValues)
	}

	pinState := &PinState{}
	pins := dataValues[0]

	for _, pin := range pins {
		trueVal := true
		switch pin {
		case 'X':
			pinState.XLimit = &trueVal
		case 'Y':
			pinState.YLimit = &trueVal
		case 'Z':
			pinState.ZLimit = &trueVal
		case 'A':
			pinState.ALimit = &trueVal
		case 'P':
			pinState.Probe = &trueVal
		case 'D':
			pinState.Door = &trueVal
		case 'H':
			pinState.Hold = &trueVal
		case 'R':
			pinState.SoftReset = &trueVal
		case 'S':
			pinState.CycleStart = &trueVal
		default:
			return nil, fmt.Errorf("pin state unknown pin: %#v", string(pin))
		}
	}

	return pinState, nil
}

//gocyclo:ignore
func (p *PinState) String() string {
	var buf bytes.Buffer
	if p.XLimit != nil && *p.XLimit {
		fmt.Fprint(&buf, "X")
	}
	if p.YLimit != nil && *p.YLimit {
		fmt.Fprint(&buf, "Y")
	}
	if p.ZLimit != nil && *p.ZLimit {
		fmt.Fprint(&buf, "Z")
	}
	if p.ALimit != nil && *p.ALimit {
		fmt.Fprint(&buf, "A")
	}
	if p.Probe != nil && *p.Probe {
		fmt.Fprint(&buf, "P")
	}
	if p.Door != nil && *p.Door {
		fmt.Fprint(&buf, "D")
	}
	if p.Hold != nil && *p.Hold {
		fmt.Fprint(&buf, "H")
	}
	if p.SoftReset != nil && *p.SoftReset {
		fmt.Fprint(&buf, "R")
	}
	if p.CycleStart != nil && *p.CycleStart {
		fmt.Fprint(&buf, "S")
	}
	return buf.String()
}

// Indicates current override values in percent of programmed values.
type OverrideValues struct {
	Feed    float64
	Rapids  float64
	Spindle float64
}

func NewOverrideValues(dataValues []string) (*OverrideValues, error) {
	if len(dataValues) != 3 {
		return nil, fmt.Errorf("override values field malformed: %#v", dataValues)
	}

	feed, err := strconv.ParseFloat(dataValues[0], 64)
	if err != nil {
		return nil, fmt.Errorf("override values feed invalid: %#v", dataValues[0])
	}

	rapids, err := strconv.ParseFloat(dataValues[1], 64)
	if err != nil {
		return nil, fmt.Errorf("override values rapids invalid: %#v", dataValues[1])
	}

	spindle, err := strconv.ParseFloat(dataValues[2], 64)
	if err != nil {
		return nil, fmt.Errorf("override values spindle invalid: %#v", dataValues[2])
	}

	return &OverrideValues{
		Feed:    feed,
		Rapids:  rapids,
		Spindle: spindle,
	}, nil
}

func (o *OverrideValues) HasOverride() bool {
	if o.Feed != 100 || o.Rapids != 100 || o.Spindle != 100 {
		return true
	}
	return false
}

type AccessoryState struct {
	// indicates spindle is enabled in the CW direction. This does not appear with `C`.
	SpindleCW *bool
	// indicates spindle is enabled in the CCW direction. This does not appear with `S`.
	SpindleCCW *bool
	// indicates flood coolant is enabled.
	FloodCoolant *bool
	// indicates mist coolant is enabled.
	MistCoolant *bool
}

func NewAccessoryState(dataValues []string) (*AccessoryState, error) {
	if len(dataValues) != 1 {
		return nil, fmt.Errorf("accessory state field malformed: %#v", dataValues)
	}

	accessoryState := &AccessoryState{}
	accessories := dataValues[0]

	for _, accessory := range accessories {
		trueVal := true
		switch accessory {
		case 'S':
			accessoryState.SpindleCW = &trueVal
		case 'C':
			accessoryState.SpindleCCW = &trueVal
		case 'F':
			accessoryState.FloodCoolant = &trueVal
		case 'M':
			accessoryState.MistCoolant = &trueVal
		default:
			return nil, fmt.Errorf("accessory state unknown accessory: %#v", string(accessory))
		}
	}

	return accessoryState, nil
}

type StatusReportPushMessage struct {
	Message              string
	MachineState         MachineState
	MachinePosition      *MachinePosition
	WorkPosition         *WorkPosition
	WorkCoordinateOffset *WorkCoordinateOffset
	BufferState          *BufferState
	LineNumber           *LineNumber
	Feed                 *Feed
	FeedSpindle          *FeedSpindle
	PinState             *PinState
	OverrideValues       *OverrideValues
	AccessoryState       *AccessoryState
}

//gocyclo:ignore
func NewStatusReportPushMessage(message string) (*StatusReportPushMessage, error) {
	if !strings.HasPrefix(message, "<") {
		return nil, fmt.Errorf("status report message does not start with '<': %#v", message)
	}

	dataFields := strings.Split(message[1:len(message)-1], "|")
	if len(dataFields) < 2 {
		return nil, fmt.Errorf("status report message missing required data fields: %#v", message)
	}

	machineState, err := NewMachineState(dataFields[0])
	if err != nil {
		return nil, fmt.Errorf("status report message parsing failed: %#v: %w", message, err)
	}

	statusReportPushMessage := &StatusReportPushMessage{
		Message:      message,
		MachineState: *machineState,
	}

	for _, dataField := range dataFields[1:] {
		parts := strings.Split(dataField, ":")
		if len(parts) != 2 {
			return nil, fmt.Errorf("status report message malformed data field: %#v: %#v", message, dataField)
		}
		dataType := parts[0]
		dataValues := strings.Split(parts[1], ",")

		switch dataType {
		case "MPos":
			statusReportPushMessage.MachinePosition, err = NewMachinePosition(dataValues)
			if err != nil {
				return nil, fmt.Errorf("status report message: failed to parse MPos: %w", err)
			}
		case "WPos":
			statusReportPushMessage.WorkPosition, err = NewWorkPosition(dataValues)
			if err != nil {
				return nil, fmt.Errorf("status report message: failed to parse WPos: %w", err)
			}
		case "WCO":
			statusReportPushMessage.WorkCoordinateOffset, err = NewWorkCoordinateOffset(dataValues)
			if err != nil {
				return nil, fmt.Errorf("status report message: failed to parse WCO: %w", err)
			}
		case "Bf":
			statusReportPushMessage.BufferState, err = NewBufferState(dataValues)
			if err != nil {
				return nil, fmt.Errorf("status report message: failed to parse Bf: %w", err)
			}
		case "Ln":
			statusReportPushMessage.LineNumber, err = NewLineNumber(dataValues)
			if err != nil {
				return nil, fmt.Errorf("status report message: failed to parse Ln: %w", err)
			}
		case "F":
			statusReportPushMessage.Feed, err = NewFeed(dataValues)
			if err != nil {
				return nil, fmt.Errorf("status report message: failed to parse F: %w", err)
			}
		case "FS":
			statusReportPushMessage.FeedSpindle, err = NewStatusReportFeedSpindle(dataValues)
			if err != nil {
				return nil, fmt.Errorf("status report message: failed to parse FS: %w", err)
			}
		case "Pn":
			statusReportPushMessage.PinState, err = NewPinState(dataValues)
			if err != nil {
				return nil, fmt.Errorf("status report message: failed to parse Pn: %w", err)
			}
		case "Ov":
			statusReportPushMessage.OverrideValues, err = NewOverrideValues(dataValues)
			if err != nil {
				return nil, fmt.Errorf("status report message: failed to parse Ov: %w", err)
			}
		case "A":
			statusReportPushMessage.AccessoryState, err = NewAccessoryState(dataValues)
			if err != nil {
				return nil, fmt.Errorf("status report message: failed to parse A: %w", err)
			}
		}
	}

	if !strings.HasPrefix(message, "<") {
		return nil, fmt.Errorf("status report message does not end with '>': %#v", message)
	}

	return statusReportPushMessage, nil
}

func (m *StatusReportPushMessage) String() string {
	return m.Message
}

////////////////////////////////////////////////////////////////////////////////////////////////////
// Echo
////////////////////////////////////////////////////////////////////////////////////////////////////

type EchoPushMessage struct {
	Message string
}

func (m *EchoPushMessage) String() string {
	return m.Message
}

////////////////////////////////////////////////////////////////////////////////////////////////////
// Empty
////////////////////////////////////////////////////////////////////////////////////////////////////

type EmptyPushMessage struct {
}

func (m *EmptyPushMessage) String() string {
	return "(empty)"
}

////////////////////////////////////////////////////////////////////////////////////////////////////
// New
////////////////////////////////////////////////////////////////////////////////////////////////////

type PushMessage interface {
	String() string
}

func NewPushMessage(message string) (PushMessage, error) {
	if strings.HasPrefix(message, "Grbl ") {
		return &WelcomePushMessage{Message: message}, nil
	}
	if strings.HasPrefix(message, alarmPushMessagePrefix) {
		return &AlarmPushMessage{Message: message}, nil
	}
	if strings.HasPrefix(message, "$") {
		return NewSettingPushMessage(message)
	}
	if strings.HasPrefix(message, "[MSG:") {
		return &FeedbackPushMessage{Message: message}, nil
	}
	if strings.HasPrefix(message, "[GC:") {
		return NewGcodeStatePushMessage(message)
	}
	if strings.HasPrefix(message, "[HLP:") {
		return &HelpPushMessage{Message: message}, nil
	}
	for prefix := range gcodeParamPrefixes {
		if strings.HasPrefix(message, prefix) {
			return NewGcodeParamPushMessage(message)
		}
	}
	if strings.HasPrefix(message, "[VER:") {
		return NewVersionPushMessage(message)
	}
	if strings.HasPrefix(message, "[OPT:") {
		return NewCompileTimeOptionsPushMessage(message)
	}
	if strings.HasPrefix(message, ">") {
		return &StartupLineExecutionPushMessage{Message: message}, nil
	}
	if strings.HasPrefix(message, "<") {
		return NewStatusReportPushMessage(message)
	}
	if strings.HasPrefix(message, "[echo:") {
		return &EchoPushMessage{Message: message}, nil
	}
	if len(message) == 0 {
		return &EmptyPushMessage{}, nil
	}
	return nil, ErrInvalidMessage
}
