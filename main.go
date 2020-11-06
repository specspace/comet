package main

import (
	"bufio"
	"bytes"
	"github.com/gorilla/websocket"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"sync"
)

const minecraftServerJarFileName = "minecraft_server.jar"

var (
	stdin     io.Writer
	stdout    io.Reader
	connsLock sync.RWMutex
	conns     []*websocket.Conn
	wg        sync.WaitGroup
)

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
		connsLock.RLock()
		for _, conn := range conns {
			err := conn.WriteMessage(1, line)
			if err != nil {
				conn.Close()
			}
		}
		connsLock.RUnlock()
	}
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
