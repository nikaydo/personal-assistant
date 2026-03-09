package tools

type Agent struct {
	Steps int

	SystemPrompt string
	

}

func (a *Agent) Run() {
	for range a.Steps {
		//собрать контекст
		//вызвать llm
		//вызвать функцию
		//повторить
	}
}

func (a *Agent) CollectContext() {

}
