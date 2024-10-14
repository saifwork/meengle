package main

import (
	"fmt"
	"io/ioutil"
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
	r.POST("/event", HandleWebHookEvent)
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

func HandleWebHookEvent(c *gin.Context) {
	// Read the body of the POST request
	body, err := ioutil.ReadAll(c.Request.Body)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Unable to read request body"})
		return
	}

	// Print the body to the console
	fmt.Println("Received Webhook Data: ", string(body))

	// You can also log it to a file if needed

	// Send a response back to acknowledge the webhook was received
	c.JSON(http.StatusOK, gin.H{"status": "Webhook received successfully"})
}

func Healthcheck(c *gin.Context) {

	log.Println("Healthcheck handler called")
	version := os.Getenv("VERSION")
	if version == "" {
		version = "OK"
	}
	response := map[string]string{
		"status":  "up",
		"version": version,
	}

	log.Println(response)
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
