package internal

// Manages incoming connections
import (
	"github.com/cip8/autoname"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"log"
	"os"
)

var channel_buffer int = 10

type Session struct {
	Id   string
	Name string
	conn *websocket.Conn
	in   chan []byte
	out  chan []byte
}

func InitSession(conn *websocket.Conn) Session {
	return Session{
		Id:   uuid.New().String(),
		Name: autoname.Generate(""),
		conn: conn,
		in:   make(chan []byte),
		out:  make(chan []byte),
	}
}

// Allow user to reconnect to existing session
func (s *Session) Reconnect(conn *websocket.Conn) {
	s.conn = conn
	s.in = make(chan []byte)
	s.out = make(chan []byte)
}

func (s *Session) Run(interupt chan os.Signal) {
	defer close(s.out)
	// Handle socket connection with client
	go func() {
		for {
			_, message, err := s.conn.ReadMessage()
			if err != nil {
				log.Println("Error reading message from socket conn: ", err)
				return
			}
			s.in <- message
		}
	}()

	for {
		select {
		case msgFromClient := <-s.in:
			log.Printf("%s <<: %s", s.Name, string(msgFromClient))
			s.out <- []byte("Acked")

		case msgToClient := <-s.out:
			if err := s.conn.WriteMessage(websocket.TextMessage, msgToClient); err != nil {
				log.Println("Error writing to client")
			}
			log.Printf("%s >>: %s", s.Name, string(msgToClient))

		case <-interupt:
			log.Println("Interupted by user")
			err := s.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				log.Println("Write close:", err)
				return
			}
			return
		}
	}
}
