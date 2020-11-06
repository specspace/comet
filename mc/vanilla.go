package mc

import (
	"encoding/json"
	"errors"
	"net/http"
	"os/exec"
)

const versionManifestURL = "https://launchermeta.mojang.com/mc/game/version_manifest.json"

func NewVanillaServer(version string) (Server, error) {
	if err := downloadVanillaServer(version); err != nil {
		return Server{}, err
	}

	return Server{
		Cmd: exec.Command("java", "-jar", minecraftServerJarFilePath),
	}, nil
}

func downloadVanillaServer(version string) error {
	url, err := fetchVanillaServerURL(version)
	if err != nil {
		return err
	}

	return downloadServer(url)
}

func fetchVanillaServerURL(version string) (string, error) {
	gameVersionInfoURL, err := fetchGameVersionInfoURL(version)
	if err != nil {
		return "", err
	}

	resp, err := http.Get(gameVersionInfoURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var gameVersionInfo struct {
		Downloads struct {
			Server struct {
				URL string `json:"url"`
			} `json:"server"`
		} `json:"downloads"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&gameVersionInfo); err != nil {
		return "", err
	}

	return gameVersionInfo.Downloads.Server.URL, nil
}

func fetchGameVersionInfoURL(version string) (string, error) {
	resp, err := http.Get(versionManifestURL)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	var versionManifest struct {
		Latest struct {
			Release  string `json:"release"`
			Snapshot string `json:"snapshot"`
		} `json:"latest"`
		Versions []struct {
			ID  string `json:"id"`
			URL string `json:"url"`
		} `json:"versions"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&versionManifest); err != nil {
		return "", err
	}

	if version == LatestServerVersion {
		version = versionManifest.Latest.Release
	} else if version == LatestSnapshotServerVersion {
		version = versionManifest.Latest.Snapshot
	}

	for _, gv := range versionManifest.Versions {
		if gv.ID == version {
			return gv.URL, nil
		}
	}

	return "", errors.New("game version not found")
}
