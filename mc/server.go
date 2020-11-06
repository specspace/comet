package mc

import (
	"io"
	"net/http"
	"os"
	"os/exec"
)

const minecraftServerJarFilePath = "./minecraft_server.jar"

type Server struct {
	*exec.Cmd
}

func downloadServer(url string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	out, err := os.Create(minecraftServerJarFilePath)
	if err != nil {
		return nil
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}
