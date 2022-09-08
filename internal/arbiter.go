package internal

// Manages incoming connections
import (
	"errors"
	"github.com/cip8/autoname"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"log"
	"os"
	"strings"
	"time"
)

var channel_buffer int = 256

type Command interface {
	execute()
}

type Session struct {
	Id   string
	Name string
	conn *websocket.Conn
	in   chan []byte
	out  chan []byte
	room *Game
}

func InitSession(conn *websocket.Conn) Session {
	return Session{
		Id:   uuid.New().String(),
		Name: autoname.Generate(""),
		conn: conn,
		in:   make(chan []byte, channel_buffer),
		out:  make(chan []byte, channel_buffer),
	}
}

// Allow user to reconnect to existing session
func (s *Session) Reconnect(conn *websocket.Conn) {
	s.conn = conn
	s.in = make(chan []byte)
	s.out = make(chan []byte)
}

func joinGame(s *Session, args []string) (*Game, error) {
	if s.room != nil {
		s.in <- []byte("Cannot join another game till current game ends")
	}
	roomName := ClearString(args[0])
	newGame, err := CreateGameV2(roomName, 2, 2, []string{s.Id}, time.NewTicker(time.Second), []*websocket.Conn{s.conn})
	if err != nil {
		return nil, errors.New("failed to join game")
	}
	return newGame, nil
}

func (s *Session) parseCommand(msg string) {
	splittedMsg := strings.Split(msg, " ")
	command := splittedMsg[0]
	args := splittedMsg[1:]
	switch command {
	case "/join":
		log.Println("Join a game room")
		joinGame(s, args)
	case "/ready":
		log.Printf("Player %s is ready to rumble in %s", s.Name, s.room.Id)
	default:
		log.Println("No matching command")

	}
}

func (s *Session) Run(interupt chan os.Signal, rooms map[string]*Game) {
	// Handle socket connection with client
	go func() {
		defer s.conn.Close()
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
			s.parseCommand((string(msgFromClient)))
			s.out <- []byte("Acked")

		case msgToClient := <-s.out:
			if err := s.conn.WriteMessage(websocket.TextMessage, msgToClient); err != nil {
				log.Println("Error writing to client")
			}

		case <-interupt:
			log.Println("Interupted by user")
			err := s.conn.WriteMessage(websocket.CloseMessage,
				websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				log.Println("Write close:", err)
				return
			}
			return
		}
	}
}
