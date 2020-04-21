package requests

import "github.com/google/uuid"

type (
	HookSetupRequest struct {
		Id   uuid.UUID `json:"id" format:"uuid" example:"bb296731-3d58-4c4a-8227-315bdc2bf3ff"`
		Type string    `json:"type" example:"bot"`
		Url  string    `json:"url" example:"https://example.org/receive/connect/push/"`
	}
)
