package socket

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	log "log"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/saifwork/socket-service/responses"
	"github.com/saifwork/socket-service/types"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 15 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 30 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 8192
)

var (
// newline = []byte{'\n'}
// space   = []byte{' '}
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  8192,
	WriteBufferSize: 8192,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type User struct {
	ID        string    `json:"uId"`       // Unique identifier for the user (could be a UUID)
	Addr      string    `json:"addr"`      // Network address of the user (IP or WebSocket addr)
	EnterAt   time.Time `json:"enterAt"`   // Timestamp when the user entered the waiting state
	IsWaiting bool      `json:"isWaiting"` // Whether the user is in the waiting state for matchmaking
}

type ClientMessage struct {
	Client  *Client
	Message []byte
}

type MessageRequest struct {
	Action string          `json:"action"`
	Data   json.RawMessage `json:"data"`
}

type Offer struct {
	ID       string `json:"uId"`  // UID or Peer ID
	OfferSDP string `json:"sdp"`  // SDP sent by the frontend
	Type     string `json:"type"` // Type should be "offer"
}

type Answer struct {
	ID        string `json:"uId"`    // UID or Peer ID
	AnswerSDP string `json:"answer"` // SDP sent by the frontend
	Type      string `json:"type"`   // Type should be "offer"
}

type IceCandidate struct {
	ID            string `json:"uId"`
	Candidate     string `json:"candidate"`
	SdpMid        string `json:"sdpMid"`
	SdpMLineIndex int    `json:"sdpMLineIndex"`
}

type ClientDisconnect struct {
	ID string `json:"uId"`
}

type ClientHangUp struct {
	ID string `json:"uId"`
}

type MessageResponse struct {
	Action  string      `json:"action"`
	Message interface{} `json:"message"`
}

// Client is a middleman between the websocket connection and the hub.
type Client struct {
	// The hub handling the messages logic
	hub *Hub

	// The websocket connection.
	conn *websocket.Conn

	// Buffered channel of outbound messages.
	send chan []byte

	User
}

// ReadPump pumps messages from the websocket connection to the hub.
//
// The application runs ReadPump in a per-connection goroutine. The application
// ensures that there is at most one reader on a connection by executing
// reads from this goroutine.
func (c *Client) ReadPump() {
	log.Println("INFO: inside ReadPump()")

	defer func() {
		if c == nil || c.hub == nil {
			return
		}
		c.hub.unregister <- c
		if c.conn != nil {
			_ = c.conn.Close()
		}
	}()
	c.conn.SetReadLimit(maxMessageSize)
	_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))

	c.conn.SetPongHandler(func(string) error {
		_ = c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Println(err)
			}
			break
		}

		// Parse the message into the generic MessageRequest struct
		var msgReq MessageRequest
		if err := json.Unmarshal(message, &msgReq); err != nil {
			log.Println("Error parsing message:", err)
			continue
		}

		log.Print("req")
		log.Println(msgReq)

		switch msgReq.Action {

		case types.ActionPing:

			c.hub.clients[c] = time.Now()

			mr := &MessageResponse{
				Action:  types.ActionPong,
				Message: responses.NewSuccessResponse("pong"),
			}
			msg, err := json.Marshal(mr)
			if err != nil {
				log.Printf("Failed to marshal start chat ack: %s", err)
				continue
			}
			c.send <- msg

		case types.ActionStartChatReq:
			// Set the client as waiting
			c.IsWaiting = true
			log.Printf("Client %s is now waiting for a match", c.ID)

			// Acknowledge the client that they are in the waiting state
			mr := &MessageResponse{
				Action:  types.ActionStartChatAck,
				Message: responses.NewSuccessResponse("waiting for a match"),
			}
			msg, err := json.Marshal(mr)
			if err != nil {
				log.Printf("Failed to marshal start chat ack: %s", err)
				continue
			}
			c.send <- msg

		case types.ActionOfferRes:

			print("inside offer_res")

			var offer Offer
			if err := json.Unmarshal(msgReq.Data, &offer); err != nil {

				log.Println("inside offer_res err")
				log.Println("Error parsing offer:", err)
				return
			}
			log.Printf("Received offer: %s", offer)

			remoteId := offer.ID
			offer.ID = c.ID
			// Call the refactored function
			c.handleMessageResponse(types.ActionAnswerReq, offer, remoteId)

		case types.ActionAnswerRes:
			var answer Answer
			if err := json.Unmarshal(msgReq.Data, &answer); err != nil {
				log.Println("Error parsing answer:", err)
				continue
			}
			log.Printf("Received answer: %s", answer)

			remoteId := answer.ID
			answer.ID = c.ID

			// Call the refactored function
			c.handleMessageResponse(types.ActionAnswerRec, answer, remoteId)

		case types.ActionIceCandidateRes:

			log.Printf("Inside ActionIceCandidateRes")

			var iceCandidate IceCandidate
			if err := json.Unmarshal(msgReq.Data, &iceCandidate); err != nil {
				log.Println("Error parsing ICE candidate:", err)
				continue
			}
			log.Printf("Received ICE candidate: %s", iceCandidate.Candidate)

			remoteId := iceCandidate.ID
			iceCandidate.ID = c.ID

			// Call the refactored function
			c.handleMessageResponse(types.ActionIceCandidateRec, iceCandidate, remoteId)

		case types.ActionDisConnected:

			var clientDisconnect ClientDisconnect
			if err := json.Unmarshal(msgReq.Data, &clientDisconnect); err != nil {
				log.Println("Error parsing clientDisconnect:", err)
				continue
			}

			if clientDisconnect.ID != "" {
				c.handleMessageResponse(types.ActionHangUpRec, nil, clientDisconnect.ID)

				// Caller Peer
				client := c.hub.GetClientByID(clientDisconnect.ID)
				if client != nil {
					client.IsWaiting = true
				}
			}

			c.hub.mu.Lock()

			client := c.hub.GetClientByID(c.ID)
			if client == nil {
				c.hub.mu.Unlock()
				log.Println("Client not found for ID:", c.ID)
				return
			}

			if _, ok := c.hub.clients[client]; ok {
				delete(c.hub.clients, client)
				close(client.send)
			}
			c.hub.mu.Unlock()

			log.Printf("Client Disconnect: %s", c.ID)

		case types.ActionHangUpRes:

			log.Println("inside ActionHangUpRes")

			var clientHangUp ClientHangUp
			if err := json.Unmarshal(msgReq.Data, &clientHangUp); err != nil {
				log.Println("Error parsing Client HangUp :", err)
				continue
			}

			log.Println(clientHangUp.ID)

			c.handleMessageResponse(types.ActionHangUpRec, nil, clientHangUp.ID)

			// Callie Peer
			c.IsWaiting = true

			// Caller Peer
			client := c.hub.GetClientByID(clientHangUp.ID)
			if client != nil {
				client.IsWaiting = true
			}

			log.Printf("Status Updated for: %s - %s", c.ID, client.ID)

		default:
			log.Println("Unknown action:", msgReq.Action)
		}

	}
}

