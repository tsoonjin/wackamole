// Copyright 2015 The Gorilla WebSocket Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

//go:build ignore
// +build ignore

package main

import (
	"encoding/json"
	"flag"
	"github.com/gorilla/websocket"
	"github.com/tsoonjin/wackamole/internal"
	"golang.org/x/exp/maps"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
)

var rooms = make(map[string]*internal.Game)
var sessions = make(map[string]internal.Session)

var addr = flag.String("addr", "localhost:8080", "http service address")

var upgrader = websocket.Upgrader{} // use default options

// HTTP Handlers

func handleConnect(interupt chan os.Signal) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		upgrader.CheckOrigin = func(r *http.Request) bool { return true }
		c, err := upgrader.Upgrade(w, r, nil)
		session := internal.InitSession(c)
		if err != nil {
			log.Print("upgrade:", err)
			return
		}
		go session.Run(interupt, &rooms)
	}
}

func handleListRoom(w http.ResponseWriter, r *http.Request) {
	var (
		res      []*internal.Game
		startIdx = 0
		endIdx   = len(rooms)
	)
	page, _ := strconv.Atoi(r.FormValue("page"))
	perPage, _ := strconv.Atoi(r.FormValue("limit"))

	if idx := (page - 1) * perPage; idx < len(rooms) {
		startIdx = idx
	}
	if idx := page * perPage; idx < len(rooms) {
		endIdx = idx
	}
	var roomList = maps.Values(rooms)
	res = roomList[startIdx:endIdx]
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(res)
}

func main() {
	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt)
	flag.Parse()
	log.SetFlags(0)
	http.HandleFunc("/connect", handleConnect(interrupt))
	http.HandleFunc("/rooms", handleListRoom)
	log.Fatal(http.ListenAndServe(*addr, nil))
}
