package workers

type Update struct {
	Type         string        `json:"type"`
	Subtype      string        `json:"subtype"`
	IsError      bool          `json:"is_error"`
	DurationMs   int           `json:"duration_ms"`
	DurationApi  int           `json:"duration_api_ms"`
	NumTurns     int           `json:"num_turns"`
	Result       string        `json:"result"`
	Message      UpdateMessage `json:"message"`
	StopReason   string        `json:"stop_reason"`
	StopSequence int           `json:"stop_sequence"`
	Usage        struct {
		InputTokens              int `json:"input_tokens"`
		CacheCreationInputTokens int `json:"cache_creation_input_tokens"`
		CacheReadInputTokens     int `json:"cache_read_input_tokens"`
		CacheCreation            struct {
			Ephemeral5mInputTokens int `json:"ephemeral_5m_input_tokens"`
			Ephemeral1hInputTokens int `json:"ephemeral_1h_input_tokens"`
		} `json:"cache_creation"`
		OutputTokens int    `json:"output_tokens"`
		ServiceTier  string `json:"service_tier"`
	} `json:"usage"`
	UUID      string `json:"uuid"`
	SessionID string `json:"session_id"`
}

type UpdateMessage struct {
	ID              string                 `json:"id"`
	Type            string                 `json:"type"`
	Role            string                 `json:"role"`
	Model           string                 `json:"model"`
	Content         []UpdateMessageContent `json:"content"`
	ParentToolUseID string                 `json:"parent_tool_use_id"`
}

type UpdateMessageContent struct {
	Type  string                 `json:"type"`
	Text  string                 `json:"text"`
	Name  string                 `json:"name"`
	Input map[string]interface{} `json:"input"`
}
