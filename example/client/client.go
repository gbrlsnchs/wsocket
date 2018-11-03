package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/gbrlsnchs/websocket"
)

func main() {
	address := flag.String("addr", "ws://echo.websocket.org", "address to connect to")
	flag.Parse()

	ws, err := websocket.Open(*address)
	if err != nil {
		log.Fatal(err)
	}

	w := ws.NewWriter()
	go func() {
		for ws.IsOpen() {
			b, _, err := ws.Accept()
			if err != nil {
				fmt.Println(err)
				os.Exit(1)
			}
			fmt.Printf("Message sent from server = %s\n", b)
		}
		fmt.Println(ws.CloseCode())
	}()

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		w.Write(scanner.Bytes())
	}
}
