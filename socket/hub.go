package socket

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	log "log"

	"math/rand"

	"github.com/gin-gonic/gin"
	"github.com/saifwork/socket-service/responses"
	"github.com/saifwork/socket-service/types"
)

// Hub maintains the set of active clients and broadcasts messages to the
// clients.
type Hub struct {
	// Gin context
	ctx *gin.Context

	// Registered clients.
	clients map[*Client]time.Time

	// Inbound messages from the clients.
	Broadcast chan *ClientMessage

	// Register requests from the clients.
	register chan *Client

	// Unregister requests from clients.
	unregister chan *Client

	mu sync.Mutex // Mutex to protect the clients map
}

func NewHub() *Hub {
	hub := &Hub{
		Broadcast:  make(chan *ClientMessage),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[*Client]time.Time),
	}

	return hub
}

func (h *Hub) SetContext(c *gin.Context) {
	h.ctx = c
}

func (h *Hub) SendMessage(clientMessage *ClientMessage) {
	select {
	case clientMessage.Client.send <- clientMessage.Message:
	default:
		close(clientMessage.Client.send)
		delete(h.clients, clientMessage.Client)
		log.Printf("[%s] Total clients %d closing connection for: %s", time.Now(), len(h.clients), clientMessage.Client.ID)
	}
}

func (hub *Hub) Run() {

	// Start cleaning process
	hub.cleanClients()

	for {
		select {
		case client := <-hub.register:
			hub.mu.Lock()
			hub.clients[client] = time.Now()
			hub.mu.Unlock()

			// Broadcast the updated user count
			hub.BroadcastUserCount()

		case client := <-hub.unregister:
			hub.mu.Lock()
			if _, ok := hub.clients[client]; ok {
				delete(hub.clients, client)
				close(client.send)
			}
			hub.mu.Unlock()

			// Broadcast the updated user count
			hub.BroadcastUserCount()

		case message := <-hub.Broadcast:
			// Handle normal broadcast logic
			for client := range hub.clients {
				select {
				case client.send <- message.Message:
				default:
					close(client.send)
					delete(hub.clients, client)
				}
			}
		}
	}
}

// Function to find a client by their ID
func (h *Hub) GetClientByID(id string) *Client {
	for client := range h.clients {
		if client.User.ID == id {
			return client // Return the whole client object if ID matches
		}
	}
	return nil // Return nil if no client with the given ID is found
}

// GetWaitingClientByUID searches for a client in the hub by its UID and checks if they are waiting.
func (h *Hub) GetWaitingClientByUID(uid string) *Client {
	h.mu.Lock()
	defer h.mu.Unlock()

	for client := range h.clients {
		if client.ID == uid {
			return client
		}
	}

	return nil // Return nil if no matching client is found
}

func (h *Hub) GetWaitingClients() []*Client {

	h.mu.Lock()         // Lock before accessing
	defer h.mu.Unlock() // Unlock after

	var waitingClients []*Client
	for client := range h.clients {

		log.Println(client)
		if client.IsWaiting {
			waitingClients = append(waitingClients, client)
		}
	}

	// log.Println(waitingClients)
	return waitingClients
}

func (h *Hub) PairWaitingClients() {

	log.Println("Starting PairWaitingClients")

	for {
		time.Sleep(1 * time.Second) // Check every 5 seconds

		waitingClients := h.GetWaitingClients() // Fetch waiting clients

		if len(waitingClients) < 2 {
			continue // If less than two clients are waiting, wait for more
		}

		var client1, client2 *Client

		if len(waitingClients) == 2 {
			// If exactly two waiting clients, just pair them
			client1 = waitingClients[0]
			client2 = waitingClients[1]
		} else {
			// If more than two, pick two random clients

			source := rand.NewSource(time.Now().UnixNano())
			rng := rand.New(source)

			index1 := rng.Intn(len(waitingClients))
			index2 := rng.Intn(len(waitingClients))

			// Ensure index1 and index2 are not the same
			for index1 == index2 {
				index2 = rng.Intn(len(waitingClients))
			}

			client1 = waitingClients[index1]
			client2 = waitingClients[index2]
		}

		// Set both clients' IsWaiting to false
		client1.IsWaiting = false
		client2.IsWaiting = false

		// Notify first clients about the pairing (you can customize this message)
		go client1.sendConnectionRequest(client2.ID, types.ActionOfferReq)
		// go client2.sendConnectionRequest(client1.ID, types.ActionAnswerReq)
	}
}

// SendUserMessage sends a success response with the client's user data.
func (client *Client) sendConnectionRequest(remoteUser, action string) {
	mr := &MessageResponse{
		Action:  action,
		Message: responses.NewSuccessResponse(remoteUser),
	}

	msg, err := json.Marshal(mr)
	if err != nil {
		log.Printf("Failed to marshal user data: %s", err)

		// Aborting context with error response
		client.hub.ctx.AbortWithStatusJSON(http.StatusBadRequest, responses.NewErrorResponse(http.StatusBadRequest, err.Error(), nil))
		return
	}

	client.send <- msg
	// Create and send the ClientMessage
	// cm := &ClientMessage{
	// 	Client:  client,
	// 	Message: msg,
	// }
	// client.hub.Broadcast <- cm
}

// BroadcastUserCount sends the current number of connected clients to all clients.
func (hub *Hub) BroadcastUserCount() {
	hub.mu.Lock()
	activeUsers := len(hub.clients)
	hub.mu.Unlock()

	// Acknowledge the client that they are in the waiting state
	mr := &MessageResponse{
		Action:  types.ActionActiveUsers,
		Message: responses.NewSuccessResponse(activeUsers),
	}
	msg, err := json.Marshal(mr)
	if err != nil {
		log.Printf("Failed to marshal start chat ack: %s", err)
		return
	}

	// Send the message to all connected clients
	for client := range hub.clients {
		client.send <- msg
	}
}

// Close hanging clients channels
func (h *Hub) cleanClients() {

	loopTime := 5 * time.Second
	go func() {
		for range time.Tick(loopTime) {
			// log.Printf("[%s] Total connected clients: %d", time.Now(), len(h.clients))
			for client, createdAt := range h.clients {
				td := time.Since(createdAt).Seconds()
				// Closing inactive clients stored since more than 60 secs
				if td > 60 {

					mr := &MessageResponse{
						Action:  types.ActionConnected,
						Message: responses.NewSuccessResponse(false),
					}

					msg, err := json.Marshal(mr)
					if err != nil {
						log.Printf("Failed to marshal user data: %s", err)

						client.hub.ctx.AbortWithStatusJSON(http.StatusBadRequest, responses.NewErrorResponse(http.StatusBadRequest, err.Error(), nil))
						return
					}

					client.send <- msg
					// log.Printf("[%s] Total clients: %d - Cleaning client %s inactive since %f secs", time.Now(), len(h.clients), client.ID, td)
					delete(h.clients, client)
					close(client.send)
				}
			}
		}
	}()
}
