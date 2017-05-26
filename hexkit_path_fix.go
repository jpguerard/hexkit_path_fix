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
)

type jsonObjectRaw map[string]json.RawMessage
type jsonObject map[string]interface{}

type tilePosition struct {
	collection string
	path       string
}

// Log to standard error
var stderr = log.New(os.Stderr, "", 0)

var fileList = make(map[string][]tilePosition, 4096)

func getJSONRawSlice(r jsonObjectRaw, k string) (*[]jsonObjectRaw, error) {
	blob, ok := r[k]
	if !ok {
		return nil, fmt.Errorf("No \"%s\" key in object", k)
	}
	var rawSlice []jsonObjectRaw
	err := json.Unmarshal(blob, &rawSlice)
	if err != nil {
		return nil, err
	}
	return &rawSlice, nil
}

func getJSONSlice(r jsonObjectRaw, k string) (*[]jsonObject, error) {
	blob, ok := r[k]
	if !ok {
		return nil, fmt.Errorf("No \"%s\" key in object", k)
	}
	var decodedSlice []jsonObject
	err := json.Unmarshal(blob, &decodedSlice)
	if err != nil {
		return nil, err
	}
	return &decodedSlice, nil
}

func getJSONRawObject(r jsonObjectRaw, k string) (*jsonObjectRaw, error) {
	blob, ok := r[k]
	if !ok {
		return nil, fmt.Errorf("No \"%s\" key in object", k)
	}
	var decodedObject jsonObjectRaw
	err := json.Unmarshal(blob, &decodedObject)
	if err != nil {
		return nil, err
	}
	return &decodedObject, nil
}

func getJSONObject(r jsonObjectRaw, k string) (*jsonObject, error) {
	blob, ok := r[k]
	if !ok {
		return nil, fmt.Errorf("No \"%s\" key in object", k)
	}
	var decodedObject jsonObject
	err := json.Unmarshal(blob, &decodedObject)
	if err != nil {
		return nil, err
	}
	return &decodedObject, nil
}

func findHomeDir() string {
	u, err := user.Current()
	if err != nil {
		log.Fatal("error: unable to get the username:", err)
	}
	return u.HomeDir
}

func readSettingsBlob(userConfig string) ([]byte, error) {
	settings := filepath.Join(userConfig, "hex-kit", "Settings")
	log.Print("Reading user settings from: ", settings)
	settingsBlob, err := ioutil.ReadFile(settings)
	return settingsBlob, err
}

// https://electron.atom.io/docs/api/app/#appgetpathname
func getSettings() jsonObjectRaw {
	var userConfig string
	var settingsBlob []byte
	var settingsRaw jsonObjectRaw
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
	err = json.Unmarshal(settingsBlob, &settingsRaw)
	if err != nil {
		log.Fatal(err)
	}
	return settingsRaw
}

// Search fon all png files under the current path
func pathMap(collectionName, basePath string) filepath.WalkFunc {
	return func(path string, info os.FileInfo, err error) error {
		if err != nil {
			stderr.Println("Warning: while searching for PNG files: ", err)
			return nil
		}
		tileName := info.Name()
		lenPath := len(path)
		if lenPath > 4 && path[lenPath-4:] == ".png" {
			relPathTile, err := filepath.Rel(basePath, path)
			if err != nil {
				log.Fatal("fatal:", path, "is not under", basePath, ":", err)
			}
			var tp tilePosition
			tp.collection = collectionName
			tp.path = relPathTile
			fileList[tileName] = append(fileList[tileName], tp)
		}
		return nil
	}
}

