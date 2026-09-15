
package main

import (
	"log"

	"github.com/JerryJeager/raglearn/cmd"
	"github.com/JerryJeager/raglearn/config"
)

func init() {
	config.LoadEnv()
	config.ConnectToDB()
}

func main() {
	log.Println("Starting Server")


	cmd.ExecuteApiRoutes()
}

	