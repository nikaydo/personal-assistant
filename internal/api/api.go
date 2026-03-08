package api

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/nikaydo/personal-assistant/internal/ai"
	"github.com/nikaydo/personal-assistant/internal/config"
	"github.com/nikaydo/personal-assistant/internal/database"
	"github.com/nikaydo/personal-assistant/internal/logg"
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
	dbLog := log.WithModule("DB")
	aiLog := log.WithModule("AI")

	apiLog.Info("Initializing integrations")

	dbLog.Info("Initializing database client")
	db, err := database.InitDB(&config)
	if err != nil {
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
		Ai:     ai.Init(config, aiLog, db),
	}

	return a, nil
}

func (api *API) Start() error {
	return http.ListenAndServe(api.Addr.Host+":"+api.Addr.Port, api.Router)
}

func (api *API) SetupRoutes() {
	api.Router.Post("/chat", api.chat)
	api.Router.Post("/memory", api.GetMemory)
	api.Router.Post("/msg", api.GetMessage)
}
