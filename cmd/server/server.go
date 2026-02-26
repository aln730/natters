package server

import (
	"bufio"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/aln730/natters/internal/parser"
)

type Client struct {
	conn net.Conn
	mu   sync.Mutex
}

type Subscription struct {
	sid    int
	client *Client
}

const MaxPayload = 1024 * 1024 // 1 MB

var (
	topics      = make(map[string][]*Subscription)
	topicsMutex sync.RWMutex
	sidCounter  = 1
	seqCounter  = make(map[string]*uint64)
	seqMutex    sync.Mutex
)

func Start(address string) {
	ln, err := net.Listen("tcp", address)
	if err != nil {
		panic(err)
	}
	fmt.Println(`
 _        _______ __________________ _______  _______  _______ 
( (    /|(  ___  )\__   __/\__   __/(  ____ \(  ____ )(  ____ \
|  \  ( || (   ) |   ) (      ) (   | (    \/| (    )|| (    \/
|   \ | || (___) |   | |      | |   | (__    | (____)|| (_____ 
| (\ \) ||  ___  |   | |      | |   |  __)   |     __)(_____  )
| | \   || (   ) |   | |      | |   | (      | (\ (         ) |
| )  \  || )   ( |   | |      | |   | (____/\| ) \ \__/\____) |
|/    )_)|/     \|   )_(      )_(   (_______/|/   \__/\_______)
                                                               

NATTERS SERVER v1.0.24
Listening on ` + address + `
`)
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		count := 1
		for range ticker.C {
			msg := fmt.Sprintf("Auto message #%d", count)
			publish("AUTO", []byte(msg))
			fmt.Println("Published:", msg)
			count++
		}
	}()

	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Println("Error accepting connection:", err)
			continue
		}

		client := &Client{conn: conn}
		go handleClient(client)
	}
}

func handleClient(client *Client) {
	defer client.conn.Close()
	clientAddr := client.conn.RemoteAddr().String()

	info := map[string]interface{}{
		"server_id":   "natters",
		"version":     "1.02.4",
		"go":          "go1.24.11",
		"host":        "nixon.csh.rit.edu",
		"port":        4222,
		"max_payload": MaxPayload,
	}

	infoBytes, _ := json.Marshal(info)

	// Send ASCII banner + info
	banner := fmt.Sprintf(`
	_        _______ __________________ _______  _______  _______ 
	( (    /|(  ___  )\__   __/\__   __/(  ____ \(  ____ )(  ____ \
	|  \  ( || (   ) |   ) (      ) (   | (    \/| (    )|| (    \/
	|   \ | || (___) |   | |      | |   | (__    | (____)|| (_____ 
	| (\ \) ||  ___  |   | |      | |   |  __)   |     __)(_____  )
	| | \   || (   ) |   | |      | |   | (      | (\ (         ) |
	| )  \  || )   ( |   | |      | |   | (____/\| ) \ \__/\____) |
	|/    )_)|/     \|   )_(      )_(   (_______/|/   \__/\_______/

	NATTERS SERVER v1.0.24
	Listening on nixon.csh.rit.edu
	INFO %s`, infoBytes)

	client.write([]byte(banner + "\r\n"))

	reader := bufio.NewReader(client.conn)

	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			fmt.Println("Connection closed:", clientAddr)
			removeClientFromAllTopics(client)
			return
		}

		lineStr := strings.TrimSpace(string(line))
		if lineStr == "" {
			continue
		}

		cmd, err := parser.ParseCommand(line, func(size int) ([]byte, error) {
			if size > MaxPayload {
				return nil, fmt.Errorf("payload too large")
			}
			buf := make([]byte, size)
			_, err := reader.Read(buf)
			if err != nil {
				return nil, err
			}
			reader.ReadBytes('\n')
			return buf, nil
		})
		if err != nil {
			fmt.Println("Parse error:", err)
			continue
		}

		switch cmd.Type {
		case parser.CONNECT:

		case parser.PING:
			client.write([]byte("PONG\r\n"))

		case parser.SUB:
			addSubscriber(cmd.Topic, cmd.SID, client)
			client.write([]byte("+OK\r\n"))

		case parser.PUB:
			if len(cmd.Payload) > MaxPayload {
				fmt.Println("Payload too large, ignoring")
				continue
			}
			publish(cmd.Topic, cmd.Payload)
			client.write([]byte("+OK\r\n"))

		case parser.UNSUB:
			removeSubscriber(cmd.Topic, cmd.SID, cmd.Max, client)
			client.write([]byte("+OK\r\n"))

		case parser.PONG:
			// ignore
		}
	}
}

func (c *Client) write(data []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.conn.Write(data)
}

func addSubscriber(topic string, sid int, client *Client) {
	topicsMutex.Lock()
	defer topicsMutex.Unlock()

	for _, s := range topics[topic] {
		if s.sid == sid && s.client == client {
			return
		}
	}

	topics[topic] = append(topics[topic], &Subscription{sid: sid, client: client})

	seqMutex.Lock()
	defer seqMutex.Unlock()
	if _, ok := seqCounter[topic]; !ok {
		var z uint64
		seqCounter[topic] = &z
	}
}

func removeSubscriber(topic string, sid int, max int, client *Client) {
	topicsMutex.Lock()
	defer topicsMutex.Unlock()

	subs := topics[topic]
	filtered := subs[:0]
	removed := 0

	for _, s := range subs {
		if s.sid == sid && s.client == client && (max == 0 || removed < max) {
			removed++
			continue
		}
		filtered = append(filtered, s)
	}

	topics[topic] = filtered
}

func removeClientFromAllTopics(client *Client) {
	topicsMutex.Lock()
	defer topicsMutex.Unlock()
	for topic, subs := range topics {
		filtered := subs[:0]
		for _, s := range subs {
			if s.client != client {
				filtered = append(filtered, s)
			}
		}
		topics[topic] = filtered
	}
}

func publish(topic string, payload []byte) {
	topicsMutex.RLock()
	subs := append([]*Subscription(nil), topics[topic]...)
	topicsMutex.RUnlock()

	if len(subs) == 0 {
		return
	}

	seqMutex.Lock()
	_ = atomic.AddUint64(seqCounter[topic], 1)
	seqMutex.Unlock()

	for _, sub := range subs {
		msg := fmt.Sprintf("MSG %s %d %d\r\n%s\r\n", topic, sub.sid, len(payload), payload)
		sub.client.write([]byte(msg))
	}
}
