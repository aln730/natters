package parser

import (
	"bytes"
	"errors"
	"strconv"
)

type CommandType int

const (
	CONNECT CommandType = iota
	PING
	PONG
	SUB
	PUB
)

type Command struct {
	Type    CommandType
	Topic   string
	SID     string
	Payload []byte
}

func ParseCommand(line []byte, payloadReader func(int) ([]byte, error)) (*Command, error) {
	line = bytes.TrimSpace(line)

	switch {
	case bytes.HasPrefix(line, []byte("PING")):
		return &Command{Type: PING}, nil

	case bytes.HasPrefix(line, []byte("PONG")):
		return &Command{Type: PONG}, nil

	case bytes.HasPrefix(line, []byte("CONNECT")):
		return &Command{Type: CONNECT}, nil

	case bytes.HasPrefix(line, []byte("SUB")):
		parts := bytes.Fields(line)
		if len(parts) < 3 {
			return nil, errors.New("invalid SUB command")
		}
		return &Command{
			Type:  SUB,
			Topic: string(parts[1]),
			SID:   string(parts[2]),
		}, nil

	case bytes.HasPrefix(line, []byte("PUB")):
		parts := bytes.Fields(line)
		if len(parts) < 3 {
			return nil, errors.New("invalid PUB command")
		}

		topic := string(parts[1])
		size, err := strconv.Atoi(string(parts[2]))
		if err != nil {
			return nil, errors.New("invalid payload size")
		}

		payload, err := payloadReader(size)
		if err != nil {
			return nil, err
		}

		return &Command{
			Type:    PUB,
			Topic:   topic,
			Payload: payload,
		}, nil
	}

	return nil, errors.New("unknown command")
}
