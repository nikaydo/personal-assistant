package chatcommand

// Commands
const (
	CallAgent     = "/agent"
	CallWebSearch = "/web"
)

// Additional prompts
const (
	ToolPrompt = `
	\nTool policy: If the user request requires actions or tool usage (files, commands, external actions), 
	you must call the agent_mode function with the original user request in the question field. 
	Built-in web search is always available, so use it when the task needs current or public online information.
	Otherwise answer directly without tool calls.`
	WebSearchPrompt = `
	\nFreshness policy: Built-in web search is always available.
	Use current web information when the answer depends on recent or public online data, and say so if the search results are incomplete.`
	ReqNoTool = `
	\nThe user can use commands to clarify their actions. If the request matches the command description, 
	ask the user to perform the request using the command: 
		/agent - allows llm to perform actions on the PC, work with the API, and use built-in web search.
		/web - emphasizes using fresh web information for the current request.`
)
