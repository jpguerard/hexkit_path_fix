package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

type jsonObjectRaw map[string]json.RawMessage
type jsonObject map[string]interface{}

var fileList = make(map[string][]string, 4096)

func findHomeDir() string {
	u, err := user.Current()
	failOnError(err)
	return u.HomeDir
}

func readSettingsBlob(userConfig string) ([]byte, error) {
	settings := filepath.Join(userConfig, "hex-kit", "Settings")
	settingsBlob, err := ioutil.ReadFile(settings)
	return settingsBlob, err
}

// https://electron.atom.io/docs/api/app/#appgetpathname
func getSettings() jsonObjectRaw {
	var userConfig string
	var settingsBlob []byte
	var jsonSettingsRaw jsonObjectRaw
	var err error
	switch runtime.GOOS {
	case "darwin":
		homeDir := findHomeDir()
		userConfig = filepath.Join(homeDir, "Library", "Application Support")
		settingsBlob, err = readSettingsBlob(userConfig)
	case "linux":
		userConfig = os.Getenv("XDG_CONFIG_HOME")
		if userConfig != "" {
			settingsBlob, err = readSettingsBlob(userConfig)
		}
		if userConfig == "" || err != nil {
			homeDir := findHomeDir()
			userConfig = filepath.Join(homeDir, ".config")
			settingsBlob, err = readSettingsBlob(userConfig)
		}
	case "windows":
		userConfig = os.Getenv("APPDATA")
		if userConfig == "" {
			log.Fatal("Unable to find user config (no APPDATA environment variable)")
		}
		settingsBlob, err = readSettingsBlob(userConfig)
	}
	// Did the log read succeed
	if err != nil {
		log.Fatal(err)
	}
	err = json.Unmarshal(settingsBlob, &jsonSettingsRaw)
	if err != nil {
		log.Fatal(err)
	}
	return jsonSettingsRaw
}

// Search fon all png files under the current path
func pathMap(path string, info os.FileInfo, err error) error {
	if err != nil {
		fmt.Println(err)
	}
	name := info.Name()
	lenPath := len(path)
	if lenPath > 16 && path[lenPath-4:] == ".png" {
		fileList[name] = append(fileList[name], path)
	}
	return nil
}

func failOnError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage:", os.Args[0], "CollectionPath... MapPath")
		return
	}
	// Build the list of PNG files
	for i := 0; i <= (len(os.Args) - 2); i++ {
		err := filepath.Walk(os.Args[1], pathMap)
		failOnError(err)
	}
	// Read the file
	mapFile := os.Args[len(os.Args)-1]
	mapBlob, err := ioutil.ReadFile(mapFile)
	failOnError(err)

	// Decode it in hexMap
	var hexMap jsonObjectRaw
	err = json.Unmarshal(mapBlob, &hexMap)
	failOnError(err)

	// Get the layers list
	layersBlob, ok := hexMap["layers"]
	if !ok {
		log.Fatal("Map format error (no layers found)")
	}
	var layers []jsonObjectRaw
	err = json.Unmarshal(layersBlob, &layers)
	failOnError(err)

	// Search for tiles
	layersModified := false
	for i, v := range layers {
		var tiles []jsonObject
		tilesBlob, ok := v["tiles"]
		if !ok {
			log.Fatal("Map format error: no tiles in layer", i+1)
		}
		err = json.Unmarshal(tilesBlob, &tiles)
		if err != nil {
			log.Fatal("Map format error in layer", i+1, ":", err)
		}
		// Search for the source of tiles
		tilesModified := false
		for j, t := range tiles {
			// Ignore undefined tiles
			if t == nil {
				continue
			}
			sourceBlob, ok := t["source"]
			if !ok {
				log.Println("Layer", i+1, "Tile", j+1, "no tile source found")
				continue
			}
			source, ok := sourceBlob.(string)
			if !ok {
				log.Println("Layer", i+1, "Tile", j+1, "the tile source is incorrect (not a string)")
				continue
			}
			// Skip the default blank tiles
			if source[:6] == "Blank:" {
				continue
			}
			// Have we found the tile
			nameSelect := regexp.MustCompile(`[^:/\\]+$`)
			fileName := nameSelect.FindString(source)
			pathList, ok := fileList[fileName]
			if !ok {
				log.Println("Layer", i+1, "Tile", j+1, "unable to find tile image file for", source)
				continue
			}
			// Search for the current path in the
			firstSplit := strings.SplitN(source, ":", 2)
			if len(firstSplit) < 2 {
				log.Println("Layer", i+1, "Tile", j+1, "incorrect source:", source)
				continue
			}
			targetCollection := firstSplit[0]
			targetPath := firstSplit[1]
			var bestMatch int
			var selected string
		pathSearch:
			for _, p := range pathList {
				splitPath := strings.Split(p, string(os.PathSeparator))
				for k, segment := range splitPath {
					if segment != targetCollection {
						continue
					}
					foundPath := "/" + strings.Join(splitPath[k+1:], "/")
					if targetPath == foundPath {
						selected = targetCollection + ":" + foundPath
						break pathSearch
					}
					if len(foundPath) > bestMatch {
						bestMatch = len(foundPath)
						selected = targetCollection + ":" + foundPath
					}
				}
				// A new value was found: update the source
				if selected != "" {
					tilesModified = true
					t["source"] = selected
				}
			}
		}
		if tilesModified {
			tilesBlob, err := json.Marshal(tiles)
			failOnError(err)
			v["tiles"] = tilesBlob
			layersModified = true
		}
	}
	if layersModified {
		layersBlob, err := json.Marshal(layers)
		failOnError(err)
		hexMap["layers"] = layersBlob
	}
	b, err := json.Marshal(hexMap)
	failOnError(err)
	_, err = os.Stdout.Write(b)
	failOnError(err)
}
