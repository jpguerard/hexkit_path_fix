package main

import (
	"encoding/json"
	"errors"
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

var nameSelect = regexp.MustCompile(`[^:/\\]+$`)
var tilepathCut = regexp.MustCompile(`:/{0,2}`)

func getJSONRawSlice(r jsonObjectRaw, k string) (*[]jsonObjectRaw, error) {
	blob, ok := r[k]
	if !ok {
		return nil, fmt.Errorf("no \"%s\" key in object", k)
	}
	var rawSlice []jsonObjectRaw
	if err := json.Unmarshal(blob, &rawSlice); err != nil {
		return nil, fmt.Errorf("JSON decode failed: %w", err)
	}
	return &rawSlice, nil
}

func getJSONSlice(r jsonObjectRaw, k string) (*[]jsonObject, error) {
	blob, ok := r[k]
	if !ok {
		return nil, fmt.Errorf("no \"%s\" key in object", k)
	}
	var decodedSlice []jsonObject
	if err := json.Unmarshal(blob, &decodedSlice); err != nil {
		return nil, fmt.Errorf("JSON decode failed: %w", err)
	}
	return &decodedSlice, nil
}

func getJSONRawObject(r jsonObjectRaw, k string) (*jsonObjectRaw, error) {
	blob, ok := r[k]
	if !ok {
		return nil, fmt.Errorf("no \"%s\" key in object", k)
	}
	var decodedObject jsonObjectRaw

	if err := json.Unmarshal(blob, &decodedObject); err != nil {
		return nil, fmt.Errorf("JSON decode failed: %w", err)
	}
	return &decodedObject, nil
}

func readMapFile(path string) (*jsonObjectRaw, error) {
	// Read the file
	mapBlob, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read failed: %w", err)
	}
	// Decode it in hexMap
	var hexMap jsonObjectRaw
	if err = json.Unmarshal(mapBlob, &hexMap); err != nil {
		return nil, fmt.Errorf("JSON decode failed: %w", err)
	}
	return &hexMap, nil
}

func findHomeDir() (string, error) {
	u, err := user.Current()
	if err != nil {
		return "", fmt.Errorf("unable to get user information: %w", err)
	}
	return u.HomeDir, nil
}

func readSettingsBlob(userConfig string) ([]byte, error) {
	settings := filepath.Join(userConfig, "hex-kit", "Settings")
	stderr.Print("Reading user settings from: ", settings)
	settingsBlob, err := ioutil.ReadFile(settings)
	return settingsBlob, err
}

// https://electron.atom.io/docs/api/app/#appgetpathname
func getSettings() (jsonObjectRaw, error) {
	var userConfig string
	var settingsBlob []byte
	var settingsRaw jsonObjectRaw
	var err, sysErr error
	switch runtime.GOOS {
	case "darwin":
		homeDir, err := findHomeDir()
		if err != nil {
			return nil, err
		}
		userConfig = filepath.Join(homeDir, "Library", "Application Support")
		settingsBlob, sysErr = readSettingsBlob(userConfig)
	case "linux":
		userConfig = os.Getenv("XDG_CONFIG_HOME")
		if userConfig != "" {
			settingsBlob, err = readSettingsBlob(userConfig)
		}
		if userConfig == "" || err != nil {
			homeDir, err := findHomeDir()
			if err != nil {
				return nil, err
			}
			userConfig = filepath.Join(homeDir, ".config")
			settingsBlob, sysErr = readSettingsBlob(userConfig)
		}
	case "windows":
		userConfig = os.Getenv("APPDATA")
		if userConfig == "" {
			return nil, errors.New("Error: unable to find user config (no APPDATA environment variable)")
		}
		settingsBlob, sysErr = readSettingsBlob(userConfig)
	}
	// Did the log read succeed
	if sysErr != nil {
		return nil, fmt.Errorf("unable to read settings: %w", err)
	}
	err = json.Unmarshal(settingsBlob, &settingsRaw)
	if err != nil {
		return nil, fmt.Errorf("unable to decode settings: %w", err)
	}
	return settingsRaw, nil
}

// Search fon all png files under the current path
func pathMap(collectionName, basePath string, fileList *map[string][]tilePosition) filepath.WalkFunc {
	return func(path string, info os.FileInfo, err error) error {
		if err != nil {
			stderr.Println("Warning: while searching for PNG files: ", err)
			return nil
		}
		tileName := info.Name()
		lenPath := len(path)
		if lenPath > 4 && path[lenPath-4:] == ".png" {
			relPathTile, _ := filepath.Rel(basePath, path)
			var tp tilePosition
			tp.collection = collectionName
			tp.path = filepath.ToSlash(filepath.Clean(relPathTile))
			(*fileList)[tileName] = append((*fileList)[tileName], tp)
		}
		return nil
	}
}

