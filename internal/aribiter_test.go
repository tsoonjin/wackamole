package internal_test

import (
	"github.com/gorilla/websocket"
	. "github.com/tsoonjin/wackamole/internal"
	"reflect"
	"testing"
)

func TestInitSession(t *testing.T) {
	t.Parallel()
	dummyConn := websocket.Conn{}
	want := "string"
	session := InitSession(&dummyConn)
	got := reflect.TypeOf(session.Name).String()
	if want != got {
		t.Errorf("Session name should be of type %s, got %s", want, got)
	}
}
