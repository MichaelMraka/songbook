//go:build !pocketbook

package main

import "fmt"

func main() {
	fmt.Println("Build the PocketBook binary with: make build")
	fmt.Println("(requires Docker image with the PocketBook Go SDK, tag pocketbook)")
}