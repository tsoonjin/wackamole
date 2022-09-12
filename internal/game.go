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
		for _, s := range g.sessions {
			msgToClient := fmt.Sprintf("[%s]: Game is over\n", s.Id)
			s.out <- []byte(msgToClient)
		}
		log.Println("Game is over")
	}
	if g.state == Running {
		for _, item := range g.actions {
			if time.Now().Unix()-item.timestamp <= 3 {
				log.Printf("%s send %s", item.id, item.msg)
			}
		}
		for _, s := range g.sessions {
			msgToClient := fmt.Sprintf("Game Update [%s]: Time left: %d\n", s.Name, timeLeft)
			s.out <- []byte(msgToClient)
		}
		log.Printf("Game will be over in : %d seconds", timeLeft)
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

type Action struct {
	timestamp int64
	id        string
	msg       string
}

type Game struct {
	// Id must be unique. Akin to room name
	actions        []Action
	Id             string
	startTime      time.Time
	gameDurationMs int64
	maxPlayers     int
	minPlayers     int
	Players        []string
	state          GameState
	playerReady    []string
	conn           map[string]*websocket.Conn
	sessions       []*Session
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
	newGame := &Game{gameDurationMs: 60000, Id: name, maxPlayers: maxPlayers, minPlayers: minPlayers, Players: players, state: WaitEnoughPlayers, playerReady: []string{}, conn: conns, actions: []Action{}}
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

func CreateGameV2(name string, minPlayers int, maxPlayers int, players []string, ticker *time.Ticker, sessions []*Session) (*Game, error) {
	if minPlayers == 0 {
		minPlayers = 2
	}
	if maxPlayers == 0 {
		maxPlayers = 2
	}
	if len(players) > maxPlayers {
		return nil, ErrorMaxPlayersReached
	}
	newGame := &Game{gameDurationMs: 60000, Id: name, maxPlayers: maxPlayers, minPlayers: minPlayers, Players: players, state: WaitEnoughPlayers, playerReady: []string{}, actions: []Action{}, sessions: sessions}
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

func (g *Game) AddPlayer(playerId string, session *Session) error {
	if len(g.Players) == g.maxPlayers {
		return ErrorMaxPlayersReached
	}
	g.Players = append(g.Players, playerId)
	g.sessions = append(g.sessions, session)
	log.Printf("%d no of connections registered", len(g.sessions))
	return nil
}

func (g *Game) AddPlayerReady(playerId string) {
	if !contains(g.playerReady, playerId) && contains(g.Players, playerId) && g.state == WaitPlayersReady {
		g.playerReady = append(g.playerReady, playerId)
	}
}

func (g *Game) AddAction(ts int64, playerId string, msg string) {
	g.actions = append(g.actions, Action{timestamp: ts, id: playerId, msg: msg})
}
