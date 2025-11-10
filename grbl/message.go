package grbl

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

type MessageType int

const (
	// Message sent back from Grbl in response to a block being sent.
	MessageTypeResponse MessageType = iota
	// Message pushed back from Grbl either asynchronously or in response to a block.
	MessageTypePush
)

// Message represents a message received from Grbl.
type Message interface {
	Type() MessageType
	String() string
}

var messageResponseOk = "ok"
var messageResponseErrorPrefix = "error:"

type MessageResponse struct {
	Message string
}

func (m *MessageResponse) Type() MessageType {
	return MessageTypeResponse
}

func (m *MessageResponse) String() string {
	return m.Message
}

//gocyclo:ignore
func (m *MessageResponse) Error() error {
	if !strings.HasPrefix(m.Message, messageResponseErrorPrefix) {
		return nil
	}

	n, err := strconv.Atoi(m.Message[len(messageResponseErrorPrefix):])
	if err != nil {
		return fmt.Errorf("unable to parse error number (%s)", m.Message)
	}

	switch n {
	case 1:
		//lint:ignore ST1005 vanilla Grbl error string
		return errors.New("G-code words consist of a letter and a value. Letter was not found")
	case 2:
		//lint:ignore ST1005 vanilla Grbl error string
		return errors.New("Numeric value format is not valid or missing an expected value")
	case 3:
		return errors.New("Grbl '$' system command was not recognized or supported")
	case 4:
		//lint:ignore ST1005 vanilla Grbl error string
		return errors.New("Negative value received for an expected positive value")
	case 5:
		//lint:ignore ST1005 vanilla Grbl error string
		return errors.New("Homing cycle is not enabled via settings")
	case 6:
		//lint:ignore ST1005 vanilla Grbl error string
		return errors.New("Minimum step pulse time must be greater than 3usec")
	case 7:
		return errors.New("EEPROM read failed. Reset and restored to default values")
	case 8:
		return errors.New("Grbl '$' command cannot be used unless Grbl is IDLE. Ensures smooth operation during a job")
	case 9:
		//lint:ignore ST1005 vanilla Grbl error string
		return errors.New("G-code locked out during alarm or jog state")
	case 10:
		//lint:ignore ST1005 vanilla Grbl error string
		return errors.New("Soft limits cannot be enabled without homing also enabled")
	case 11:
		//lint:ignore ST1005 vanilla Grbl error string
		return errors.New("Max characters per line exceeded. Line was not processed and executed")
	case 12:
		return errors.New("(Compile Option) Grbl '$' setting value exceeds the maximum step rate supported")
	case 13:
		//lint:ignore ST1005 vanilla Grbl error string
		return errors.New("Safety door detected as opened and door state initiated")
	case 14:
		return errors.New("(Grbl-Mega Only) Build info or startup line exceeded EEPROM line length limit")
	case 15:
		//lint:ignore ST1005 vanilla Grbl error string
		return errors.New("Jog target exceeds machine travel. Command ignored")
	case 16:
		//lint:ignore ST1005 vanilla Grbl error string
		return errors.New("Jog command with no '=' or contains prohibited g-code")
	case 17:
		//lint:ignore ST1005 vanilla Grbl error string
		return errors.New("Laser mode requires PWM output")
	case 20:
		//lint:ignore ST1005 vanilla Grbl error string
		return errors.New("Unsupported or invalid g-code command found in block")
	case 21:
		//lint:ignore ST1005 vanilla Grbl error string
		return errors.New("More than one g-code command from same modal group found in block")
	case 22:
		//lint:ignore ST1005 vanilla Grbl error string
		return errors.New("Feed rate has not yet been set or is undefined")
	case 23:
		//lint:ignore ST1005 vanilla Grbl error string
		return errors.New("G-code command in block requires an integer value")
	case 24:
		//lint:ignore ST1005 vanilla Grbl error string
		return errors.New("Two G-code commands that both require the use of the XYZ axis words were detected in the block")
	case 25:
		//lint:ignore ST1005 vanilla Grbl error string
		return errors.New("A G-code word was repeated in the block")
	case 26:
		//lint:ignore ST1005 vanilla Grbl error string
		return errors.New("A G-code command implicitly or explicitly requires XYZ axis words in the block, but none were detected")
	case 27:
		//lint:ignore ST1005 vanilla Grbl error string
		return errors.New("N line number value is not within the valid range of 1 - 9,999,999")
	case 28:
		//lint:ignore ST1005 vanilla Grbl error string
		return errors.New("A G-code command was sent, but is missing some required P or L value words in the line")
	case 29:
		return errors.New("Grbl supports six work coordinate systems G54-G59. G59.1, G59.2, and G59.3 are not supported")
	case 30:
		//lint:ignore ST1005 vanilla Grbl error string
		return errors.New("The G53 G-code command requires either a G0 seek or G1 feed motion mode to be active. A different motion was active")
	case 31:
		//lint:ignore ST1005 vanilla Grbl error string
		return errors.New("There are unused axis words in the block and G80 motion mode cancel is active")
	case 32:
		//lint:ignore ST1005 vanilla Grbl error string
		return errors.New("A G2 or G3 arc was commanded but there are no XYZ axis words in the selected plane to trace the arc")
	case 33:
		//lint:ignore ST1005 vanilla Grbl error string
		return errors.New("The motion command has an invalid target. G2, G3, and G38.2 generates this error, if the arc is impossible to generate or if the probe target is the current position")
	case 34:
		//lint:ignore ST1005 vanilla Grbl error string
		return errors.New("A G2 or G3 arc, traced with the radius definition, had a mathematical error when computing the arc geometry. Try either breaking up the arc into semi-circles or quadrants, or redefine them with the arc offset definition")
	case 35:
		//lint:ignore ST1005 vanilla Grbl error string
		return errors.New("A G2 or G3 arc, traced with the offset definition, is missing the IJK offset word in the selected plane to trace the arc")
	case 36:
		//lint:ignore ST1005 vanilla Grbl error string
		return errors.New("There are unused, leftover G-code words that aren't used by any command in the block")
	case 37:
		//lint:ignore ST1005 vanilla Grbl error string
		return errors.New("The G43.1 dynamic tool length offset command cannot apply an offset to an axis other than its configured axis. The Grbl default axis is the Z-axis")
	case 38:
		//lint:ignore ST1005 vanilla Grbl error string
		return errors.New("Tool number greater than max supported value")
	default:
		return fmt.Errorf("unknown (%s)", m.Message)
	}
}

