package internal

import (
	"fmt"
)

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
