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
	"os"
	"os/signal"
)

var rooms = make(map[string]*internal.Game)
var sessions = make(map[string]internal.Session)

var addr = flag.String("addr", "localhost:8080", "http service address")

var upgrader = websocket.Upgrader{} // use default options

func handleConnect(interupt chan os.Signal) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		c, err := upgrader.Upgrade(w, r, nil)
		session := internal.InitSession(c)
		if err != nil {
			log.Print("upgrade:", err)
			return
		}
		go session.Run(interupt, &rooms)
	}
}

func main() {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	flag.Parse()
	log.SetFlags(0)
	http.HandleFunc("/connect", handleConnect(interrupt))
	log.Fatal(http.ListenAndServe(*addr, nil))
}
