package api

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

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

	server *http.Server
}

type Addr struct {
	Host string
	Port string
}

func SetupApi(config config.Config, log *logg.Logger) (API, error) {

	log.WithModule("API").Info("Initializing integrations")

	log.WithModule("DB").Info("Initializing database client")
	db, err := database.InitDB(&config)
	if err != nil {
		return API{}, err
	}

	log.WithModule("DB").Info("Database index ready")
	log.WithModule("API").Info("Core services initialized")

	a := API{
		Addr: &Addr{
			Host: config.ApiHost,
			Port: strconv.Itoa(config.ApiPort),
		},
		Router: chi.NewRouter(),
		Ai:     ai.Init(config, log.WithModule("AI"), db),
	}

	return a, nil
}

func (api *API) Start() error {
	api.server = &http.Server{
		Addr:              api.Addr.Host + ":" + api.Addr.Port,
		Handler:           api.Router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
	return api.server.ListenAndServe()
}

func (api *API) Shutdown(ctx context.Context) error {
	var errs []error

	if api.Ai != nil && api.Ai.Queue != nil {
		api.Ai.Queue.Stop()
	}
	if api.Ai != nil && api.Ai.Memory != nil {
		if err := api.Ai.Memory.FlushState(); err != nil {
			errs = append(errs, err)
		}
	}
	if api.Ai != nil && api.Ai.Memory != nil && api.Ai.Memory.DBase != nil {
		if err := api.Ai.Memory.DBase.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if api.server != nil {
		if err := api.server.Shutdown(ctx); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errs = append(errs, err)
		}
	}

	return errors.Join(errs...)
}

func (api *API) SetupRoutes() {
	api.Router.Post("/chat", api.chat)
	api.Router.Post("/memory", api.GetMemory)
	api.Router.Post("/msg", api.GetMessage)
}
