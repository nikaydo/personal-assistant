package main

import (
	"fmt"
	"log"

	_ "github.com/ctreminiom/go-atlassian/v2/jira/v3"
	"github.com/nikaydo/jira-filler/internal/api"
	"github.com/nikaydo/jira-filler/internal/config"
	"github.com/nikaydo/jira-filler/internal/logg"
)

func main() {
	config, err := config.ConfigRead("./settings.json")
	if err != nil {
		log.Fatal("Failed to read config:", err)
	}
	l := logg.InitLogger()
	l.Info("Config readed")
	api, err := api.SetupApi(*config, l)
	if err != nil {
		log.Fatal("Failed to setup api:", err)
	}
	l.Info(fmt.Sprintf("Server configurated on addr %s:%d", config.ApiHost, config.ApiPort))
	api.Ai.GetModelData(*config, l)
	l.Info(fmt.Sprintf("Context length: %d", api.Ai.Context.ContextLeghtMax))
	l.Info(fmt.Sprintf("Main model is: %s. Spare models: %v", api.Ai.Model[0], api.Ai.Model[1:]))
	api.SetupRoutes()
	l.Info("Routes setup")
	l.Info(fmt.Sprintf("Server started on addr %s:%d", config.ApiHost, config.ApiPort))
	l.Info("Ready!!!")
	err = api.Start()
	if err != nil {
		panic(err)
	}
}
