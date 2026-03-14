package chatcommand

// Commands
const (
	CallAgent = "/agent"
)

// Additional prompts
const (
	ToolPrompt = `
	\nTool policy: If the user request requires actions or tool usage (files, commands, external actions), 
	you must call the agent_mode function with the original user request in the question field. 
	Otherwise answer directly without tool calls.`
	ReqNoTool = `
	\nThe user can use commands to clarify their actions. If the request matches the command description, 
	ask the user to perform the request using the command: 
		/agent - allows llm to perform actions on the PC or use the Internet to search for information or work with the API.`
)
