package chat

import (
	"encoding/json"
	"github.com/github-123456/goblog/common"
	"github.com/github-123456/goblog/dbservice"
	"github.com/github-123456/goweb"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"time"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

//client is a middleware between the websocket connection and the hub
type Client struct {
	hub  *Hub
	conn *websocket.Conn
	//buffered channel of outbound messages
	send chan []byte
}

func (c *Client) readPump() {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()
	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error { c.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			println(err)
			return
		}
		dbservice.WsmessageInsert(string(message))
		c.hub.broadcast <- TextMessage{Time: time.Now().Format(common.TimeLayout2), Text: string(message)}.getBytes()
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case message := <-c.send:
			err := c.sendMessage(websocket.TextMessage, message)
			if err != nil {
				println(err)
				return
			}
		case <-ticker.C:
			err := c.sendMessage(websocket.PingMessage, nil)
			if err != nil {
				println(err)
				return
			}
		}
	}
}

func (c *Client) sendMessage(messageType int, msg []byte) error {
	c.conn.SetWriteDeadline(time.Now().Add(writeWait))
	if err := c.conn.WriteMessage(messageType, msg); err != nil {
		log.Println(err)
		return err
	}
	return nil
}

func WebSocket(ctx *goweb.Context) {
	conn, err := upgrader.Upgrade(ctx.Writer, ctx.Request, nil)
	if err != nil {
		panic(err)
	}
	client := &Client{GetHub(), conn, make(chan []byte, 256),}
	client.hub.register <- client
	go client.writePump()
	go client.readPump()
}

type TextMessage struct {
	Time string `json:"time"`
	Text string `json:"text"`
}

func (msg TextMessage) getBytes() []byte {
	b, err := json.Marshal(msg)
	if err != nil {
		panic(err)
	}
	return b
}