// WritePump pumps messages from the hub to the websocket connection.
//
// A goroutine running WritePump is started for each connection. The
// application ensures that there is at most one writer to a connection by
// executing all writes from this goroutine.
func (c *Client) WritePump() {

	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		_ = c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			_ = c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel.
				messageStr := fmt.Sprintf("[%s] INFO: Client closed the channel - Total clients %d", time.Now(), len(c.hub.clients))
				log.Println(messageStr)
				return
			}

			// Send the first message
			err := c.conn.WriteMessage(websocket.TextMessage, message)
			if err != nil {
				log.Println("Error sending message: ", err)
				return
			}

			// Send remaining messages one by one
			for i := 0; i < len(c.send); i++ {
				nextMessage := <-c.send
				err = c.conn.WriteMessage(websocket.TextMessage, nextMessage)
				if err != nil {
					log.Println("Error sending message: ", err)
					return
				}
			}

		case <-ticker.C:
			if err := c.conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
				return
			}
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// ServeWebsockets handles websocket requests from the peer.
func ServeWebsockets(hub *Hub, w http.ResponseWriter, r *http.Request) {

	// Extract the uId from the query parameters
	uId := r.URL.Query().Get("uId")
	if uId == "" {
		log.Printf("[%s] uId not provided in the request", time.Now())
		hub.ctx.AbortWithStatusJSON(http.StatusBadRequest, responses.NewErrorResponse(http.StatusBadRequest, "uId is required", nil))
		return
	}

	existClient := hub.GetClientByID(uId)
	if existClient != nil {
		log.Printf("[%s] user already connected", time.Now())
		hub.ctx.AbortWithStatusJSON(http.StatusBadRequest, responses.NewErrorResponse(http.StatusBadRequest, "user already connected", nil))
		return
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("[%s] Error on serving the websocket: %s", time.Now(), err.Error())
		if conn != nil {
			_ = conn.Close()
		}

		if hub != nil {

			hub.ctx.AbortWithStatusJSON(http.StatusBadRequest, responses.NewErrorResponse(http.StatusBadRequest, "Bad socket request %s", err.Error()))
		}

		return
	}

	log.Printf("INFO: Req conn upgraded")

	log.Printf("[%s] DEBUG: Creating the client", time.Now())
	client := &Client{hub: hub, conn: conn, send: make(chan []byte, 256)}

	log.Printf("[%s] DEBUG: Generating the user id", time.Now())
	if uId != "" {
		client.ID = uId // Assign uId if it's not empty
	} else {
		client.ID = GenUserId() // Call GenUserId() if uId is empty
	}

	log.Printf("[%s] DEBUG: Registering remote address", time.Now())
	client.Addr = conn.RemoteAddr().String()

	log.Printf("[%s] DEBUG: Registering time", time.Now())
	client.EnterAt = time.Now()

	log.Printf("[%s] DEBUG: Registering isWaiting ", time.Now())
	client.IsWaiting = false

	log.Printf("[%s] DEBUG: Registering the client", time.Now())
	client.hub.register <- client

	// Logging the client info
	log.Printf("[%s] INFO: Client info: %+v", time.Now(), client)

	// Allow collection of memory referenced by the caller by doing all work in
	// new goroutines.

	log.Println("INFO: before calling Read and Write Pump")

	go client.WritePump()
	go client.ReadPump()

	mr := &MessageResponse{
		Action:  types.ActionConnected,
		Message: responses.NewSuccessResponse(true),
	}

	msg, err := json.Marshal(mr)
	if err != nil {
		log.Printf("Failed to marshal user data: %s", err)

		hub.ctx.AbortWithStatusJSON(http.StatusBadRequest, responses.NewErrorResponse(http.StatusBadRequest, err.Error(), nil))
		return
	}

	client.send <- msg
}

func (c *Client) handleMessageResponse(action string, obj interface{}, remoteUid string) {
	// Marshal the object into JSON
	mr := &MessageResponse{
		Action:  action,
		Message: responses.NewSuccessResponse(obj),
	}

	msg, err := json.Marshal(mr)
	if err != nil {
		log.Printf("Failed to marshal user data: %s", err)
		c.hub.ctx.AbortWithStatusJSON(http.StatusBadRequest, responses.NewErrorResponse(http.StatusBadRequest, err.Error(), nil))
		return
	}

	// Find the remote client by UID
	remoteClient := c.hub.GetWaitingClientByUID(remoteUid)

	if remoteClient != nil {
		// Client with the specified UID and isWaiting == true was found
		log.Printf("Found waiting client with UID: %s", remoteClient.ID)

		remoteClient.send <- msg
	} else {
		// Handle the case where no client was found
		log.Println("No waiting client found with the given UID")
	}
}

// GenUserId generate a new custom user id
func GenUserId() string {
	return uuid.NewString()
}
