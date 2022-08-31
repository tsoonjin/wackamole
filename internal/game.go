package internal

type GameState struct {
	slug string
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
	Id         string
	maxPlayers int
	minPlayers int
	Players    []string
	state      GameState
}

func CreateGame(name string, minPlayers int, maxPlayers int) Game {
	if minPlayers == 0 {
		minPlayers = 2
	}
	if maxPlayers == 0 {
		maxPlayers = 2
	}
	return Game{Id: name, maxPlayers: maxPlayers, minPlayers: minPlayers, Players: []string{}, state: WaitEnoughPlayers}
}
