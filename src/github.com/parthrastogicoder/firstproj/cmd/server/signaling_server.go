package main

import (
	"log"
	"net/http"
	"sync"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	// Allow any origin (for testing). In production, you should verify the origin.
	CheckOrigin: func(r *http.Request) bool { return true },
}

var (
	clients = make(map[*websocket.Conn]bool)
	mu      sync.Mutex
)

func handleConnections(w http.ResponseWriter, r *http.Request) {
	// Upgrade the HTTP connection to a WebSocket connection.
	ws, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("Upgrade error: %v", err)
		return
	}
	defer ws.Close()

	// Add the new connection to our list of clients.
	mu.Lock()
	clients[ws] = true
	mu.Unlock()

	// Listen for incoming messages.
	for {
		var msg map[string]interface{}
		err := ws.ReadJSON(&msg)
		if err != nil {
			log.Printf("Read error: %v", err)
			mu.Lock()
			delete(clients, ws)
			mu.Unlock()
			break
		}
		// Broadcast the message to all other clients.
		mu.Lock()
		for client := range clients {
			if client != ws {
				if err := client.WriteJSON(msg); err != nil {
					log.Printf("Write error: %v", err)
					client.Close()
					delete(clients, client)
				}
			}
		}
		mu.Unlock()
	}
}

func main() {
	http.HandleFunc("/ws", handleConnections)
	log.Println("Signaling server started on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
