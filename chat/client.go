package chat

import (
	"encoding/json"
	"fmt"
	"github.com/github-123456/goblog/common"
	"github.com/github-123456/goblog/dbservice"
	"github.com/github-123456/goweb"
	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"log"
	"net/http"
	"os"
	"strconv"
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
	maxMessageSize = 1024 * 1024
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
	send chan *[]byte
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
			println(err.Error())
			break
		}
		msg, err := parseMessage(message)
		if err != nil {
			println(err.Error())
			break
		}
		if msg.Type == 1 {
			savePath := c.hub.FileLocation + "/chat-images/" + uuid.New().String()+msg.R1
			file, err := os.Create(savePath)
			if err != nil {
				println(err.Error())
				break
			}
			_, err = file.Write(*msg.Content)
			file.Close()
			if err != nil {
				println(err.Error())
				break
			}

			c.hub.broadcast <- NewMessage(1, time.Now().Format(common.TimeLayout2), msg.Content, msg.Id).getBytes()
		} else {
			err = dbservice.WsmessageInsert(string(*msg.Content))
			if err != nil {
				println(err.Error())
				break
			}
			c.hub.broadcast <- NewMessage(2, time.Now().Format(common.TimeLayout2), msg.Content, msg.Id).getBytes()
		}
	}
}

type RequestHeader struct {
	Id   string `json:"id"`
	Type int    `json:"type"`
	R1   string    `json:"r1"`
	R2   string    `json:"r2"`
}
type RequestMessage struct {
	RequestHeader
	Content *[]byte
}

func parseMessage(message []byte) (*RequestMessage, error) {
	len := ""
	if message[3] == 0 {
		len = string((message[:3]))
	} else {
		len = string((message[:2]))
	}
	intLen, err := strconv.Atoi(len)
	if err != nil {
		panic(err)
	}
	jsonBytes := message[3 : 3+intLen]
	header := &RequestHeader{}
	err = json.Unmarshal(jsonBytes, header)
	if err != nil {
		return nil, err
	}
	content := message[3+intLen:]
	return &RequestMessage{*header, &content}, nil
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
			if message == nil {
				println("send channel closed")
				return
			}
			err := c.sendMessage(websocket.BinaryMessage, message)
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

func (c *Client) sendMessage(messageType int, msg *[]byte) error {
	var msgBuffer []byte
	if msg == nil {
		msgBuffer = nil
	} else {
		msgBuffer = *msg
	}
	c.conn.SetWriteDeadline(time.Now().Add(writeWait))
	if err := c.conn.WriteMessage(messageType, msgBuffer); err != nil {
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
	client := &Client{hub: GetHub(), conn: conn, send: make(chan *[]byte, 256)}
	client.hub.register <- client
	go client.writePump()
	go client.readPump()
}

type MessageHeader struct {
	Size int64  `json:"size"`
	Type int    `json:"type"`
	Time string `json:"time"`
	Id   string `json:"id"`
}

func (header MessageHeader) getBytes() []byte {
	b, err := json.Marshal(header)
	if err != nil {
		panic(err)
	}
	return b
}

type Message struct {
	MessageHeader
	Content *[]byte
}

func NewMessage(msgType int, time string, message *[]byte, id string) Message {
	header := MessageHeader{Size: int64(len(*message)), Time: time, Type: msgType, Id: id}
	return Message{header, message}
}
func (msg Message) getBytes() *[]byte {
	header := msg.MessageHeader.getBytes()
	bytes := make([]byte, msg.MessageHeader.Size+int64(len(header))+2)
	copy(bytes, []byte(fmt.Sprintf("%d", len(header))))
	copy(bytes[2:], header)
	copy(bytes[len(header)+2:], *msg.Content)
	return &bytes
}

func GetImageBytes(path string) *[]byte {
	file, err := os.Open(path)
	if err != nil {
		panic(err)
	}
	fileInfo, err := file.Stat()
	if err != nil {
		panic(err)
	}
	bytes := make([]byte, fileInfo.Size())
	_, err = file.Read(bytes)
	if err != nil {
		panic(err)
	}
	return &bytes;
}
