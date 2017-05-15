package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type hexkitMap map[string]json.RawMessage
type layersList []map[string]json.RawMessage
type tilesList []map[string]interface{}

var fileList = make(map[string][]string, 4096)

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

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage:", os.Args[0], "CollectionPath... MapPath")
		return
	}
	pathSeparator := "/"
	if os.PathSeparator != '/' {
		pathSeparator = "\\\\"
	}
	// Build the list of PNG files
	for i := 0; i <= (len(os.Args) - 2); i++ {
		err := filepath.Walk(os.Args[1], pathMap)
		if err != nil {
			log.Fatal(err)
		}
	var hexMap hexkitMap
	var layers layersList
	}
	// Read the file
	mapFile := os.Args[len(os.Args)-1]
	mapBlob, err := ioutil.ReadFile(mapFile)
	if err != nil {
		log.Fatal("error:", err)
	}
	// Decode it in hexMap
	err = json.Unmarshal(mapBlob, &hexMap)
	if err != nil {
		log.Fatal("error:", mapFile, ":", err)
	}
	// Get the layers list
	layersBlob, ok := hexMap["layers"]
	if !ok {
		log.Fatal("error: no layer in", mapFile)
	}
	err = json.Unmarshal(layersBlob, &layers)
	if err != nil {
		log.Fatal("error:", mapFile, ":", err)
	}
	layersModified := false
	for _, v := range layers {
		var tiles tilesList
		tilesBlob, ok := v["tiles"]
		if !ok {
			log.Println("error: no tiles in", mapFile)
			continue
		}
		err = json.Unmarshal(tilesBlob, &tiles)
		if err != nil {
			log.Println("error:", mapFile, ":", err)
			continue
		}
		tilesModified := false
		for _, t := range tiles {
			// Ignore undefined tiles
			if t == nil {
				continue
			}
			sourceBlob, ok := t["source"]
			if !ok {
				log.Println("error: tile with no source in", mapFile)
				continue
			}
			source, ok := sourceBlob.(string)
			if !ok {
				log.Println("error: incorrect source found in", mapFile)
				continue
			}
			if source[:6] == "Blank:" {
				continue
			}
			fileName := filepath.Base(source)
			pathList, ok := fileList[fileName]
			if !ok {
				log.Println("error: tile", source, "not found in", mapFile)
				continue
			}
			firstSplit := strings.SplitN(source, ":", 2)
			if len(firstSplit) >= 2 {
				targetCollection := firstSplit[0]
				targetPath := firstSplit[1]
				var bestMatch int
				var selected string
			pathSearch:
				for _, p := range pathList {
					splitPath := strings.Split(p, string(os.PathSeparator))
					for k, segment := range splitPath {
						if segment == targetCollection {
							foundPath := pathSeparator + strings.Join(splitPath[k+1:], pathSeparator)
							if targetPath == foundPath {
								selected = targetCollection + ":" + foundPath
								break pathSearch
							}
							if len(foundPath) > bestMatch {
								bestMatch = len(foundPath)
								selected = targetCollection + ":" + foundPath
							}
						}
					}
					if selected != "" {
						tilesModified = true
						t["source"] = selected
					}
				}
			}
		}
		if tilesModified {
			tilesBlob, err := json.Marshal(tiles)
			if err != nil {
				log.Println("error:", err)
			}
			v["tiles"] = tilesBlob
			layersModified = true
		}
	}
	if layersModified {
		layersBlob, err := json.Marshal(layers)
		if err != nil {
			log.Println("error:", err)
		}
		hexMap["layers"] = layersBlob
	}
	b, err := json.Marshal(hexMap)
	if err != nil {
		log.Println("error:", err)
	}
	_, err = os.Stdout.Write(b)
	if err != nil {
		log.Println("error:", err)
	}

}
