package websocket

// Kutish xonasi real-time xabar turlari.
const (
	TypeWaitingRoomRequest  MessageType = "waiting_room.request"  // mentorga: yangi so'rov keldi
	TypeWaitingRoomAdmitted MessageType = "waiting_room.admitted" // guestga: qabul qilindingiz (+ token)
	TypeWaitingRoomRejected MessageType = "waiting_room.rejected" // guestga: rad etildi
)

// NewWaitingRoomRequestMsg — mentorga yuboriladigan "yangi so'rov" xabari.
func NewWaitingRoomRequestMsg(payload any) Message {
	return newMsg(TypeWaitingRoomRequest, "", payload)
}

// NewWaitingRoomAdmittedMsg — guestga yuboriladigan "qabul qilindi" xabari (payload: RoomToken).
func NewWaitingRoomAdmittedMsg(payload any) Message {
	return newMsg(TypeWaitingRoomAdmitted, "", payload)
}

// NewWaitingRoomRejectedMsg — guestga yuboriladigan "rad etildi" xabari.
func NewWaitingRoomRejectedMsg(payload any) Message {
	return newMsg(TypeWaitingRoomRejected, "", payload)
}
