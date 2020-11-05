package main

import (
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
)

const minecraftServerJarFileName = "minecraft_server.jar"

func main() {
	if err := downloadVanilla(); err != nil {
		log.Fatal(err)
	}

	log.Println(execServer(os.Stdin, os.Stdout))
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

func execServer(in io.Reader, out io.Writer) error {
	cmd := exec.Command("java", "-jar", minecraftServerJarFileName)
	cmd.Stdin = in
	cmd.Stdout = out

	if err := cmd.Start(); err != nil {
		return err
	}

	return cmd.Wait()
}
