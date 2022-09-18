package internal

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gorilla/websocket"
	"log"
	"math/rand"
	"sort"
	"time"
)

type GameBoard struct {
	Scores   map[string]int64 `json:"scores"`
	Healths  map[string]int64 `json:"healths"`
	GameTime int64            `json:"gameTime"`
	Board    [3][3]int        `json:"board"`
}

type GameState struct {
	slug string
}

var ErrorMaxPlayersReached = errors.New("max players reached")
var keyMap = map[string][]int{"w": []int{0, 0}, "e": []int{0, 1}, "r": []int{0, 2}, "s": []int{1, 0}, "d": []int{1, 1}, "f": []int{1, 2}, "x": []int{2, 0}, "c": []int{2, 1}, "v": []int{2, 2}}

func generateGameBoard(prevBoard [3][3]int) [3][3]int {
	molePosX := rand.Intn(3)
	molePosY := rand.Intn(3)
	rabbitPosX := rand.Intn(3)
	rabbitPosY := rand.Intn(3)
	for rabbitPosX == molePosX && rabbitPosY == molePosY {
		rabbitPosX = rand.Intn(3)
		rabbitPosY = rand.Intn(3)
	}
	newBoard := [3][3]int{}
	newBoard[molePosX][molePosY] = 1
	newBoard[rabbitPosX][rabbitPosY] = 2
	return newBoard
}

func (g *Game) initGameBoard() GameBoard {
	var scores = make(map[string]int64)
	var healths = make(map[string]int64)
	for _, s := range g.sessions {
		scores[s.Id] = 0
		healths[s.Id] = 3
	}
	newGameBoard := [3][3]int{}
	return GameBoard{Scores: scores, Healths: healths, GameTime: g.gameDurationMs, Board: newGameBoard}
}

func (g *Game) transitionGameState() {
	timeElapsed := time.Since(g.startTime).Milliseconds()
	timeLeft := g.gameDurationMs - timeElapsed
	if g.state == Running && timeLeft <= 0 {
		g.state = Over
		for _, s := range g.sessions {
			msgToClient := fmt.Sprintf("[%s]: Game is over\n", s.Id)
			s.out <- []byte(msgToClient)
		}
		keys := make([]string, 0, len(g.board.Scores))

		for key := range g.board.Scores {
			keys = append(keys, key)
		}

		sort.SliceStable(keys, func(i, j int) bool {
			return g.board.Scores[keys[i]] < g.board.Scores[keys[j]]
		})
		log.Println("Game is over")
	}
	if g.state == Running {
		encodedBoard, _ := json.Marshal(g.board)
		actions := g.actions
		sort.SliceStable(actions, func(i, j int) bool {
			return actions[i].timestamp < actions[j].timestamp
		})
		for _, item := range g.actions {
			key, exists := keyMap[item.msg]
			if exists {
				if g.board.Board[key[0]][key[1]] == 1 {
					g.board.Scores[item.id] += 1
				}
				if g.board.Board[key[0]][key[1]] == 2 {
					g.board.Healths[item.id] -= 1
				}
			}
		}
		for _, s := range g.sessions {
			s.out <- encodedBoard
		}
		newBoard := GameBoard{Scores: g.board.Scores, Healths: g.board.Healths, GameTime: timeLeft, Board: generateGameBoard(g.board.Board)}
		g.board = newBoard
		log.Printf("Game will be over in : %d seconds\nScores: %v", timeLeft, g.board.Scores)
	}
	if len(g.Players) == g.minPlayers && g.state == WaitEnoughPlayers {
		g.state = WaitPlayersReady
		log.Printf("Waiting for players to get ready: %d/%d\n", len(g.playerReady), g.minPlayers)
	}
	if len(g.playerReady) == g.minPlayers && g.state == WaitPlayersReady {
		for _, s := range g.sessions {
			s.out <- []byte("Game started")
		}
		newBoard := g.initGameBoard()
		g.state = Running
		log.Println("Game is starting")
		g.board = newBoard
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
	board          GameBoard
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
