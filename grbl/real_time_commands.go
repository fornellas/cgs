package grbl

import (
	"errors"
	"fmt"
)

var ErrNotRealTimeCommand = errors.New("not a real time command")

type RealTimeCommand byte

var realTimeCommandStringsMap = map[RealTimeCommand]string{
	RealTimeCommandSoftReset:                                          "Soft-Reset",
	RealTimeCommandStatusReportQuery:                                  "Status Report Query",
	RealTimeCommandCycleStartResume:                                   "Cycle Start / Resume",
	RealTimeCommandFeedHold:                                           "Feed Hold",
	RealTimeCommandSafetyDoor:                                         "Safety Door",
	RealTimeCommandJogCancel:                                          "Jog Cancel",
	RealTimeCommandFeedOverrideSet100OfProgrammedRate:                 "Feed Override: Set 100% of programmed rate.",
	RealTimeCommandFeedOverrideIncrease10:                             "Feed Override: Increase 10%",
	RealTimeCommandFeedOverrideDecrease10:                             "Feed Override: Decrease 10%",
	RealTimeCommandFeedOverrideIncrease1:                              "Feed Override: Increase 1%",
	RealTimeCommandFeedOverrideDecrease1:                              "Feed Override: Decrease 1%",
	RealTimeCommandRapidOverrideSetTo100FullRapidRate:                 "Rapid Override: Set to 100% full rapid rate.",
	RealTimeCommandRapidOverrideSetTo50OfRapidRate:                    "Rapid Override: Set to 50% of rapid rate.",
	RealTimeCommandRapidOverrideSetTo25OfRapidRate:                    "Rapid Override: Set to 25% of rapid rate.",
	RealTimeCommandSpindleSpeedOverrideSet100OfProgrammedSpindleSpeed: "Spindle Speed Override: Set 100% of programmed spindle speed",
	RealTimeCommandSpindleSpeedOverrideIncrease10:                     "Spindle Speed Override: Increase 10%",
	RealTimeCommandSpindleSpeedOverrideDecrease10:                     "Spindle Speed Override: Decrease 10%",
	RealTimeCommandSpindleSpeedOverrideIncrease1:                      "Spindle Speed Override: Increase 1%",
	RealTimeCommandSpindleSpeedOverrideDecrease1:                      "Spindle Speed Override: Decrease 1%",
	RealTimeCommandToggleSpindleStop:                                  "Toggle Spindle Stop",
	RealTimeCommandToggleFloodCoolant:                                 "Toggle Flood Coolant",
	RealTimeCommandToggleMistCoolant:                                  "Toggle Mist Coolant",
}

func NewRealTimeCommand(b byte) (RealTimeCommand, error) {
	rtc := RealTimeCommand(b)
	if _, ok := realTimeCommandStringsMap[rtc]; ok {
		return rtc, nil
	}
	return 0, ErrNotRealTimeCommand
}

func (c RealTimeCommand) String() string {
	if str, ok := realTimeCommandStringsMap[c]; ok {
		return str
	}
	return fmt.Sprintf("Unknown (%#v)", c)
}

var (
	// Soft-Reset
	RealTimeCommandSoftReset RealTimeCommand = 0x18
	// Status Report Query
	RealTimeCommandStatusReportQuery RealTimeCommand = '?'
	// Cycle Start / Resume
	RealTimeCommandCycleStartResume RealTimeCommand = '~'
	// Feed Hold
	RealTimeCommandFeedHold RealTimeCommand = '!'
	// Safety Door
	RealTimeCommandSafetyDoor RealTimeCommand = 0x84
	// Jog Cancel
	RealTimeCommandJogCancel RealTimeCommand = 0x85
	// Feed Override: Set 100% of programmed rate.
	RealTimeCommandFeedOverrideSet100OfProgrammedRate RealTimeCommand = 0x90
	// Feed Override: Increase 10%
	RealTimeCommandFeedOverrideIncrease10 RealTimeCommand = 0x91
	// Feed Override: Decrease 10%
	RealTimeCommandFeedOverrideDecrease10 RealTimeCommand = 0x92
	// Feed Override: Increase 1%
	RealTimeCommandFeedOverrideIncrease1 RealTimeCommand = 0x93
	// Feed Override: Decrease 1%
	RealTimeCommandFeedOverrideDecrease1 RealTimeCommand = 0x94
	// Rapid Override: Set to 100% full rapid rate.
	RealTimeCommandRapidOverrideSetTo100FullRapidRate RealTimeCommand = 0x95
	// Rapid Override: Set to 50% of rapid rate.
	RealTimeCommandRapidOverrideSetTo50OfRapidRate RealTimeCommand = 0x96
	// Rapid Override: Set to 25% of rapid rate.
	RealTimeCommandRapidOverrideSetTo25OfRapidRate RealTimeCommand = 0x97
	// Spindle Speed Override: Set 100% of programmed spindle speed
	RealTimeCommandSpindleSpeedOverrideSet100OfProgrammedSpindleSpeed RealTimeCommand = 0x99
	// Spindle Speed Override: Increase 10%
	RealTimeCommandSpindleSpeedOverrideIncrease10 RealTimeCommand = 0x9A
	// Spindle Speed Override: Decrease 10%
	RealTimeCommandSpindleSpeedOverrideDecrease10 RealTimeCommand = 0x9B
	// Spindle Speed Override: Increase 1%
	RealTimeCommandSpindleSpeedOverrideIncrease1 RealTimeCommand = 0x9C
	// Spindle Speed Override: Decrease 1%
	RealTimeCommandSpindleSpeedOverrideDecrease1 RealTimeCommand = 0x9D
	// Toggle Spindle Stop
	RealTimeCommandToggleSpindleStop RealTimeCommand = 0x9E
	// Toggle Flood Coolant
	RealTimeCommandToggleFloodCoolant RealTimeCommand = 0xA0
	// Toggle Mist Coolant
	RealTimeCommandToggleMistCoolant RealTimeCommand = 0xA1
)
