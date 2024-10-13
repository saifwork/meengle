package configs

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	ServiceName string
	ServiceHost string
	ServicePort string
	Version string
}

func NewConfig() *Config {
	c := Config{}
	c.initialise()
	return &c
}

func (c *Config) initialise() {
	// Load config
	if err := godotenv.Load(".env"); err != nil {
		log.Printf("Environment file missed. Err: %s", err)
	}

	if c.ServiceName = os.Getenv("SERVICE_NAME"); c.ServiceName == "" {
		log.Println("SERVICE_NAME missed on the environment variables, setting default to 'socket-service'")
		c.ServiceName = "socket-service"
	}

	if c.ServiceHost = os.Getenv("SERVICE_HOST"); c.ServiceHost == "" {
		log.Println("SERVICE_HOST missed on the environment variables, setting default to '0.0.0.0'")
		c.ServiceHost = "0.0.0.0"
	}

	if c.ServicePort = os.Getenv("SERVICE_PORT"); c.ServicePort == "" {
		log.Println("SERVICE_PORT missed on the environment variables, setting default to '8080'")
		c.ServicePort = "8080"
	}

	if c.Version = os.Getenv("VERSION"); c.Version == "" {
		log.Println("VERSION missed on the environment variables, setting default to '1.0.0'")
		c.Version = "1.0.0"
	}
}
