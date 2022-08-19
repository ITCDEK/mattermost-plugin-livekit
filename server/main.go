package main

import (
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	kitSDK "github.com/livekit/server-sdk-go"
	pluginSDK "github.com/mattermost/mattermost-plugin-api"
	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/mattermost/mattermost-server/v6/plugin"
	"github.com/pkg/errors"
)

// LiveKitPlugin implements the interface expected by the Mattermost server to communicate between the server and plugin processes.
type LiveKitPlugin struct {
	plugin.MattermostPlugin
	botUserID         string
	bundlePath        string
	configurationLock sync.RWMutex
	configuration     *configuration
	master            *kitSDK.RoomServiceClient
	sdk               *pluginSDK.Client
}

func main() {
	plugin.ClientMain(&LiveKitPlugin{})
}

func (lkp *LiveKitPlugin) OnActivate() error {
	lkp.API.LogInfo("Activating LiveKit integration...")
	configuration := lkp.getConfiguration()
	// validate configuration here
	lkp.configuration = configuration
	lkp.sdk = pluginSDK.NewClient(lkp.API, lkp.Driver)

	//Bot
	liveBot := &model.Bot{
		Username:    "livekit.bot",
		DisplayName: "Broadcasting",
		Description: "Created by the LiveKit plugin",
	}

	lkp.API.LogInfo("Ensuring bot", "name", liveBot.Username)
	botUserID, err := lkp.sdk.Bot.EnsureBot(liveBot)
	if err == nil {
		lkp.botUserID = botUserID
	} else {
		return errors.Wrap(err, "failed to ensure bot account")
	}

	lkp.API.LogInfo("Setting bot profile image")
	bundlePath, err := lkp.API.GetBundlePath()
	if err == nil {
		lkp.bundlePath = bundlePath
		profileImage, err := ioutil.ReadFile(filepath.Join(bundlePath, "assets", "bot-icon.png")) //SVG format is not supported
		if err == nil {
			if appErr := lkp.API.SetProfileImage(botUserID, profileImage); appErr != nil {
				return errors.Wrap(appErr, "couldn't set profile image")
			}
		} else {
			return errors.Wrap(err, "couldn't read profile image")
		}
	} else {
		return errors.Wrap(err, "couldn't get bundle path")
	}

	lkp.API.LogInfo("Compiling slash command")
	command, err := lkp.compileSlashCommand()
	if err == nil {
		lkp.API.LogInfo("Registering slash command")
		err = lkp.API.RegisterCommand(command)
		if err == nil {
			lkp.API.LogInfo("slash command registered")
			serverURL := fmt.Sprintf("https://%s:%d", lkp.configuration.Host, lkp.configuration.Port)
			lkp.master = kitSDK.NewRoomServiceClient(serverURL, lkp.configuration.ApiKey, lkp.configuration.ApiValue)
			lkp.API.LogInfo("LiveKit integration activated")
			return nil
		}
	}

	return errors.Wrap(err, "couldn't compile slash-command")
}

func (lkp *LiveKitPlugin) OnDeactivate() error {
	return nil
}

func (lkp *LiveKitPlugin) compileSlashCommand() (*model.Command, error) {
	// https://developers.mattermost.com/integrate/admin-guide/admin-slash-commands/
	acData := model.NewAutocompleteData("liveroom", "[topic]", "Start a LiveKit meeting in current channel. Topic should be provided in double quotes.")
	// start := model.NewAutocompleteData("start", "[topic]", "Start a new meeting in the current channel")
	// start.AddTextArgument("(optional) The topic of the new meeting", "[topic]", "")
	// acData.AddCommand(start)

	command := &model.Command{
		Trigger:          "liveroom",
		AutoComplete:     true,
		AutoCompleteDesc: "Start a LiveKit meeting in current channel. Other available commands: start, help, settings",
		AutoCompleteHint: "[command]",
		AutocompleteData: acData,
		// AutocompleteIconData: iconData,
	}
	return command, nil
}

func (lkp *LiveKitPlugin) ExecuteCommand(c *plugin.Context, args *model.CommandArgs) (*model.CommandResponse, *model.AppError) {
	response := &model.CommandResponse{ResponseType: model.CommandResponseTypeEphemeral}
	// fields := strings.Fields(args.Command)

	splitted := strings.Split(args.Command, "\"")
	if len(splitted) > 1 && splitted[0] == "/liveroom " {
		maxParticipants := uint32(0)
		topic := splitted[1]
		n := strings.ReplaceAll(splitted[2], " ", "")
		if len(splitted) == 3 && len(n) > 0 {
			integer, err := strconv.Atoi(n)
			if err != nil {
				response.Text = err.Error()
				return response, nil
			}
			maxParticipants = uint32(integer)
		}
		lkp.API.LogInfo("creating rom", "topic", topic, "n", maxParticipants)
		appErr := lkp.createPost(args.ChannelId, args.UserId, topic, maxParticipants)
		if appErr == nil {
			response.Text = fmt.Sprintf("Creating room with topic = %s; maxParticipants = %d", topic, maxParticipants)
		} else {
			response.Text = fmt.Sprintf("Room creation failed: %s", appErr.DetailedError)
		}
	}
	return response, nil
}
