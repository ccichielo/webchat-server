package main

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type Client struct {
	conn     *websocket.Conn
	username string
}

type Message struct {
	User string `json:"user"`
	Text string `json:"text"`
}

var clients = make(map[*Client]bool)
var broadcast = make(chan Message)
var mu sync.Mutex

func handleConnection(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println("Error while connecting:", err)
		return
	}

	username := r.URL.Query().Get("username")
	client := &Client{conn: conn, username: username}
	clients[client] = true

	go handleMessages(client)
}

func handleMessages(client *Client) {
	defer func() {
		client.conn.Close()
		mu.Lock()
		delete(clients, client)
		mu.Unlock()
	}()

	for {
		var msg Message
		err := client.conn.ReadJSON(&msg)
		if err != nil {
			fmt.Println("Error while reading message:", err)
			break
		}

		msg.User = client.username
		broadcast <- msg
	}

}

func handleBroadcast() {
	for msg := range broadcast {
		mu.Lock()
		for client := range clients {
			err := client.conn.WriteJSON(msg)
			if err != nil {
				client.conn.Close()
				delete(clients, client)
			}
		}
		mu.Unlock()
	}
}

func main() {
	http.HandleFunc("/ws", handleConnection)
	go handleBroadcast()

	fmt.Println("Server started on :8080")
	http.ListenAndServe(":8080", nil)
}
