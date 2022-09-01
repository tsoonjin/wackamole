package internal

import (
	"fmt"
	"regexp"
)

var nonAlphanumericRegex = regexp.MustCompile(`[^a-zA-Z0-9 ]+`)

func ClearString(str string) string {
	return nonAlphanumericRegex.ReplaceAllString(str, "")
}

func FormatMsg(s string, t string) string {
	switch t {
	case "client":
		return fmt.Sprintf("[me]: %s", s)
	case "server":
		return fmt.Sprintf("[server]: %s", s)
	default:
		return s
	}
}
func contains(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}

	return false
}
