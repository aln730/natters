package main

import (
	"fmt"

	"github.com/aln730/natters/internal/server"
)

func main() {
	fmt.Println("Starting natters on :4222")
	server.Start(":4222")
}
