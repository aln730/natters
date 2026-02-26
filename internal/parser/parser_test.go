package parser_test

import (
	"bytes"
	"testing"

	"github.com/aln730/natters/internal/parser"
)

func TestParseCommand(t *testing.T) {
	tests := []struct {
		name        string
		line        []byte
		payload     []byte
		wantType    parser.CommandType
		wantTopic   string
		wantSID     string
		wantPayload []byte
	}{
		{
			name:     "PING",
			line:     []byte("PING\r\n"),
			wantType: parser.PING,
		},
		{
			name:     "PONG",
			line:     []byte("PONG\r\n"),
			wantType: parser.PONG,
		},
		{
			name:     "CONNECT",
			line:     []byte("CONNECT {}\r\n"),
			wantType: parser.CONNECT,
		},
		{
			name:      "SUB",
			line:      []byte("SUB FOO 1\r\n"),
			wantType:  parser.SUB,
			wantTopic: "FOO",
			wantSID:   "1",
		},
		{
			name:        "PUB",
			line:        []byte("PUB CodingChallenge 11\r\n"),
			payload:     []byte("Hello John!"),
			wantType:    parser.PUB,
			wantTopic:   "CodingChallenge",
			wantPayload: []byte("Hello John!"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd, err := parser.ParseCommand(tt.line, func(size int) ([]byte, error) {
				if tt.payload != nil {
					if size != len(tt.payload) {
						t.Fatalf("expected payload size %d, got %d", len(tt.payload), size)
					}
					return tt.payload, nil
				}
				return nil, nil
			})

			if err != nil {
				t.Fatalf("ParseCommand() error = %v", err)
			}
			if cmd.Type != tt.wantType {
				t.Errorf("got Type %v, want %v", cmd.Type, tt.wantType)
			}
			if cmd.Topic != tt.wantTopic {
				t.Errorf("got Topic %q, want %q", cmd.Topic, tt.wantTopic)
			}
			if cmd.SID != tt.wantSID {
				t.Errorf("got SID %q, want %q", cmd.SID, tt.wantSID)
			}
			if !bytes.Equal(cmd.Payload, tt.wantPayload) {
				t.Errorf("got Payload %q, want %q", cmd.Payload, tt.wantPayload)
			}
		})
	}
}