func getCollectionDir(settings jsonObjectRaw) (*map[string]string, error) {
	// Build the list of collections
	collectionsDir := make(map[string]string)
	collections, err := getJSONRawObject(settings, "tiles")
	if err != nil {
		return nil, fmt.Errorf("unable to access the \"tiles\" list: %w", err)
	}
	for name, collectionBlob := range *collections {
		var collection jsonObject
		if err := json.Unmarshal(collectionBlob, &collection); err != nil {
			return nil, fmt.Errorf("unable to parse tiles[%s]: %w", name, err)
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
			return nil, fmt.Errorf("no path for %s: %w", name, err)
		}
		path, ok := pathIntf.(string)
		if !ok {
			return nil, fmt.Errorf("the path for %s is not a string: %w", name, err)
		}
		// Relative collection path
		if !filepath.IsAbs(path) {
			collectionsDir[name] = filepath.Join(os.Args[1], "resources", "app.asar.unpacked", path)
			continue
		}
		// Absolute collection path
		collectionsDir[name] = path
	}
	return &collectionsDir, nil

}

func tileUpdate(t *jsonObject, fileList map[string][]tilePosition) (modified bool, err error) {
	sourceBlob, ok := (*t)["source"]
	if !ok {
		return false, errors.New("no tile source found")
	}
	source, ok := sourceBlob.(string)
	if !ok {
		return false, errors.New("incorrect tile source (not a string)")
	}
	// Skip the default blank tiles
	if source[:6] == "Blank:" {
		return false, nil
	}
	// Have we found the tile
	fileName := nameSelect.FindString(source)
	pathList, ok := fileList[fileName]
	if !ok {
		return false, fmt.Errorf("unknown tile %s (%s)", fileName, source)
	}
	// Search for the current path in the
	firstSplit := tilepathCut.Split(source, 2)
	if len(firstSplit) < 2 {
		return false, fmt.Errorf("no \":\" in source (%s)", source)
	}
	targetCollection := firstSplit[0]
	targetPath := firstSplit[1]
	var bestScore int
	var selected tilePosition
pathSearch:
	for _, p := range pathList {
		// The tiles still exists at the same place. We keep it.
		if targetCollection == p.collection && targetPath == p.path {
			break pathSearch
		}
		currentScore := len(p.path)
		// A tile in the same collection will be preferred
		if targetCollection == p.collection && (currentScore+256) > bestScore {
			bestScore = currentScore + 256
			selected = p
		}
		// Otherwise, take the longer path
		if currentScore > bestScore {
			bestScore = currentScore
			selected = p
		}
	}
	// A new value was found: update the source
	if selected.collection != "" && selected.path != "" {
		modified = true
		(*t)["source"] = selected.collection + "://" + selected.path
	}
	return modified, nil

}

func updateMapFile(mapFile *jsonObjectRaw, fileList map[string][]tilePosition) error {
	// Get the layers list
	layers, err := getJSONRawSlice(*mapFile, "layers")
	if err != nil {
		return fmt.Errorf("Map format error: %w", err)
	}
	// Search each layer
	layersModified := false
	for i, v := range *layers {
		tiles, err := getJSONSlice(v, "tiles")
		if err != nil {
			return fmt.Errorf("Layer %d: Map format error: %w", i+1, err)
		}
		// Update all tiles
		tilesModified := false
		for j, t := range *tiles {
			// Ignore undefined tiles
			if t == nil {
				continue
			}
			modified, err := tileUpdate(&t, fileList)
			if err != nil {
				stderr.Println("Warning: layer", i+1, "tile", j+1, ":", err)
				continue
			}
			tilesModified = tilesModified || modified
		}
		if tilesModified {
			tilesBlob, err := json.Marshal(tiles)
			if err != nil {
				return err
			}
			v["tiles"] = tilesBlob
			layersModified = true
		}
	}
	if layersModified {
		layersBlob, err := json.Marshal(layers)
		if err != nil {
			return err
		}
		(*mapFile)["layers"] = layersBlob
	}
	return nil
}

func main() {
	if len(os.Args) != 3 {
		stderr.Println("Usage:", os.Args[0], "HexkitPath MapPath")
		return
	}
	// Read the settngs and get the
	settings, err := getSettings()
	if err != nil {
		stderr.Fatal("Unable to read user settings: ", err)
	}
	collectionsDir, err := getCollectionDir(settings)
	if err != nil {
		stderr.Fatal("Unable to read the list of collections: ", err)
	}
	// Build the list of PNG files
	fileList := make(map[string][]tilePosition, 4096)
	for name, path := range *collectionsDir {
		if err := filepath.Walk(path, pathMap(name, path, &fileList)); err != nil {
			stderr.Fatal(err)
		}
	}
	// Read the file
	hexMap, err := readMapFile(os.Args[2])
	if err != nil {
		stderr.Fatal("Error: unable to read map file: ", err)
	}
	if err = updateMapFile(hexMap, fileList); err != nil {
		stderr.Fatal(err)
	}
	b, err := json.Marshal(*hexMap)
	if err != nil {
		stderr.Fatal(err)
	}
	_, err = os.Stdout.Write(b)
	if err != nil {
		stderr.Fatal(err)
	}
}
