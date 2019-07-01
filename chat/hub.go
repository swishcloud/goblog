package chat

type Hub struct {
	//Inbound message from the clients
	broadcast chan []byte
	//registered clients
	clients map[*Client]bool
	//register requests from the clients
	register chan *Client
	//unregister requests from the clients
	unregister chan *Client
}

func GetHub() *Hub {
	if hub == nil {
		hub = &Hub{
			broadcast:  make(chan []byte),
			register:   make(chan *Client),
			unregister: make(chan *Client),
			clients:    map[*Client]bool{},
		}
	}
	return hub
}

var hub *Hub

func (h *Hub) Run() {
	for {
		select {
		case message := <-h.broadcast:
			for client := range h.clients {
				client.send <- message
			}
		case regClient := <-h.register:
			h.clients[regClient] = true
		case unRegClient := <-h.unregister:
			delete(h.clients, unRegClient)
			close(unRegClient.send)
		}
	}
}
