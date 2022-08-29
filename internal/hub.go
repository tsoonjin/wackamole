package internal

import (
	"fmt"
	"strings"
)

// Hub maintains the set of active clients and broadcasts messages to the
// clients.
type Hub struct {
	// Registered clients.
	clients map[*Client]bool

	// Inbound messages from the clients.
	broadcast chan []byte

	// Register requests from the clients.
	register chan *Client

	// Unregister requests from clients.
	unregister chan *Client
	rooms      []Room
}

type Room struct {
	size    int
	Name    string
	clients []int
}

func newHub() *Hub {
	return &Hub{
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
		rooms:      []Room{},
	}
}

func (h *Hub) run() {
	for {
		select {
		case client := <-h.register:
			h.clients[client] = true
		case client := <-h.unregister:
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.send)
			}
		case message := <-h.broadcast:
			msg := string(message)
			fmt.Println(msg)
			command := strings.Split(msg, " ")[0]
			if strings.HasPrefix(command, "/") {
				fmt.Println("Command detected", command)
				switch command {
				case "/new":
					fmt.Println("Create new room")
					h.rooms = append(h.rooms, Room{Name: "New Room"})
					fmt.Println("No of rooms now", len(h.rooms))
				default:
					fmt.Println("No matching command")
				}
			}
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					close(client.send)
					delete(h.clients, client)
				}
			}
		}
	}
}
