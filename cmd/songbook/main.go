//go:build pocketbook

package main

import (
	"log"
	"os"
	"path/filepath"

	ink "github.com/dennwc/inkview"
)

func main() {
	if err := os.Chdir(filepath.Dir(os.Args[0])); err != nil {
		log.Printf("chdir: %v", err)
	}
	if err := ink.Run(newApp()); err != nil {
		log.Fatal(err)
	}
}