package api

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/nikaydo/jira-filler/internal/ai"
	"github.com/nikaydo/jira-filler/internal/config"
	"github.com/nikaydo/jira-filler/internal/jira"
	"github.com/nikaydo/jira-filler/internal/logg"
)

type API struct {
	Addr   *Addr
	Router *chi.Mux

	Ai *ai.Ai
}

type Addr struct {
	Host string
	Port string
}

func SetupApi(config config.Config, log *logg.Logger) (API, error) {
	jira, err := jira.NewJira(config.JiraEmail, config.JiraApiKey, config.JiraPersonalUrl, log)
	if err != nil {
		return API{}, err
	}

	a := API{
		Addr: &Addr{
			Host: config.ApiHost,
			Port: strconv.Itoa(config.ApiPort),
		},
		Router: chi.NewRouter(),
		Ai: &ai.Ai{
			ApiKey: config.ApiKeyOpenrouter,
			Model:  config.ModelOpenRouter,
			Url:    config.ApiUrlOpenrouter,
			Memory: &ai.Memory{},
			Config: config,
			Jira:   jira,
			Logger: log,
		},
	}

	if err := a.fillAiToolParam(); err != nil {
		return API{}, err
	}
	return a, nil
}

func (api *API) fillAiToolParam() error {
	if err := api.Ai.Jira.GetUserInfo(); err != nil {
		return err
	}
	api.Ai.ToolConf = &ai.ToolConf{
		AccountId: api.Ai.Jira.MainUser.AccountID,
	}
	return nil
}

func (api *API) Start() error {
	return http.ListenAndServe(api.Addr.Host+":"+api.Addr.Port, api.Router)
}

func (api *API) SetupRoutes() {
	api.Router.Get("/chat", api.chat)
	api.Router.Get("/memory", api.GetMemory)
}
