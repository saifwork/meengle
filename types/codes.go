// types/constants.go
package types

const (
	ErrorGeneric                     = 400
	DatabaseError                    = 400
	ErrorNotFound                    = "Not Found"
	InvalidRequest                   = "the request is not valid"
	DatabaseErrorNotConnectedMessage = "database not connected"

	ActionPing = "ping"
	ActionPong = "pong"

	ActionConnected    = "connected"
	ActionDisConnected = "dis_connected"

	ActionOfferReq = "offer_req"
	ActionOfferRes = "offer_res"

	ActionAnswerReq = "answer_req"
	ActionAnswerRes = "answer_res"
	ActionAnswerRec = "answer_rec"

	ActionIceCandidateRes = "ice_candidate_res"
	ActionIceCandidateRec = "ice_candidate_rec"

	ActionStartChatReq = "start_chat_req"
	ActionStartChatAck = "start_chat_ack"

	ActionActiveUsers = "active_users"
)
