package control

import (
	"bytes"
	"fmt"
	"strconv"

	"github.com/gdamore/tcell/v2"

	grblMod "github.com/fornellas/cgs/grbl"
	iFmt "github.com/fornellas/cgs/internal/fmt"
)

var gcodeColor = tcell.NewRGBColor(111, 170, 210)

const coordinateWidth = len("20000.0000") + 1
const feedWidth = len("20000") + 1

func acceptFloat(textToCheck string, lastChar rune) bool {
	if len(textToCheck) > 0 && textToCheck[0] == '-' {
		return true
	}
	_, err := strconv.ParseFloat(textToCheck, 64)
	return err == nil
}

func acceptUFloat(textToCheck string, lastChar rune) bool {
	_, err := strconv.ParseFloat(textToCheck, 64)
	return err == nil
}

func sprintFloat(value float64, decimal uint) string {
	return fmt.Sprintf("[%s]%s[-]", tcell.ColorOrange, iFmt.SprintFloat(value, decimal))
}

func sprintCoordinate(value float64) string {
	return sprintFloat(value, 4)
}

func sprintCoordinatesSingleLine(coordinates *grblMod.Coordinates, sep string) string {
	var buf bytes.Buffer
	fmt.Fprintf(&buf, "X:%s%sY:%s%sZ:%s", sprintCoordinate(coordinates.X), sep, sprintCoordinate(coordinates.Y), sep, sprintCoordinate(coordinates.Z))
	if coordinates.A != nil {
		fmt.Fprintf(&buf, " %sA:%.4f", sep, *coordinates.A)
	}
	return buf.String()
}

func sprintBlocks(value int) string {
	return fmt.Sprintf("[%s]%d[-]", tcell.ColorOrange, value)
}

func sprintBytes(value int) string {
	return fmt.Sprintf("[%s]%d[-]", tcell.ColorOrange, value)
}

func sprintSpindle(value float64) string {
	return sprintFloat(value, 0)
}

func sprintLine(value int) string {
	return fmt.Sprintf("[%s]%d[-]", tcell.ColorOrange, value)
}

func sprintFeed(value float64) string {
	return sprintFloat(value, 0)
}

func sprintSpeed(value float64) string {
	return sprintFloat(value, 0)
}

func sprintTool(value float64) string {
	return sprintFloat(value, 0)
}

func sprintGcodeWord(word string) string {
	return fmt.Sprintf("[%s]%s[-]", tcell.ColorBlue, word)
}

func getMachineStateColor(state grblMod.State) tcell.Color {
	switch state {
	case grblMod.StateIdle:
		return tcell.ColorBlack
	case grblMod.StateRun:
		return tcell.ColorGreen
	case grblMod.StateHold:
		return tcell.ColorYellow
	case grblMod.StateJog:
		return tcell.ColorDarkGreen
	case grblMod.StateAlarm:
		return tcell.ColorRed
	case grblMod.StateDoor:
		return tcell.ColorOrange
	case grblMod.StateCheck:
		return tcell.ColorDarkCyan
	case grblMod.StateHome:
		return tcell.ColorLightGreen
	case grblMod.StateSleep:
		return tcell.ColorDarkBlue
	default:
		return tcell.ColorWhite
	}
}
