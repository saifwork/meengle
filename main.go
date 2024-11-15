package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/saifwork/socket-service/configs"
	"github.com/saifwork/socket-service/responses"
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
	r.POST("/static-event", StaticHandleWebHookEvent)
	r.GET("/ws", func(c *gin.Context) {
		hub.SetContext(c)
		socket.ServeWebsockets(hub, c.Writer, c.Request)
	})

	// Redirect route for the base URL
	r.GET("/", func(c *gin.Context) {
		c.Redirect(http.StatusFound, "https://meengle-f-build.vercel.app/")
	})

	// Initializing the client pairing bot
	go hub.PairWaitingClients(socket.Chat)
	go hub.PairWaitingClients(socket.AudioChat)
	go hub.PairWaitingClients(socket.VideoChat)

	h := config.ServiceHost
	p := config.ServicePort

	isHttps, err := strconv.Atoi(config.ServiceHTTPS)
	if err == nil && isHttps == 1 {
		crt := os.Getenv("SERVICE_CERT")
		key := os.Getenv("SERVICE_KEY")
		log.Printf("Starting the HTTPS server on %s:%s", h, p)
		err := r.RunTLS(fmt.Sprintf("%s:%s", h, p), crt, key)
		if err != nil {
			log.Fatalf("Error on starting the service: %v", err)
		}
	} else {
		log.Printf("Starting the HTTP server on %s:%s", h, p)
		err := r.Run(fmt.Sprintf("%s:%s", h, p))
		if err != nil {
			log.Fatalf("Error on starting the service: %v", err)
		}
	}

}

func HandleWebHookEvent(c *gin.Context) {

	log.Println("initla HandleWebHookEvent log")
	log.Println("inside HandleWebHookEvent")

	// Path to your script
	scriptPath := "go_meengle.sh"

	// Execute the script using exec.Command
	cmd := exec.Command("/bin/bash", scriptPath)

	// Run the command and capture any output or errors
	_, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Error running script: %v", err.Error())
		c.JSON(http.StatusInternalServerError, responses.NewErrorResponse(http.StatusBadRequest, err.Error(), nil))
		return
	}

	// Send a response back to acknowledge the webhook and script execution
	c.JSON(http.StatusOK, responses.NewSuccessResponse("service restarted"))
}

func StaticHandleWebHookEvent(c *gin.Context) {

	log.Println("initla StaticHandleWebHookEvent log")
	log.Println("inside StaticHandleWebHookEvent")

	// Path to your script
	scriptPath := "go_meengle.sh"

	// Execute the script using exec.Command
	cmd := exec.Command("/bin/bash", scriptPath)

	// Run the command and capture any output or errors
	_, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Error running script: %v", err.Error())
		c.JSON(http.StatusInternalServerError, responses.NewErrorResponse(http.StatusBadRequest, err.Error(), nil))
		return
	}

	// Send a response back to acknowledge the webhook and script execution
	c.JSON(http.StatusOK, responses.NewSuccessResponse("service restarted"))
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
