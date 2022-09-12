// Copyright 2015 The Gorilla WebSocket Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build ignore
// +build ignore

package main

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/gorilla/websocket"
	"log"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"time"
)

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

func readFromStdin(ws *websocket.Conn, in chan string, done chan struct{}) {
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

	go readFromStdin(c, input, done)
	go func() {
		defer close(done)
		for {
			_, message, err := c.ReadMessage()
			if err != nil {
				log.Println("[error]:", err)
				return
			}
			if !strings.HasPrefix(string(message), "Game Update") {
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
