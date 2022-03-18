package main

import (
	"fmt"
	"net/http"
	"sync"

	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/mattermost/mattermost-server/v6/plugin"
)

// LivePlugin implements the interface expected by the Mattermost server to communicate between the server and plugin processes.
type LivePlugin struct {
	plugin.MattermostPlugin

	// configurationLock synchronizes access to the configuration.
	configurationLock sync.RWMutex

	// configuration is the active plugin configuration. Consult getConfiguration and
	// setConfiguration for usage.
	configuration *configuration
	bot           *model.Bot
}

func (kit *LivePlugin) OnActivate() error {
	kit.API.LogInfo("Activating...")
	config := kit.getConfiguration()
	// validate config
	kit.configuration = config

	command, err := kit.compileSlashCommand()
	if err != nil {
		return err
	}

	if err = kit.API.RegisterCommand(command); err != nil {
		return err
	}

	liveBot := &model.Bot{
		Username:    "livekit",
		DisplayName: "Live Bot",
		Description: "A bot account created by the LiveKit plugin",
	}

	bot, ae := kit.API.CreateBot(liveBot)
	if ae == nil {
		kit.bot = bot
	}

	return nil
}

func (kit *LivePlugin) OnDeactivate() error {
	return nil
}

func (kit *LivePlugin) compileSlashCommand() (*model.Command, error) {
	acData := model.NewAutocompleteData("call", "[command]", "Start a LiveKit meeting in current channel. Other available commands: start, help, settings")
	start := model.NewAutocompleteData("start", "[topic]", "Start a new meeting in the current channel")
	start.AddTextArgument("(optional) The topic of the new meeting", "[topic]", "")
	acData.AddCommand(start)

	command := &model.Command{
		Trigger:          "call",
		AutoComplete:     true,
		AutoCompleteDesc: "Start a Jitsi meeting in current channel. Other available commands: start, help, settings",
		AutoCompleteHint: "[command]",
		AutocompleteData: acData,
		// AutocompleteIconData: iconData,
	}
	return command, nil
}

// ServeHTTP demonstrates a plugin that handles HTTP requests by greeting the world.
func (kit *LivePlugin) ServeHTTP(c *plugin.Context, w http.ResponseWriter, r *http.Request) {
	userID := r.Header.Get("Mattermost-User-ID")
	if userID == "" {
		http.Error(w, "Not authorized", http.StatusUnauthorized)
		return
	}
	switch r.URL.Path {
	case "/webhook":
		fmt.Fprint(w, "Hello, world! This hook is not implemented yet.")
	case "/token":
		fmt.Fprint(w, "Hello, world! Token feature is not implemented yet.")
		// kit.API.LogInfo()
	default:
		http.NotFound(w, r)
	}
}

// See https://developers.mattermost.com/extend/plugins/server/reference/
