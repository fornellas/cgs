package grbl

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/fornellas/cgs/gcode"
)

type Coordinates struct {
	X float64
	Y float64
	Z float64
	A *float64
}

// NewCoordinates creates a Coordinates string values for X, Y, Z and A (optional).
func NewCoordinatesFromStrValues(dataValues []string) (*Coordinates, error) {
	coordinates := &Coordinates{}

	if len(dataValues) < 3 || len(dataValues) > 4 {
		return nil, fmt.Errorf("machine position field malformed: %#v", dataValues)
	}

	var err error

	coordinates.X, err = strconv.ParseFloat(dataValues[0], 64)
	if err != nil {
		return nil, fmt.Errorf("machine position X invalid: %#v", dataValues[0])
	}
	coordinates.Y, err = strconv.ParseFloat(dataValues[1], 64)
	if err != nil {
		return nil, fmt.Errorf("machine position Y invalid: %#v", dataValues[1])
	}
	coordinates.Z, err = strconv.ParseFloat(dataValues[2], 64)
	if err != nil {
		return nil, fmt.Errorf("machine position Z invalid: %#v", dataValues[2])
	}
	if len(dataValues) > 3 {
		a, err := strconv.ParseFloat(dataValues[3], 64)
		if err != nil {
			return nil, fmt.Errorf("machine position a invalid: %#v", dataValues[3])
		}
		coordinates.A = &a
	}
	return coordinates, nil
}

