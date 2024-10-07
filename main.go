package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/saifwork/socket-service/configs"
	"github.com/saifwork/socket-service/socket"
)

func main() {

	// Load the configurations
	config := configs.NewConfig()

	// Initialize gin server
	r := gin.New()

	// Initialize Hub
	hub := socket.NewHub()
	go hub.Run()

	// Enable CORS middleware
	r.Use(CORSMiddleware())

	// Setup routes
	r.GET("/health", Healthcheck)
	r.GET("/ws", func(c *gin.Context) {
		hub.SetContext(c)
		socket.ServeWebsockets(hub, c.Writer, c.Request)
	})

	// Initializing the client pairing bot
	go hub.PairWaitingClients()

	p := config.ServicePort
	h := config.ServiceHost
	log.Printf("Serving at %s\n", fmt.Sprintf("%s:%s", h, p))
	if err := r.Run(fmt.Sprintf("%s:%s", h, p)); err != nil {
		log.Fatalf("Fail to start the server on %s:%s ", h, p)
	}
}

func Healthcheck(c *gin.Context) {
	version := os.Getenv("VERSION")
	if version == "" {
		version = "OK"
	}
	response := map[string]string{
		"status":  "up",
		"version": version,
	}
	c.JSON(http.StatusOK, response)
}

func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, accept, origin, Cache-Control, X-Requested-With")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS, GET, PUT DELETE")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}
