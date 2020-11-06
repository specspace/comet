package main

import (
	"bufio"
	"bytes"
	"errors"
	"github.com/gorilla/websocket"
	"github.com/specspace/comet/mc"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"sync"
	"time"
)

var (
	stdin     io.Writer
	stdout    io.Reader
	connsLock sync.RWMutex
	conns     []*websocket.Conn
	logRegexp = regexp.MustCompile("\\[(\\d{2}:\\d{2}:\\d{2})\\] \\[(.*)\\/(\\w*)\\]: (.*)")
)

type message struct {
	Timestamp time.Time `json:"timestamp"`
	Origin    string    `json:"origin"`
	Level     string    `json:"level"`
	Message   string    `json:"message"`
}

func main() {
	server, err := mc.NewVanillaServer(mc.LatestServerVersion)
	if err != nil {
		log.Fatal(err)
	}

	stdin, err = server.StdinPipe()
	if err != nil {
		log.Fatal(err)
	}
	stdout, err = server.StdoutPipe()
	if err != nil {
		log.Fatal(err)
	}

	if err := server.Start(); err != nil {
		log.Fatal(err)
	}

	go sendOutLoop()

	http.HandleFunc("/", wsEndpoint)
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func sendOutLoop() {
	reader := bufio.NewReader(stdout)
	for {
		line, err := reader.ReadBytes('\n')
		if err != nil && err != io.EOF {
			break
		}
		os.Stdout.Write(line)

		message, err := parseLine(line)
		if err != nil {
			log.Println("could not parse line: ", string(line))
			continue
		}

		connsLock.RLock()
		for _, conn := range conns {
			err := conn.WriteJSON(message)
			if err != nil {
				log.Println(err)
				conn.Close()
			}
		}
		connsLock.RUnlock()
	}
}

func parseLine(data []byte) (message, error) {
	//  timestamp  origin        level  message
	// [14:10:49] [Server thread/INFO]: There are 0 of a max of 20 players online:
	matches := logRegexp.FindStringSubmatch(string(data))
	if len(matches) < 5 {
		return message{}, errors.New("invalid console output")
	}

	timestamp, err := time.Parse("15:04:05", matches[1])
	if err != nil {
		return message{}, err
	}

	return message{
		Timestamp: timestamp,
		Origin:    matches[2],
		Level:     matches[3],
		Message:   matches[4],
	}, nil
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func wsEndpoint(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	conn.WriteMessage(1, []byte("hello"))

	connsLock.Lock()
	conns = append(conns, conn)
	connsLock.Unlock()

	for {
		_, p, err := conn.ReadMessage()
		if err != nil {
			return
		}
		p = append(p, 0x0A)
		io.Copy(stdin, bytes.NewReader(p))
	}
	conn.Close()
}
