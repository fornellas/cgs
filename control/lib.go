package control

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"

	"github.com/gdamore/tcell/v2"

	grblMod "github.com/fornellas/cgs/grbl"
)

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
	var floatStr string
	if decimal > 0 {
		floatFormat := fmt.Sprintf("%%.%df", decimal)
		floatStr = fmt.Sprintf(floatFormat, value)
		floatStr = strings.TrimRight(strings.TrimRight(floatStr, "0"), ".")
	} else {
		floatStr = fmt.Sprintf("%.0f", value)
	}
	return fmt.Sprintf("[%s]%s[-]", tcell.ColorOrange, floatStr)
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

func sprintBool(value bool) string {
	color := tcell.ColorGreen
	if !value {
		color = tcell.ColorRed
	}
	return fmt.Sprintf("[%s]%v[-]", color, value)
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

func getMachineStateColor(state string) tcell.Color {
	switch state {
	case "Idle":
		return tcell.ColorBlack
	case "Run":
		return tcell.ColorGreen
	case "Hold":
		return tcell.ColorYellow
	case "Jog":
		return tcell.ColorDarkGreen
	case "Alarm":
		return tcell.ColorRed
	case "Door":
		return tcell.ColorOrange
	case "Check":
		return tcell.ColorDarkCyan
	case "Home":
		return tcell.ColorLightGreen
	case "Sleep":
		return tcell.ColorDarkBlue
	default:
		return tcell.ColorWhite
	}
}
