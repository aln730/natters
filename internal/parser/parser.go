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
	UNSUB
	PUB
)

type Command struct {
	Type    CommandType
	Topic   string
	SID     int
	Max     int
	Payload []byte
}

func ParseCommand(line []byte, payloadReader func(int) ([]byte, error)) (*Command, error) {
	line = bytes.TrimSpace(line)
	if len(line) == 0 {
		return nil, errors.New("empty line")
	}

	parts := bytes.Fields(line)
	if len(parts) == 0 {
		return nil, errors.New("empty command")
	}

	switch string(parts[0]) {
	case "PING":
		return &Command{Type: PING}, nil
	case "PONG":
		return &Command{Type: PONG}, nil
	case "CONNECT":
		return &Command{Type: CONNECT}, nil
	case "SUB":
		if len(parts) < 3 {
			return nil, errors.New("invalid SUB command")
		}
		sid, err := strconv.Atoi(string(parts[2]))
		if err != nil {
			return nil, errors.New("invalid SUB SID")
		}
		return &Command{
			Type:  SUB,
			Topic: string(parts[1]),
			SID:   sid,
		}, nil

	case "UNSUB":
		if len(parts) < 3 {
			return nil, errors.New("invalid UNSUB")
		}

		topic := string(parts[1])

		sid, err := strconv.Atoi(string(parts[2]))
		if err != nil {
			return nil, errors.New("invalid SID")
		}

		max := 0
		if len(parts) >= 4 {
			max, err = strconv.Atoi(string(parts[3]))
			if err != nil {
				return nil, errors.New("invalid max value")
			}
		}

		return &Command{
			Type:  UNSUB,
			Topic: topic,
			SID:   sid,
			Max:   max,
		}, nil
	case "PUB":
		if len(parts) < 3 {
			return nil, errors.New("invalid PUB command")
		}
		topic := string(parts[1])
		size, err := strconv.Atoi(string(parts[2]))
		if err != nil {
			return nil, errors.New("invalid PUB payload size")
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
	default:
		return nil, errors.New("unknown command")
	}
}
