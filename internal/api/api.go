package api

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/nikaydo/personal-assistant/internal/ai"
	"github.com/nikaydo/personal-assistant/internal/config"
	"github.com/nikaydo/personal-assistant/internal/database"
	"github.com/nikaydo/personal-assistant/internal/logg"
	"github.com/nikaydo/personal-assistant/internal/services"
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
	apiLog := log.WithModule("API")
	serviceLog := log.WithModule("SERVICE")
	dbLog := log.WithModule("DB")
	aiLog := log.WithModule("AI")

	apiLog.Info("Initializing integrations")
	jira, err := services.NewJira(config.JiraEmail, config.JiraApiKey, config.JiraPersonalUrl, serviceLog)
	if err != nil {
		return API{}, err
	}

	dbLog.Info("Initializing database client")
	db, err := database.InitDB(config.DatabaseApiKey, &config)
	if err != nil {
		return API{}, err
	}

	if err := db.WaitIndexReady(config.IndexName, dbLog); err != nil {
		return API{}, err
	}

	dbLog.Info("Database index ready")
	apiLog.Info("Core services initialized")

	a := API{
		Addr: &Addr{
			Host: config.ApiHost,
			Port: strconv.Itoa(config.ApiPort),
		},
		Router: chi.NewRouter(),
		Ai: &ai.Ai{
			Memory: &ai.Memory{DBase: db, Cfg: &config, Logger: aiLog},
			Config: config,
			Jira:   jira,
			Logger: aiLog,
		},
	}

	if err := a.fillAiToolParam(); err != nil {
		return API{}, err
	}
	return a, nil
}

func (api *API) fillAiToolParam() error {
	api.Ai.Logger.Info("Loading Jira user profile for tool configuration")
	if err := api.Ai.Jira.GetUserInfo(api.Ai.Config.JiraPersonalUrl); err != nil {
		api.Ai.Logger.Error("fillAiToolParam: get Jira user info failed:", err)
		return err
	}
	api.Ai.ToolConf = &ai.ToolConf{
		AccountId: api.Ai.Jira.MainUser.AccountID,
	}
	api.Ai.Logger.Info("Tool configuration loaded", "jira_account_id", api.Ai.ToolConf.AccountId)
	return nil
}

func (api *API) Start() error {
	return http.ListenAndServe(api.Addr.Host+":"+api.Addr.Port, api.Router)
}

func (api *API) SetupRoutes() {
	api.Router.Post("/chat", api.chat)
	api.Router.Post("/memory", api.GetMemory)
	api.Router.Post("/msg", api.GetMessage)

}
