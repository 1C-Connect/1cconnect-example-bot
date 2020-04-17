package messages

import "github.com/google/uuid"

type MessageType int

const(
	MESSAGE_TEXT MessageType = 1
)

type (


	Message struct {
		LineId uuid.UUID `json:"line_id" binding:"required" example:"4e48509f-6366-4897-9544-46f006e47074"`
		UserId uuid.UUID `json:"user_id" binding:"required" example:"4e48509f-6366-4897-9544-46f006e47074"`

		MessageID uuid.UUID `json:"message_id" binding:"required" example:"4e48509f-6366-4897-9544-46f006e47074"`
		MessageType MessageType `json:"message_type" binding:"required" example:"1"`
		MessageAuthor uuid.UUID `json:"author_id" binding:"required" example:"4e48509f-6366-4897-9544-46f006e47074"`
		MessageTime string `json:"message_time" binding:"required" example:"1"`
		Text string `json:"text" example:"Привет"`
	}
)