// NewCoordinates creates a Coordinates struct from a string CSV: X,Y,Z,A (A is optional)
func NewCoordinatesFromCSV(s string) (*Coordinates, error) {
	return NewCoordinatesFromStrValues(strings.Split(s, ","))
}

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
		return NewMessagePushGcodeState(message)
	}
	if strings.HasPrefix(message, "[HLP:") {
		return &MessagePushHelp{Message: message}, nil
	}

	for prefix := range gcodeParamPrefixes {
		if strings.HasPrefix(message, prefix) {
			return NewMessagePushGcodeParam(message)
		}
	}
	if strings.HasPrefix(message, "[VER:") {
		return NewMessagePushBuildInfo(message)
	}
	if strings.HasPrefix(message, "[OPT:") {
		return NewMessagePushCompileTimeOptions(message)
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
	if len(message) == 0 {
		return &MessagePushEmpty{}, nil
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
	Message      string
	ModalGroup   *gcode.ModalGroup
	Tool         *float64
	SpindleSpeed *float64
	FeedRate     *float64
}

func NewMessagePushGcodeState(message string) (*MessagePushGcodeState, error) {
	m := &MessagePushGcodeState{
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

// Updates GcodeParameters from given MessagePushGcodeParam.
func (g *GcodeParameters) Update(messagePushGcodeParam *MessagePushGcodeParam) {
	if messagePushGcodeParam.GcodeParameters.CoordinateSystem1 != nil {
		g.CoordinateSystem1 = messagePushGcodeParam.GcodeParameters.CoordinateSystem1
	}
	if messagePushGcodeParam.GcodeParameters.CoordinateSystem2 != nil {
		g.CoordinateSystem2 = messagePushGcodeParam.GcodeParameters.CoordinateSystem2
	}
	if messagePushGcodeParam.GcodeParameters.CoordinateSystem3 != nil {
		g.CoordinateSystem3 = messagePushGcodeParam.GcodeParameters.CoordinateSystem3
	}
	if messagePushGcodeParam.GcodeParameters.CoordinateSystem4 != nil {
		g.CoordinateSystem4 = messagePushGcodeParam.GcodeParameters.CoordinateSystem4
	}
	if messagePushGcodeParam.GcodeParameters.CoordinateSystem5 != nil {
		g.CoordinateSystem5 = messagePushGcodeParam.GcodeParameters.CoordinateSystem5
	}
	if messagePushGcodeParam.GcodeParameters.CoordinateSystem6 != nil {
		g.CoordinateSystem6 = messagePushGcodeParam.GcodeParameters.CoordinateSystem6
	}
	if messagePushGcodeParam.GcodeParameters.PrimaryPreDefinedPosition != nil {
		g.PrimaryPreDefinedPosition = messagePushGcodeParam.GcodeParameters.PrimaryPreDefinedPosition
	}
	if messagePushGcodeParam.GcodeParameters.SecondaryPreDefinedPosition != nil {
		g.SecondaryPreDefinedPosition = messagePushGcodeParam.GcodeParameters.SecondaryPreDefinedPosition
	}
	if messagePushGcodeParam.GcodeParameters.CoordinateOffset != nil {
		g.CoordinateOffset = messagePushGcodeParam.GcodeParameters.CoordinateOffset
	}
	if messagePushGcodeParam.GcodeParameters.ToolLengthOffset != nil {
		g.ToolLengthOffset = messagePushGcodeParam.GcodeParameters.ToolLengthOffset
	}
	if messagePushGcodeParam.GcodeParameters.Probe != nil {
		g.Probe = messagePushGcodeParam.GcodeParameters.Probe
	}
}

type MessagePushGcodeParam struct {
	Message         string
	GcodeParameters GcodeParameters
}

//gocyclo:ignore
func NewMessagePushGcodeParam(message string) (*MessagePushGcodeParam, error) {
	m := &MessagePushGcodeParam{
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

func (m *MessagePushGcodeParam) Type() MessageType {
	return MessageTypePush
}

func (m *MessagePushGcodeParam) String() string {
	return m.Message
}

type MessagePushBuildInfo struct {
	Message string
	Version string
	Id      string
}

func NewMessagePushBuildInfo(message string) (*MessagePushBuildInfo, error) {
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
	return &MessagePushBuildInfo{
		Message: message,
		Version: parts[0],
		Id:      parts[1],
	}, nil
}

func (m *MessagePushBuildInfo) Type() MessageType {
	return MessageTypePush
}

func (m *MessagePushBuildInfo) String() string {
	return m.Message
}

type MessagePushCompileTimeOptions struct {
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

func NewMessagePushCompileTimeOptions(message string) (*MessagePushCompileTimeOptions, error) {
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
	if serialRxBufferBytes, err = strconv.ParseUint(parts[1], 10, 64); err != nil {
		return nil, fmt.Errorf("unable to parse serial RX buffer bytes: %#v: %w", message, err)
	}
	return &MessagePushCompileTimeOptions{
		Message:             message,
		CompileTimeOptions:  compileTimeOptions,
		PlannerBlocks:       plannerBlocks,
		SerialRxBufferBytes: serialRxBufferBytes,
	}, nil
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

type StatusReportMachinePosition Coordinates

func NewStatusReportMachinePosition(dataValues []string) (*StatusReportMachinePosition, error) {
	coordinates, err := NewCoordinatesFromStrValues(dataValues)
	if err != nil {
		return nil, err
	}
	return (*StatusReportMachinePosition)(coordinates), nil
}

type StatusReportWorkPosition Coordinates

func NewStatusReportWorkPosition(dataValues []string) (*StatusReportWorkPosition, error) {
	coordinates, err := NewCoordinatesFromStrValues(dataValues)
	if err != nil {
		return nil, err
	}
	return (*StatusReportWorkPosition)(coordinates), nil
}

// Work coordinate offset is the current work coordinate offset of the g-code parser, which is the
// sum of the current work coordinate system, G92 offsets, and G43.1 tool length offset.
type StatusReportWorkCoordinateOffset Coordinates

func NewStatusReportWorkCoordinateOffset(dataValues []string) (*StatusReportWorkCoordinateOffset, error) {
	coordinates, err := NewCoordinatesFromStrValues(dataValues)
	if err != nil {
		return nil, err
	}
	return (*StatusReportWorkCoordinateOffset)(coordinates), nil
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

func (o *StatusReportOverrideValues) HasOverride() bool {
	if o.Feed != 100 || o.Rapids != 100 || o.Spindle != 100 {
		return true
	}
	return false
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

type MessagePushEmpty struct {
}

func (m *MessagePushEmpty) Type() MessageType {
	return MessageTypePush
}

func (m *MessagePushEmpty) String() string {
	return "(empty)"
}

type MessagePushUnknown struct {
	Message string
}

func (m *MessagePushUnknown) Type() MessageType {
	return MessageTypePush
}

func (m *MessagePushUnknown) String() string {
	return fmt.Sprintf("Unknown message: %s", m.Message)
}

func NewMessage(message string) (Message, error) {
	if message == messageResponseOk || strings.HasPrefix(message, messageResponseErrorPrefix) {
		return &MessageResponse{
			Message: message,
		}, nil
	}
	return NewMessagePush(message)
}