type MessagePushWelcome struct {
	Message string
}

func (m *MessagePushWelcome) Type() MessageType {
	return MessageTypePush
}

func (m *MessagePushWelcome) String() string {
	return m.Message
}

var messagePushAlarmPrefix = "ALARM:"

type MessagePushAlarm struct {
	Message string
}

func (m *MessagePushAlarm) Type() MessageType {
	return MessageTypePush
}

func (m *MessagePushAlarm) String() string {
	return m.Message
}

//gocyclo:ignore
func (m *MessagePushAlarm) Error() error {
	n, err := strconv.Atoi(m.Message[len(messagePushAlarmPrefix):])
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

func NewMessagePush(message string) (Message, error) {
	if strings.HasPrefix(message, "Grbl ") {
		return &MessagePushWelcome{Message: message}, nil
	}
	if strings.HasPrefix(message, messagePushAlarmPrefix) {
		return &MessagePushAlarm{Message: message}, nil
	}
	if strings.HasPrefix(message, "$") {
		return &MessagePushSettings{Message: message}, nil
	}
	if strings.HasPrefix(message, "[MSG:") {
		return &MessagePushFeedback{Message: message}, nil
	}
	if strings.HasPrefix(message, "[GC:") {
		return &MessagePushGcodeState{Message: message}, nil
	}
	if strings.HasPrefix(message, "[HLP:") {
		return &MessagePushHelp{Message: message}, nil
	}
	gcodeParamPrefixes := []string{
		"[G54:",
		"[G55:",
		"[G56:",
		"[G57:",
		"[G58:",
		"[G59:",
		"[G28:",
		"[G30:",
		"[G92:",
		"[TLO:",
		"[PRB:",
	}
	for _, prefix := range gcodeParamPrefixes {
		if strings.HasPrefix(message, prefix) {
			return &MessagePushGcodeParam{Message: message}, nil
		}
	}
	if strings.HasPrefix(message, "[VER:") {
		return &MessagePushBuildInfo{Message: message}, nil
	}
	if strings.HasPrefix(message, "[OPT:") {
		return &MessagePushCompileTimeOptions{Message: message}, nil
	}
	if strings.HasPrefix(message, ">") {
		return &MessagePushStartupLineExecution{Message: message}, nil
	}
	if strings.HasPrefix(message, "<") {
		return NewMessagePushStatusReport(message)
	}
	if strings.HasPrefix(message, "[echo:") {
		return &MessagePushEcho{Message: message}, nil
	}

	return &MessagePushUnknown{Message: message}, nil
}

type MessagePushSettings struct {
	Message string
}

func (m *MessagePushSettings) Type() MessageType {
	return MessageTypePush
}

func (m *MessagePushSettings) String() string {
	return m.Message
}

type MessagePushFeedback struct {
	Message string
}

func (m *MessagePushFeedback) Type() MessageType {
	return MessageTypePush
}

func (m *MessagePushFeedback) String() string {
	return m.Message
}

func (m *MessagePushFeedback) Text() string {
	return strings.TrimSuffix(strings.TrimPrefix(m.Message, "[MSG:"), "]")
}

type MessagePushGcodeState struct {
	Message string
}

func (m *MessagePushGcodeState) Type() MessageType {
	return MessageTypePush
}

func (m *MessagePushGcodeState) String() string {
	return m.Message
}

type MessagePushHelp struct {
	Message string
}

func (m *MessagePushHelp) Type() MessageType {
	return MessageTypePush
}

func (m *MessagePushHelp) String() string {
	return m.Message
}

type MessagePushGcodeParam struct {
	Message string
}

func (m *MessagePushGcodeParam) Type() MessageType {
	return MessageTypePush
}

func (m *MessagePushGcodeParam) String() string {
	return m.Message
}

type MessagePushBuildInfo struct {
	Message string
}

func (m *MessagePushBuildInfo) Type() MessageType {
	return MessageTypePush
}

func (m *MessagePushBuildInfo) String() string {
	return m.Message
}

type MessagePushCompileTimeOptions struct {
	Message string
}

func (m *MessagePushCompileTimeOptions) Type() MessageType {
	return MessageTypePush
}

func (m *MessagePushCompileTimeOptions) String() string {
	return m.Message
}

type MessagePushStartupLineExecution struct {
	Message string
}

func (m *MessagePushStartupLineExecution) Type() MessageType {
	return MessageTypePush
}

func (m *MessagePushStartupLineExecution) String() string {
	return m.Message
}

type StatusReportMachineState struct {
	// Valid states types:  `Idle, Run, Hold, Jog, Alarm, Door, Check, Home, Sleep`
	State string
	// Current sub-states are:
	// - `Hold:0` Hold complete. Ready to resume.
	// - `Hold:1` Hold in-progress. Reset will throw an alarm.
	// - `Door:0` Door closed. Ready to resume.
	// - `Door:1` Machine stopped. Door still ajar. Can't resume until closed.
	// - `Door:2` Door opened. Hold (or parking retract) in-progress. Reset will throw an alarm.
	// - `Door:3` Door closed and resuming. Restoring from park, if applicable. Reset will throw an alarm.
	SubState *int
}

func NewStatusReportMachineState(dataField string) (*StatusReportMachineState, error) {
	parts := strings.Split(dataField, ":")
	if len(parts) < 1 {
		return nil, fmt.Errorf("machine state field empty: %#v", dataField)
	}
	if len(parts) > 2 {
		return nil, fmt.Errorf("machine state field malformed: %#v", dataField)
	}
	state := parts[0]
	var subStatePtr *int
	if len(parts) == 2 {
		subState, err := strconv.Atoi(parts[1])
		if err != nil {
			return nil, fmt.Errorf("machine state substate invalid: %#v", dataField)
		}
		subStatePtr = &subState
	}
	return &StatusReportMachineState{
		State:    state,
		SubState: subStatePtr,
	}, nil
}

func (m *StatusReportMachineState) SubStateString() string {
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

type StatusReportMachinePosition struct {
	X float64
	Y float64
	Z float64
	A *float64
}

func NewStatusReportMachinePosition(dataValues []string) (*StatusReportMachinePosition, error) {
	machinePosition := &StatusReportMachinePosition{}

	if len(dataValues) < 3 || len(dataValues) > 4 {
		return nil, fmt.Errorf("machine position field malformed: %#v", dataValues)
	}

	var err error

	machinePosition.X, err = strconv.ParseFloat(dataValues[0], 64)
	if err != nil {
		return nil, fmt.Errorf("machine position X invalid: %#v", dataValues[0])
	}
	machinePosition.Y, err = strconv.ParseFloat(dataValues[1], 64)
	if err != nil {
		return nil, fmt.Errorf("machine position Y invalid: %#v", dataValues[1])
	}
	machinePosition.Z, err = strconv.ParseFloat(dataValues[2], 64)
	if err != nil {
		return nil, fmt.Errorf("machine position Z invalid: %#v", dataValues[2])
	}
	if len(dataValues) > 3 {
		a, err := strconv.ParseFloat(dataValues[3], 64)
		if err != nil {
			return nil, fmt.Errorf("machine position a invalid: %#v", dataValues[3])
		}
		machinePosition.A = &a
	}
	return machinePosition, nil
}

type StatusReportWorkPosition struct {
	X float64
	Y float64
	Z float64
	A *float64
}

func NewStatusReportWorkPosition(dataValues []string) (*StatusReportWorkPosition, error) {
	workPosition := &StatusReportWorkPosition{}

	if len(dataValues) < 3 || len(dataValues) > 4 {
		return nil, fmt.Errorf("work position field malformed: %#v", dataValues)
	}

	var err error

	workPosition.X, err = strconv.ParseFloat(dataValues[0], 64)
	if err != nil {
		return nil, fmt.Errorf("work position X invalid: %#v", dataValues[0])
	}
	workPosition.Y, err = strconv.ParseFloat(dataValues[1], 64)
	if err != nil {
		return nil, fmt.Errorf("work position Y invalid: %#v", dataValues[1])
	}
	workPosition.Z, err = strconv.ParseFloat(dataValues[2], 64)
	if err != nil {
		return nil, fmt.Errorf("work position Z invalid: %#v", dataValues[2])
	}
	if len(dataValues) > 3 {
		a, err := strconv.ParseFloat(dataValues[3], 64)
		if err != nil {
			return nil, fmt.Errorf("work position A invalid: %#v", dataValues[3])
		}
		workPosition.A = &a
	}
	return workPosition, nil
}

// Work coordinate offset is the current work coordinate offset of the g-code parser, which is the
// sum of the current work coordinate system, G92 offsets, and G43.1 tool length offset.
type StatusReportWorkCoordinateOffset struct {
	X float64
	Y float64
	Z float64
	A *float64
}

func NewStatusReportWorkCoordinateOffset(dataValues []string) (*StatusReportWorkCoordinateOffset, error) {
	wco := &StatusReportWorkCoordinateOffset{}

	if len(dataValues) < 3 || len(dataValues) > 4 {
		return nil, fmt.Errorf("work coordinate offset field malformed: %#v", dataValues)
	}

	var err error

	wco.X, err = strconv.ParseFloat(dataValues[0], 64)
	if err != nil {
		return nil, fmt.Errorf("work coordinate offset X invalid: %#v", dataValues[0])
	}
	wco.Y, err = strconv.ParseFloat(dataValues[1], 64)
	if err != nil {
		return nil, fmt.Errorf("work coordinate offset Y invalid: %#v", dataValues[1])
	}
	wco.Z, err = strconv.ParseFloat(dataValues[2], 64)
	if err != nil {
		return nil, fmt.Errorf("work coordinate offset Z invalid: %#v", dataValues[2])
	}
	if len(dataValues) > 3 {
		a, err := strconv.ParseFloat(dataValues[3], 64)
		if err != nil {
			return nil, fmt.Errorf("work coordinate offset A invalid: %#v", dataValues[3])
		}
		wco.A = &a
	}
	return wco, nil
}

type StatusReportBufferState struct {
	// Number of available blocks in the planner buffer
	AvailableBlocks int
	// Number of available bytes in the serial RX buffer
	AvailableBytes int
}

func NewStatusReportBufferState(dataValues []string) (*StatusReportBufferState, error) {
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

	return &StatusReportBufferState{
		AvailableBlocks: availableBlocks,
		AvailableBytes:  availableBytes,
	}, nil
}

// Line currently being executed
type StatusReportLineNumber int

func NewStatusReportLineNumber(dataValues []string) (*StatusReportLineNumber, error) {
	if len(dataValues) != 1 {
		return nil, fmt.Errorf("line number field malformed: %#v", dataValues)
	}

	lineNumber, err := strconv.Atoi(dataValues[0])
	if err != nil {
		return nil, fmt.Errorf("line number invalid: %#v", dataValues[0])
	}

	result := StatusReportLineNumber(lineNumber)
	return &result, nil
}

// Current Feed
type StatusReportFeed float64

func NewStatusReportFeed(dataValues []string) (*StatusReportFeed, error) {
	if len(dataValues) != 1 {
		return nil, fmt.Errorf("feed field malformed: %#v", dataValues)
	}

	feed, err := strconv.ParseFloat(dataValues[0], 64)
	if err != nil {
		return nil, fmt.Errorf("feed invalid: %#v", dataValues[0])
	}

	result := StatusReportFeed(feed)
	return &result, nil
}

// Current Feed and Speed
type StatusReportFeedSpindle struct {
	Feed  float64
	Speed float64
}

func NewStatusReportFeedSpindle(dataValues []string) (*StatusReportFeedSpindle, error) {
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

	return &StatusReportFeedSpindle{
		Feed:  feed,
		Speed: speed,
	}, nil
}

// Input pins Grbl has detected as 'triggered'.
type StatusReportPinState struct {
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

func NewStatusReportPinState(dataValues []string) (*StatusReportPinState, error) {
	if len(dataValues) != 1 {
		return nil, fmt.Errorf("pin state field malformed: %#v", dataValues)
	}

	pinState := &StatusReportPinState{}
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
func (p *StatusReportPinState) String() string {
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
type StatusReportOverrideValues struct {
	Feed    float64
	Rapids  float64
	Spindle float64
}

func NewStatusReportOverrideValues(dataValues []string) (*StatusReportOverrideValues, error) {
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

	return &StatusReportOverrideValues{
		Feed:    feed,
		Rapids:  rapids,
		Spindle: spindle,
	}, nil
}

type StatusReportAccessoryState struct {
	// indicates spindle is enabled in the CW direction. This does not appear with `C`.
	SpindleCW *bool
	// indicates spindle is enabled in the CCW direction. This does not appear with `S`.
	SpindleCCW *bool
	// indicates flood coolant is enabled.
	FloodCoolant *bool
	// indicates mist coolant is enabled.
	MistCoolant *bool
}

func NewStatusReportAccessoryState(dataValues []string) (*StatusReportAccessoryState, error) {
	if len(dataValues) != 1 {
		return nil, fmt.Errorf("accessory state field malformed: %#v", dataValues)
	}

	accessoryState := &StatusReportAccessoryState{}
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

type MessagePushStatusReport struct {
	Message              string
	MachineState         StatusReportMachineState
	MachinePosition      *StatusReportMachinePosition
	WorkPosition         *StatusReportWorkPosition
	WorkCoordinateOffset *StatusReportWorkCoordinateOffset
	BufferState          *StatusReportBufferState
	LineNumber           *StatusReportLineNumber
	Feed                 *StatusReportFeed
	FeedSpindle          *StatusReportFeedSpindle
	PinState             *StatusReportPinState
	OverrideValues       *StatusReportOverrideValues
	AccessoryState       *StatusReportAccessoryState
}

//gocyclo:ignore
func NewMessagePushStatusReport(message string) (*MessagePushStatusReport, error) {
	if !strings.HasPrefix(message, "<") {
		return nil, fmt.Errorf("status report message does not start with '<': %#v", message)
	}

	dataFields := strings.Split(message[1:len(message)-1], "|")
	if len(dataFields) < 2 {
		return nil, fmt.Errorf("status report message missing required data fields: %#v", message)
	}

	machineState, err := NewStatusReportMachineState(dataFields[0])
	if err != nil {
		return nil, fmt.Errorf("status report message parsing failed: %#v: %w", message, err)
	}

	messagePushStatusReport := &MessagePushStatusReport{
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
			messagePushStatusReport.MachinePosition, err = NewStatusReportMachinePosition(dataValues)
			if err != nil {
				return nil, fmt.Errorf("status report message: failed to parse MPos: %w", err)
			}
		case "WPos":
			messagePushStatusReport.WorkPosition, err = NewStatusReportWorkPosition(dataValues)
			if err != nil {
				return nil, fmt.Errorf("status report message: failed to parse WPos: %w", err)
			}
		case "WCO":
			messagePushStatusReport.WorkCoordinateOffset, err = NewStatusReportWorkCoordinateOffset(dataValues)
			if err != nil {
				return nil, fmt.Errorf("status report message: failed to parse WCO: %w", err)
			}
		case "Bf":
			messagePushStatusReport.BufferState, err = NewStatusReportBufferState(dataValues)
			if err != nil {
				return nil, fmt.Errorf("status report message: failed to parse Bf: %w", err)
			}
		case "Ln":
			messagePushStatusReport.LineNumber, err = NewStatusReportLineNumber(dataValues)
			if err != nil {
				return nil, fmt.Errorf("status report message: failed to parse Ln: %w", err)
			}
		case "F":
			messagePushStatusReport.Feed, err = NewStatusReportFeed(dataValues)
			if err != nil {
				return nil, fmt.Errorf("status report message: failed to parse F: %w", err)
			}
		case "FS":
			messagePushStatusReport.FeedSpindle, err = NewStatusReportFeedSpindle(dataValues)
			if err != nil {
				return nil, fmt.Errorf("status report message: failed to parse FS: %w", err)
			}
		case "Pn":
			messagePushStatusReport.PinState, err = NewStatusReportPinState(dataValues)
			if err != nil {
				return nil, fmt.Errorf("status report message: failed to parse Pn: %w", err)
			}
		case "Ov":
			messagePushStatusReport.OverrideValues, err = NewStatusReportOverrideValues(dataValues)
			if err != nil {
				return nil, fmt.Errorf("status report message: failed to parse Ov: %w", err)
			}
		case "A":
			messagePushStatusReport.AccessoryState, err = NewStatusReportAccessoryState(dataValues)
			if err != nil {
				return nil, fmt.Errorf("status report message: failed to parse A: %w", err)
			}
		}
	}

	if !strings.HasPrefix(message, "<") {
		return nil, fmt.Errorf("status report message does not end with '>': %#v", message)
	}

	return messagePushStatusReport, nil
}

func (m *MessagePushStatusReport) Type() MessageType {
	return MessageTypePush
}

func (m *MessagePushStatusReport) String() string {
	return m.Message
}

type MessagePushEcho struct {
	Message string
}

func (m *MessagePushEcho) Type() MessageType {
	return MessageTypePush
}

func (m *MessagePushEcho) String() string {
	return m.Message
}

type MessagePushUnknown struct {
	Message string
}

func (m *MessagePushUnknown) Type() MessageType {
	return MessageTypePush
}

func (m *MessagePushUnknown) String() string {
	return m.Message
}

func NewMessage(message string) (Message, error) {
	if message == messageResponseOk || strings.HasPrefix(message, messageResponseErrorPrefix) {
		return &MessageResponse{
			Message: message,
		}, nil
	}
	return NewMessagePush(message)
}
