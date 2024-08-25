package main

import (
	"os"
)

func notMain() {
	os.Exit(0)
}

func main() {
	notMain()
	os.Exit(0) // want "direct call os.Exit in main package in main function"
}
