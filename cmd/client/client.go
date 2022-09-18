// Copyright 2015 The Gorilla WebSocket Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build ignore
// +build ignore

package main

import (
	"bufio"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/tsoonjin/wackamole/internal"
	"golang.org/x/term"
	"log"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"time"
)

type State struct {
	Game internal.GameState
}

var state = &State{Game: internal.WaitEnoughPlayers}
var addr = flag.String("addr", "localhost:8080", "http service address")
var endpoint = flag.String("endpoint", "/register", "endpoint of server")

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Maximum message size allowed from peer.
	maxMessageSize = 8192

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Time to wait before force close on connection.
	closeGracePeriod = 10 * time.Second
)

func readFromStdin(ws *websocket.Conn, in chan string, done chan struct{}, state *State) {
	// create new reader from stdin
	reader := bufio.NewReader(os.Stdin)
	for {
		// read by one line (enter pressed)
		s, err := reader.ReadString('\n')
		// check for errors
		if err != nil {
			fmt.Println("Error in read string", err)
			// close channel just to inform others
			close(in)
			close(done)
		}
		in <- s
		if err := ws.WriteMessage(websocket.TextMessage, []byte(s)); err != nil {
			fmt.Println("Error writing to server")
			ws.Close()
			break
		}
		log.Print("[me]: ", s)
	}
	ws.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	time.Sleep(closeGracePeriod)
	ws.Close()
}

func drawGameBoard(board [3][3]int) string {
	gameStr := []string{}
	for _, row := range board {
		drawRow := []string{}
		for _, cell := range row {
			if cell == 1 {
				drawRow = append(drawRow, " M ")
			} else {
				drawRow = append(drawRow, "   ")
			}

		}
		gameStr = append(gameStr, "|"+strings.Join(drawRow, "|")+"|")
	}
	return "-------------\r\n" + strings.Join(gameStr, "\r\n-------------\r\n") + "\r\n-------------\r\n"
}

func main() {
	flag.Parse()
	log.SetFlags(0)

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)

	u := url.URL{Scheme: "ws", Host: *addr, Path: *endpoint}

	c, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		log.Fatal("dial:", err)
	}
	defer c.Close()

	done := make(chan struct{})
	input := make(chan string)

	go readFromStdin(c, input, done, state)
	go func() {
		defer close(done)
		for {
			_, message, err := c.ReadMessage()
			var dat internal.GameBoard
			jsonErr := json.Unmarshal(message, &dat)
			if jsonErr == nil {
				fmt.Print("\033[2J")
				fmt.Print("\033[H")
				fmt.Print(drawGameBoard(dat.Board))
			}
			if string(message) == "Game started" {
				state.Game = internal.Running
				go func(ws *websocket.Conn) {
					log.Println("Do something")
					// fd 0 is stdin
					state, err := term.MakeRaw(0)
					if err != nil {
						log.Fatalln("setting stdin to raw:", err)
					}
					defer func() {
						if err := term.Restore(0, state); err != nil {
							log.Println("warning, failed to restore terminal:", err)
						}
					}()

					in := bufio.NewReader(os.Stdin)
					for {
						r, _, err := in.ReadRune()
						if err != nil {
							log.Println("stdin:", err)
							break
						}
						if err := ws.WriteMessage(websocket.TextMessage, []byte(string(r))); err != nil {
							fmt.Println("Error writing to server")
							ws.Close()
							break
						}
						fmt.Printf("read rune %q\r\n", r)
						if r == 'q' {
							break
						}
					}
				}(c)
			}
			if err != nil {
				log.Println("[error]:", err)
				return
			}
			if !strings.HasPrefix(string(message), "Game Update") && jsonErr != nil {
				log.Printf("[server]: %s", message)
			}
		}
	}()
exit:
	for {
		select {
		case in := <-input:
			// remove all leading and trailing white space
			in = strings.TrimSpace(in)
			if in == "exit" {
				// if exit command received
				// break from infinite loop to label and go next
				// line after for loop
				break exit
			}
		case <-done:
			return
		case <-interrupt:
			log.Println("interrupt")

			// Cleanly close the connection by sending a close message and then
			// waiting (with timeout) for the server to close the connection.
			err := c.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
			if err != nil {
				log.Println("write close:", err)
				return
			}
			return
		}
	}
}