func main() {
	if len(os.Args) != 3 {
		log.Println("Usage:", os.Args[0], "HexkitPath MapPath")
		return
	}

	// Prepare regexp
	nameSelect := regexp.MustCompile(`[^:/\\]+$`)
	tilepathCut := regexp.MustCompile(`:/{0,2}`)
	// Build the list of collections
	collectionsDir := make(map[string]string)
	settings := getSettings()
	collections, err := getJSONRawObject(settings, "tiles")
	if err != nil {
		log.Fatal("Error: unable to parse user settings:", err)
	}
	for name, collectionBlob := range *collections {
		var collection jsonObject
		err := json.Unmarshal(collectionBlob, &collection)
		if err != nil {
			log.Fatal("Error: unable to parse user settings:", err)
		}
		// Ignore source if hidden
		hiddenIntf, ok := collection["hidden"]
		if ok {
			hidden, isBool := hiddenIntf.(bool)
			if isBool && hidden {
				continue
			}
		}
		pathIntf, ok := collection["path"]
		if !ok {
			log.Fatal("Error: unable to parse user settings: no path for", name)
		}
		path, ok := pathIntf.(string)
		if !ok {
			log.Fatal("Error: unable to parse user settings: the path for", name, "is not a string")
		}
		// Relative collection path
		if !filepath.IsAbs(path) {
			collectionsDir[name] = filepath.Join(os.Args[1], "resources", "app.asar.unpacked", path)
			continue
		}
		// Absolute collection path
		collectionsDir[name] = path
	}

	// Build the list of PNG files
	for name, path := range collectionsDir {
		err := filepath.Walk(path, pathMap(name, path))
		if err != nil {
			log.Fatal(err)
		}
	}
	// Read the file
	mapFile := os.Args[2]
	mapBlob, err := ioutil.ReadFile(mapFile)
	if err != nil {
		log.Fatal(err)
	}

	// Decode it in hexMap
	var hexMap jsonObjectRaw
	err = json.Unmarshal(mapBlob, &hexMap)
	if err != nil {
		log.Fatal(err)
	}

	// Get the layers list
	layers, err := getJSONRawSlice(hexMap, "layers")
	if err != nil {
		log.Fatal("Map format error", err)
	}

	// Search for tiles
	layersModified := false
	for i, v := range *layers {
		tiles, err := getJSONSlice(v, "tiles")
		if err != nil {
			log.Fatal("Map format error in layer", i+1, ":", err)
		}
		// Search for the source of tiles
		tilesModified := false
		for j, t := range *tiles {
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
			fileName := nameSelect.FindString(source)
			pathList, ok := fileList[fileName]
			if !ok {
				log.Println("Layer", i+1, "Tile", j+1, "unable to find tile image file for", source)
				continue
			}
			// Search for the current path in the
			firstSplit := tilepathCut.Split(source, 2)
			if len(firstSplit) < 2 {
				log.Println("Layer", i+1, "Tile", j+1, "incorrect source:", source)
				continue
			}
			targetCollection := firstSplit[0]
			targetPath := firstSplit[1]
			var bestScore int
			var selected tilePosition
		pathSearch:
			for _, p := range pathList {
				if targetCollection == p.collection && targetPath == p.path {
					break pathSearch
				}
				currentScore := len(p.path)
				if targetCollection == p.collection && (currentScore+256) > bestScore {
					bestScore = currentScore + 256
					selected = p
				}
				if currentScore > bestScore {
					bestScore = currentScore
					selected = p
				}
			}
			// A new value was found: update the source
			if selected.collection != "" && selected.path != "" {
				tilesModified = true
				t["source"] = selected.collection + "://" + selected.path
			}
		}
		if tilesModified {
			tilesBlob, err := json.Marshal(tiles)
			if err != nil {
				log.Fatal(err)
			}
			v["tiles"] = tilesBlob
			layersModified = true
		}
	}
	if layersModified {
		layersBlob, err := json.Marshal(layers)
		if err != nil {
			log.Fatal(err)
		}
		hexMap["layers"] = layersBlob
	}
	b, err := json.Marshal(hexMap)
	if err != nil {
		log.Fatal(err)
	}
	_, err = os.Stdout.Write(b)
	if err != nil {
		log.Fatal(err)
	}
}
