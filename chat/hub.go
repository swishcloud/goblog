package chat

import (
	"strconv"
	"time"

	"github.com/swishcloud/goblog/common"
	"github.com/swishcloud/goblog/storage"
)

type Hub struct {
	//Inbound message from the clients
	broadcast chan *[]byte
	//registered clients
	clients map[*Client]bool
	//register requests from the clients
	register chan *Client
	//unregister requests from the clients
	unregister   chan *Client
	FileLocation string
	DbConnInfo   string
}

func GetHub(conn_info string) *Hub {
	if hub == nil {
		hub = &Hub{
			broadcast:  make(chan *[]byte),
			register:   make(chan *Client),
			unregister: make(chan *Client),
			clients:    map[*Client]bool{},
			DbConnInfo: conn_info,
		}
	}
	return hub
}

var hub *Hub

func (h *Hub) Run() {
	for {
		select {
		case message := <-h.broadcast:
			h.Broadcast(message)
		case regClient := <-h.register:
			h.clients[regClient] = true
			lastMsgs, err := h.GetLastMessages()
			if err != nil {
				b := []byte(err.Error())
				regClient.send <- NewMessage(2, time.Now().Format(common.TimeLayout2), &b, "").getBytes()
			} else {
				for _, v := range lastMsgs {
					regClient.send <- v.getBytes()
				}
			}
			b := []byte(strconv.Itoa(len(h.clients)))
			h.Broadcast(NewMessage(3, time.Now().Format(common.TimeLayout2), &b, "").getBytes())
		case unRegClient := <-h.unregister:
			delete(h.clients, unRegClient)
			close(unRegClient.send)
			b := []byte(strconv.Itoa(len(h.clients)))
			h.Broadcast(NewMessage(3, time.Now().Format(common.TimeLayout2), &b, "").getBytes())
		}
	}
}
func (h *Hub) Broadcast(message *[]byte) {
	for client := range h.clients {
		client.send <- message
	}
}

func (h *Hub) GetLastMessages() ([]Message, error) {
	s := storage.NewSQLManager(h.DbConnInfo)
	msgDtos, err := s.WsmessageTop()
	if err != nil {
		return nil, err
	}
	r := []Message{}
	for i := len(msgDtos) - 1; i >= 0; i-- {
		b := []byte(msgDtos[i].Msg)
		r = append(r, NewMessage(2, msgDtos[i].InsertTime, &b, ""))
	}
	return r, nil
}
