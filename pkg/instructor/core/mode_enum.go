package core

type Mode = string

const (
	ModeToolCall       Mode = "tool_call_mode"
	ModeToolCallStrict Mode = "tool_call_strict_mode"
	ModeJSON           Mode = "json_mode"
	ModeJSONStrict     Mode = "json_strict_mode"
	ModeJSONSchema     Mode = "json_schema_mode"
	ModeMarkdownJSON   Mode = "markdown_json_mode"
	ModeDefault        Mode = ModeJSONSchema
)
