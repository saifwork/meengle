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

	ActionHangUpRes = "hang_up_res"
	ActionHangUpRec = "hang_up_rec"

	ActionForceHangUpRes = "force_hang_up_res"

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
