package chat

import (
	"github.com/github-123456/goblog/common"
	"github.com/github-123456/goblog/dbservice"
	"time"
)

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
			lastMsgs, err := GetLastMessages()
			if err != nil {
				regClient.send <- TextMessage{Time: time.Now().Format(common.TimeLayout2), Text: err.Error()}.getBytes()
			} else {
				for _, v := range lastMsgs {
					regClient.send <- v.getBytes()
				}
			}
		case unRegClient := <-h.unregister:
			delete(h.clients, unRegClient)
			close(unRegClient.send)
		}
	}
}

func GetLastMessages() ([]TextMessage, error) {
	msgDtos, err := dbservice.WsmessageTop()
	if err != nil {
		return nil, err
	}
	r := []TextMessage{}
	for _, v := range msgDtos {
		r = append(r, TextMessage{Text: v.Msg, Time: v.InsertTime})
	}
	return r, nil
}
