package main

import (
	"github.com/dkaps125/juke/app"
	"github.com/dkaps125/juke/config"
	"github.com/joho/godotenv"
)

func init() {
	godotenv.Load()
}

func main() {
	config := config.GetConfig()

	app := app.NewApp(config)
	app.Start()
}
