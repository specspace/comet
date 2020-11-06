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
)

const minecraftServerJarFileName = "minecraft_server.jar"

var (
	stdin  io.WriteCloser
	stdout io.ReadCloser
)

func main() {
	if err := downloadVanilla(); err != nil {
		log.Fatal(err)
	}

	go func() {
		log.Fatal(execServer())
	}()

	http.HandleFunc("/", wsEndpoint)
	log.Fatal(http.ListenAndServe(":8080", nil))
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

	go func() {
		scanner := bufio.NewScanner(stdout)
		for scanner.Scan() {
			conn.WriteMessage(1, scanner.Bytes())
		}
	}()

	for {
		_, p, err := conn.ReadMessage()
		if err != nil {
			return
		}
		p = append(p, 0x0A)
		io.Copy(stdin, bytes.NewReader(p))
	}
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

	if err := cmd.Start(); err != nil {
		return err
	}

	return cmd.Wait()
}
