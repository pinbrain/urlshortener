package main

import "github.com/pinbrain/urlshortener/internal/app"

func main() {
	if err := app.Run(); err != nil {
		panic(err)
	}
}
