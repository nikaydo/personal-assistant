package services

import (
	"github.com/nikaydo/personal-assistant/internal/services/command"
	"github.com/nikaydo/personal-assistant/internal/services/wikipedia"
)

// wiki
var SetgowikiWiki = wikipedia.Setgowiki
var SearchWiki = wikipedia.Search
var GeoSearchWiki = wikipedia.GeoSearch
var GetBacklinksWiki = wikipedia.GetBacklinks
var SummaryWiki = wikipedia.Summary

// command
var NewCommandService = command.NewService

type CommandList = command.CommandList
