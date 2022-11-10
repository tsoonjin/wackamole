package internal

// Manages incoming connections
import (
	"encoding/json"
	"fmt"
	"github.com/cip8/autoname"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"log"
	"os"
	"strings"
	"time"
)

var channel_buffer int = 256

var clientKeyMap = map[int]string{
	0: "w",
	1: "e",
	2: "r",
	3: "s",
	4: "d",
	5: "f",
	6: "x",
	7: "c",
	8: "v",
}

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

type SocketPayload struct {
	Name     string `json:"name"`
	RoomName string `json:"roomName"`
	Hit      int    `json:"hit"`
}

type SocketRequest struct {
	Command string        `json:"command"`
	Payload SocketPayload `json:"payload"`
}

type GameRoomStream struct {
	Name           string   `json:"name"`
	IsPrivate      bool     `json:"isPrivate"`
	State          string   `json:"state"`
	ReadyPlayers   []string `json:"readyPlayers"`
	WaitingPlayers []string `json:"waitingPlayers"`
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
	socketRequest := SocketRequest{}
	json.Unmarshal([]byte(msg), &socketRequest)
	log.Println("Parsed", socketRequest)
	splittedMsg := strings.Split(msg, " ")
	command := strings.TrimRight(splittedMsg[0], "\n")
	args := splittedMsg[1:]
	switch socketRequest.Command {
	case "init":
		runningState := GameState{"running"}
		if s.room.state != runningState {
			s.room.AddPlayer("Bob", s)
			s.room.AddPlayer("Alice", s)
			s.room.state = GameState{"running"}
			newBoard := s.room.initGameBoard()
			s.room.board = newBoard
			s.room.startTime = time.Now()
		}
	case "send":
		runningState := GameState{"running"}
		if s.room.state != runningState {
			s.room.AddPlayer("Bob", s)
			s.room.AddPlayer("Alice", s)
			s.room.state = GameState{"running"}
			newBoard := s.room.initGameBoard()
			s.room.board = newBoard
			s.room.startTime = time.Now()
		}
		s.room.AddAction(time.Now().Unix(), "Bob", clientKeyMap[socketRequest.Payload.Hit])
	case "connect":
		s.Name = socketRequest.Payload.Name
		log.Println("Session name set to: ", s.Name)
		if s.room == nil {
			newGame, err := CreateGameV2(s.Name, 2, 2, []string{s.Id}, time.NewTicker(time.Second), []*Session{s})
			if err != nil {
				s.out <- []byte("Failed to create a new game room")
			}
			log.Printf("New game room created: %s", s.Name)
			(*rooms)[s.Name] = newGame
			s.room = newGame
		}
		go func() {
			players := []string{"Joe", "Nick", "Nikki", "Brand"}
			for i := 0; i < 4; i++ {
				time.Sleep(5 * time.Second)
				readyPlayers := players[:i+1]
				waitingPlayers := players[i+1:]
				var gameState = "WAITING"
				if len(readyPlayers) == len(players) {
					gameState = "READY"
				}
				for _, s := range s.room.sessions {
					payload, _ := json.Marshal(GameRoomStream{
						Name:           fmt.Sprintf("%s Room", s.room.Id),
						IsPrivate:      false,
						State:          gameState,
						ReadyPlayers:   readyPlayers,
						WaitingPlayers: waitingPlayers,
					})
					s.out <- []byte(payload)
				}

			}
		}()
		return
	case "join":
		roomName := socketRequest.Payload.RoomName
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
	}

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
			log.Printf("Recv %s, %s", s.Id, msg)
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
			log.Println("Message from someone", string(message))
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
