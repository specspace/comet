package main

import (
	"bufio"
	"bytes"
	"errors"
	"github.com/gorilla/websocket"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"regexp"
	"sync"
	"time"
)

const minecraftServerJarFileName = "minecraft_server.jar"

var (
	stdin     io.Writer
	stdout    io.Reader
	connsLock sync.RWMutex
	conns     []*websocket.Conn
	wg        sync.WaitGroup
	logRegexp = regexp.MustCompile("\\[(\\d{2}:\\d{2}:\\d{2})\\] \\[(.*)\\/(\\w*)\\]: (.*)")
)

type message struct {
	Timestamp time.Time `json:"timestamp"`
	Origin    string    `json:"origin"`
	Level     string    `json:"level"`
	Message   string    `json:"message"`
}

func main() {
	if err := downloadVanilla(); err != nil {
		log.Fatal(err)
	}

	wg.Add(1)
	go execServer()
	log.Println("starting mc")
	wg.Wait()
	log.Println("mc started")
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
			log.Println("could not parse line: ", err)
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

func downloadVanilla() error {
	resp, err := http.Get("https://launcher.mojang.com/v1/objects/35139deedbd5182953cf1caa23835da59ca3d7cd/server.jar")
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	out, err := os.Create(minecraftServerJarFileName)
	if err != nil {
		return nil
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

func execServer() error {
	cmd := exec.Command("java", "-jar", minecraftServerJarFileName)
	var err error
	stdin, err = cmd.StdinPipe()
	if err != nil {
		return err
	}
	stdout, err = cmd.StdoutPipe()
	if err != nil {
		return err
	}
	wg.Done()
	log.Println("i did")

	if err := cmd.Start(); err != nil {
		return err
	}

	return cmd.Wait()
}
