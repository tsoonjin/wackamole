// Copyright 2015 The Gorilla WebSocket Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build ignore
// +build ignore

package main

import (
	"flag"
	"github.com/gorilla/websocket"
	"github.com/tsoonjin/wackamole/internal"
	"log"
	"net/http"
	"strings"
	"time"
)

var rooms = make(map[string]*internal.Game)
var clientConnections = []websocket.Conn{}

var addr = flag.String("addr", "localhost:8080", "http service address")

var upgrader = websocket.Upgrader{} // use default options

func handleConnection(w http.ResponseWriter, r *http.Request) {
	c, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Print("upgrade:", err)
		return
	}
	defer c.Close()
	for {
		_, message, err := c.ReadMessage()
		if err != nil {
			log.Println("read:", err)
			break
		}
		splittedMsg := strings.Split(string(message), " ")
		command := splittedMsg[0]
		args := splittedMsg[1:]
		if command == "/join" {
			roomName := internal.ClearString(args[0])
			playerId := internal.ClearString(args[1])
			if game, ok := rooms[roomName]; ok {
				log.Printf("Room: %s exists with %d players\n", roomName, len(game.Players))
				log.Printf("%s, welcome to room %s", playerId, roomName)
				err := game.AddPlayer(playerId, c)
				if err != nil {
					log.Println(err)
				}
			} else {
				log.Printf("New Game: %s", roomName)
				conns := make(map[string]*websocket.Conn)
				conns[playerId] = c
				newGame, err := internal.CreateGame(roomName, 2, 2, []string{playerId}, time.NewTicker(time.Second), conns)
				if err != nil {
					log.Println(err)
				}
				rooms[roomName] = newGame
			}
		}
		if command == "/ready" {
			roomName := internal.ClearString(args[0])
			playerId := internal.ClearString(args[1])
			if game, ok := rooms[roomName]; ok {
				log.Printf("%s, you are ready now to play in room %s", playerId, roomName)
				game.AddPlayerReady(playerId)
			} else {
				log.Println("Room does not exists")

			}
		}
		if err != nil {
			log.Println("write:", err)
			break
		}
	}
}

func main() {
	flag.Parse()
	log.SetFlags(0)
	http.HandleFunc("/register", handleConnection)
	log.Fatal(http.ListenAndServe(*addr, nil))
}
