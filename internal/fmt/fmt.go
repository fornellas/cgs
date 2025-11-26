package fmt

import (
	"fmt"
	"strings"
)

func SprintFloat(value float64, decimal uint) string {
	var floatStr string
	if decimal > 0 {
		floatFormat := fmt.Sprintf("%%.%df", decimal)
		floatStr = fmt.Sprintf(floatFormat, value)
		floatStr = strings.TrimRight(strings.TrimRight(floatStr, "0"), ".")
	} else {
		floatStr = fmt.Sprintf("%.0f", value)
	}
	return floatStr
}
