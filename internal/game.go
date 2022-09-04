package internal

import (
	"errors"
	"fmt"
	"github.com/gorilla/websocket"
	"log"
	"time"
)

type GameState struct {
	slug string
}

var ErrorMaxPlayersReached = errors.New("Max players reached")

func (g *Game) transitionGameState() {
	timeElapsed := time.Since(g.startTime).Milliseconds()
	timeLeft := g.gameDurationMs - timeElapsed
	if g.state == Running && timeLeft <= 0 {
		g.state = Over
		for playerId, conn := range g.conn {
			w, err := conn.NextWriter(websocket.TextMessage)
			if err != nil {
				log.Println("Error writing message")
			}
			msgToClient := fmt.Sprintf("[%s]: Game is over\n", playerId)
			w.Write([]byte(msgToClient))
			log.Printf("Time left: %d\n", timeLeft)
		}
		log.Println("Game is over")
	}
	if g.state == Running {
		for playerId, conn := range g.conn {
			w, err := conn.NextWriter(websocket.TextMessage)
			if err != nil {
				log.Println("Error writing message")
			}
			msgToClient := fmt.Sprintf("[%s]: Time left: %d\n", playerId, timeLeft)
			w.Write([]byte(msgToClient))
			log.Printf("Time left: %d\n", timeLeft)
		}
	}
	if len(g.Players) == g.minPlayers && g.state == WaitEnoughPlayers {
		g.state = WaitPlayersReady
		log.Printf("Waiting for players to get ready: %d/%d\n", len(g.playerReady), g.minPlayers)
	}
	if len(g.playerReady) == g.minPlayers && g.state == WaitPlayersReady {
		g.state = Running
		log.Println("Game is starting")
		g.startTime = time.Now()
	}
}

func (g GameState) String() string {
	return g.slug
}

var (
	Running           = GameState{"running"}
	WaitEnoughPlayers = GameState{"waitEnoughPlayers"}
	WaitPlayersReady  = GameState{"waitPlayersReady"}
	Over              = GameState{"over"}
)

type Game struct {
	// Id must be unique. Akin to room name
	Id             string
	startTime      time.Time
	gameDurationMs int64
	maxPlayers     int
	minPlayers     int
	Players        []string
	state          GameState
	playerReady    []string
	conn           map[string]*websocket.Conn
}

func CreateGame(name string, minPlayers int, maxPlayers int, players []string, ticker *time.Ticker, conns map[string]*websocket.Conn) (*Game, error) {
	if minPlayers == 0 {
		minPlayers = 2
	}
	if maxPlayers == 0 {
		maxPlayers = 2
	}
	if len(players) > maxPlayers {
		return nil, ErrorMaxPlayersReached
	}
	newGame := &Game{gameDurationMs: 60000, Id: name, maxPlayers: maxPlayers, minPlayers: minPlayers, Players: players, state: WaitEnoughPlayers, playerReady: []string{}, conn: conns}
	go func() {
		for {
			select {
			case <-ticker.C:
				newGame.transitionGameState()
			}
			if newGame.state == Over {
				ticker.Stop()
				log.Println("Ticker is stopped. Game over")
				return
			}
		}
	}()
	return newGame, nil
}

func (g *Game) AddPlayer(playerId string, conn *websocket.Conn) error {
	if len(g.Players) == g.maxPlayers {
		return ErrorMaxPlayersReached
	}
	g.Players = append(g.Players, playerId)
	g.conn[playerId] = conn
	log.Printf("%d no of connections registered", len(g.conn))
	return nil
}

func (g *Game) AddPlayerReady(playerId string) {
	if !contains(g.playerReady, playerId) && contains(g.Players, playerId) {
		g.playerReady = append(g.playerReady, playerId)
	}
}