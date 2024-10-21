package requests

import "github.com/google/uuid"

type (
	HookSetupRequest struct {
		ID   uuid.UUID `json:"id" format:"uuid" example:"bb296731-3d58-4c4a-8227-315bdc2bf3ff"`
		Type string    `json:"type" example:"bot"`
		Url  string    `json:"url" example:"https://example.org/receive/connect/push/"`

		BotScenarioPoint *[]BotScenarioPoint `json:"bot_scenario_point"`
	}

	BotScenarioPoint struct {
		Childs      *[]BotScenarioPoint `json:"childs" binding:"omitempty,dive"`
		Data        string              `json:"data" binding:"omitempty,max=128"`
		Description string              `json:"description" binding:"omitempty,max=128"`
		Text        string              `json:"text" binding:"required,max=128"`
	}
)
