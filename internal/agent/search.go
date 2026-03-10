package agent

import "github.com/nikaydo/personal-assistant/internal/services"

type Search struct {
	Source string
	Args   SearchArgs
}

type SearchArgs struct {
	Query string
	Limit int
}

func (a *Agent) Search(s Search) {
	switch s.Source {
	case "wikipedia":
		services.SearchWiki(s.Args.Query, s.Args.Limit, false)
	}
}
