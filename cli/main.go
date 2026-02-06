package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/ankit-chaubey/media-metadata-surgery/core/jpg"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: surgery view <image.jpg>")
		os.Exit(1)
	}

	cmd := os.Args[1]
	file := os.Args[2]

	if cmd != "view" {
		log.Fatal("Only 'view' command supported in v0.1")
	}

	ext := strings.ToLower(filepath.Ext(file))
	if ext != ".jpg" && ext != ".jpeg" {
		log.Fatal("Only JPG/JPEG files supported")
	}

	if err := jpg.ViewEXIF(file); err != nil {
		log.Fatal(err)
	}
}
