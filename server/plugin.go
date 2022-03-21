package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/livekit/protocol/auth"
	"github.com/livekit/protocol/livekit"
	kitSDK "github.com/livekit/server-sdk-go"
	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/mattermost/mattermost-server/v6/plugin"
)

// References
// https://github.com/mattermost/mattermost-plugin-jitsi
// https://github.com/mattermost/mattermost-plugin-zoom
// https://github.com/streamer45/mattermost-plugin-voice
// https://github.com/niklabh/mattermost-plugin-webrtc-video
// https://github.com/Kopano-dev/mattermost-plugin-kopanowebmeetings
// https://github.com/blindsidenetworks/mattermost-plugin-bigbluebutton
// https://developers.mattermost.com/integrate/plugins/server/reference/

// LivePlugin implements the interface expected by the Mattermost server to communicate between the server and plugin processes.
type LivePlugin struct {
	plugin.MattermostPlugin

	// configurationLock synchronizes access to the configuration.
	configurationLock sync.RWMutex

	// configuration is the active plugin configuration. Consult getConfiguration and
	// setConfiguration for usage.
	configuration *configuration
	bot           *model.Bot
	master        *kitSDK.RoomServiceClient
}

func (kp *LivePlugin) OnActivate() error {
	kp.API.LogInfo("Activating...")
	config := kp.getConfiguration()
	// validate config
	kp.configuration = config

	command, err := kp.compileSlashCommand()
	if err != nil {
		return err
	}

	if err = kp.API.RegisterCommand(command); err != nil {
		return err
	}

	liveBot := &model.Bot{
		Username:    "livekit",
		DisplayName: "Live Bot",
		Description: "A bot account created by the LiveKit plugin",
	}

	bot, ae := kp.API.CreateBot(liveBot)
	if ae == nil {
		kp.bot = bot
	}

	server := kp.configuration.Servers[0]
	kp.master = kitSDK.NewRoomServiceClient(server.Host, server.ApiKey, server.ApiSecret)

	return nil
}

func (kp *LivePlugin) OnDeactivate() error {
	return nil
}

func getJoinToken(apiKey, apiSecret, room, identity string) (string, error) {
	at := auth.NewAccessToken(apiKey, apiSecret)
	grant := &auth.VideoGrant{
		RoomJoin: true,
		Room:     room,
	}
	at.AddGrant(grant).
		SetIdentity(identity).
		SetValidFor(time.Hour)

	return at.ToJWT()
}

func (kp *LivePlugin) newRoom(roomName, channelID, rootID, topic string) error {
	room, err := kp.master.CreateRoom(context.Background(), &livekit.CreateRoomRequest{Name: roomName})
	kp.API.LogInfo("room created at", room.CreationTime)
	post := &model.Post{
		UserId:    kp.bot.UserId,
		ChannelId: channelID,
		RootId:    rootID,
		Message:   "I have started a meeting",
		Type:      "custom_zoom",
		Props: map[string]interface{}{
			"room_capacity": room.MaxParticipants,
			"room_name":     room.Name,
			// "meeting_link":             meetingURL,
			// "meeting_status":           zoom.WebhookStatusStarted,
			// "meeting_personal":         false,
			// "meeting_topic":            topic,
			// "meeting_creator_username": creator.Username,
			// "meeting_provider":         zoomProviderName,
			// "attachments":              []*model.SlackAttachment{&slackAttachment},
		},
	}

	newRoomPost, appErr := kp.API.CreatePost(post)
	kp.API.LogInfo("room post created with ID =", newRoomPost.Id)
	if appErr != nil {
		return appErr
	}
	return err
}

func (kp *LivePlugin) compileSlashCommand() (*model.Command, error) {
	acData := model.NewAutocompleteData("call", "[command]", "Start a LiveKit meeting in current channel. Other available commands: start, help, settings")
	start := model.NewAutocompleteData("start", "[topic]", "Start a new meeting in the current channel")
	start.AddTextArgument("(optional) The topic of the new meeting", "[topic]", "")
	acData.AddCommand(start)

	command := &model.Command{
		Trigger:          "call",
		AutoComplete:     true,
		AutoCompleteDesc: "Start a LiveKit meeting in current channel. Other available commands: start, help, settings",
		AutoCompleteHint: "[command]",
		AutocompleteData: acData,
		// AutocompleteIconData: iconData,
	}
	return command, nil
}

// ServeHTTP demonstrates a plugin that handles HTTP requests by greeting the world.
func (kp *LivePlugin) ServeHTTP(c *plugin.Context, w http.ResponseWriter, r *http.Request) {
	type roomRequest struct {
		ChannelID string `json:"channel_id"`
		Personal  bool   `json:"personal"`
		Topic     string `json:"topic"`
		MeetingID int    `json:"meeting_id"`
	}
	userID := r.Header.Get("Mattermost-User-ID")
	if userID == "" {
		http.Error(w, "Not authorized", http.StatusUnauthorized)
		return
	}
	var rr roomRequest
	err := json.NewDecoder(r.Body).Decode(&rr)
	if err == nil {
		switch r.URL.Path {
		case "/webhook":
			fmt.Fprint(w, "Hello, world! This hook is not implemented yet.")
		case "/room":
			fmt.Fprint(w, "Hello, world! The new room feature is not implemented yet.")
		case "/token":
			fmt.Fprint(w, "Hello, world! Token feature is not implemented yet.")
			// kit.API.LogInfo()
		default:
			http.NotFound(w, r)
		}
	}
	http.Error(w, err.Error(), http.StatusBadRequest)
}

// See https://developers.mattermost.com/extend/plugins/server/reference/
