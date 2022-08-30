package internal_test

import (
	. "github.com/tsoonjin/wackamole/internal"
	"testing"
)

func TestFormatMsg(t *testing.T) {
	t.Parallel()
	var want string = "[me]: hello"
	got := FormatMsg("hello", "client")
	if want != got {
		t.Errorf("want %s, got %s", want, got)
	}
}
