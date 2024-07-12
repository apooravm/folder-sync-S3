package utils

import (
	"fmt"
	"log"
	"strings"
)

// Print colored error text
func LogColourSprintf(message string, colour string, endL bool) string {
	finalChar := ""
	if endL {
		finalChar = "\n"
	}

	switch colour {
	case "red":
		return fmt.Sprintf("\x1b[31m%s\x1b[0m%s", message, finalChar)

	case "yellow":
		return fmt.Sprintf("\x1b[33m%s\x1b[0m%s", message, finalChar)

	case "green":
		return fmt.Sprintf("\x1b[32m%s\x1b[0m%s", message, finalChar)

	case "magenta":
		return fmt.Sprintf("\x1b[35m%s\x1b[0m%s", message, finalChar)

	case "cyan":
		return fmt.Sprintf("\x1b[36m%s\x1b[0m%s", message, finalChar)

	case "blue":
		return fmt.Sprintf("\x1b[34m%s\x1b[0m%s", message, finalChar)

	default:
		return fmt.Sprintf("%s%s", message, finalChar)
	}
}

func LogColourPrint(colour string, endL bool, message ...string) {
	joinedMessage := strings.Join(message, "")
	log.Println(LogColourSprintf(joinedMessage, colour, endL))
}

func ColourPrint(message string, colour string) {
	fmt.Println(LogColourSprintf(message, colour, false))
}
