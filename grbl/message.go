package grbl

import (
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

type MessagePush struct {
	Message string
}

func (m *MessagePush) Type() MessageType {
	return MessageTypePush
}

func (m *MessagePush) String() string {
	return m.Message
}

// Welcome Message
//   Grbl X.Xx ['$' for help] : Welcome message; indicates initialization.
// Alarm Message
//   ALARM:x : Indicates an alarm has been thrown. Grbl is now in an alarm state.
//     1	Hard limit triggered. Machine position is likely lost due to sudden and immediate halt. Re-homing is highly recommended.
//     2	G-code motion target exceeds machine travel. Machine position safely retained. Alarm may be unlocked.
//     3	Reset while in motion. Grbl cannot guarantee position. Lost steps are likely. Re-homing is highly recommended.
//     4	Probe fail. The probe is not in the expected initial state before starting probe cycle, where G38.2 and G38.3 is not triggered and G38.4 and G38.5 is triggered.
//     5	Probe fail. Probe did not contact the workpiece within the programmed travel for G38.2 and G38.4.
//     6	Homing fail. Reset during active homing cycle.
//     7	Homing fail. Safety door was opened during active homing cycle.
//     8	Homing fail. Cycle failed to clear limit switch when pulling off. Try increasing pull-off setting or check wiring.
//     9	Homing fail. Could not find limit switch within search distance. Defined as 1.5 * max_travel on search and 5 * pulloff on locate phases.
//     10	Homing fail. On dual axis machines, could not find the second limit switch for self-squaring.
// Grbl $ Settings Message
//   $x=val and $Nx=line indicate a settings printout from a $ and $N user query, respectively.
// Feedback Messages
//   Non-Queried Feedback Messages
//     [MSG:] : Indicates a non-queried feedback message.
// Queried Feedback Messages
//   [GC:] : Indicates a queried $G g-code state message.
//   [HLP:] : Indicates the help message.
//   [G54:], [G55:], [G56:], [G57:], [G58:], [G59:], [G28:], [G30:], [G92:], [TLO:], and [PRB:] messages indicate the parameter data printout from a $# user query.
//   [VER:] : Indicates build info and string from a $I user query.
//   [OPT:] line follows immediately after and contains character codes for compile-time options that were either enabled or disabled.
// Startup Line Execution
//   >G54G20:ok : The open chevron indicates startup line execution. The :ok suffix shows it executed correctly without adding an unmatched ok response on a new line.
// Real-time Status Reports
//   < > : Enclosed between chevrons. Contains status report data.
// Debugging
//   [echo:] : Indicates an automated line echo from a pre-parsed string prior to g-code parsing. Enabled by config.h option.

func NewMessage(message string) Message {
	if message == messageResponseOk {
		return &MessageResponse{
			Message: message,
		}
	}
	if strings.HasPrefix(message, messageResponseErrorPrefix) {
		return &MessageResponse{
			Message: message,
		}
	}
	return &MessagePush{
		Message: message,
	}
}
