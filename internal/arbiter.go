package internal

// Manages incoming connections
import (
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

func (s *Session) parseCommand(msg string, rooms *map[string]*Game) {
	splittedMsg := strings.Split(msg, " ")
	command := strings.TrimRight(splittedMsg[0], "\n")
	args := splittedMsg[1:]
	switch command {
	case "/join":
		roomName := args[0]
		log.Printf("Player %s wants to join a game, %s", s.Name, roomName)
		if s.room != nil {
			log.Printf("Player %s unable to join a game till current game is over", s.Name)
		}
		if game, ok := (*rooms)[roomName]; ok {
			s.room = game
			game.AddPlayer(s.Id, s)
			return
		}
		newGame, err := CreateGameV2(roomName, 2, 2, []string{s.Id}, time.NewTicker(time.Second), []*Session{s})
		if err != nil {
			s.out <- []byte("Failed to create a new game room")
		}
		log.Printf("New game room created: %s", roomName)
		(*rooms)[roomName] = newGame
		s.room = newGame
	case "/ready":
		log.Printf("Player %s is ready to rumble in %s", s.Name, s.room.Id)
		s.room.AddPlayerReady(s.Id)
	default:
		if s.room.state == Running {
			s.room.AddAction(time.Now().Unix(), s.Id, msg)
		}

	}
}

func (s *Session) Run(interupt chan os.Signal, rooms *map[string]*Game) {
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
			s.parseCommand(string(msgFromClient), rooms)
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
