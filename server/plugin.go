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

// LiveKitPlugin implements the interface expected by the Mattermost server to communicate between the server and plugin processes.
type LiveKitPlugin struct {
	plugin.MattermostPlugin

	// configurationLock synchronizes access to the configuration.
	configurationLock sync.RWMutex

	// configuration is the active plugin configuration. Consult getConfiguration and
	// setConfiguration for usage.
	configuration *configuration
	bot           *model.Bot
	master        *kitSDK.RoomServiceClient
}

func (lkp *LiveKitPlugin) OnActivate() error {
	lkp.API.LogInfo("Activating...")
	config := lkp.getConfiguration()
	// validate config
	lkp.configuration = config

	command, err := lkp.compileSlashCommand()
	if err != nil {
		return err
	}

	if err = lkp.API.RegisterCommand(command); err != nil {
		return err
	}

	liveBot := &model.Bot{
		Username:    "livekit",
		DisplayName: "Live Bot",
		Description: "A bot account created by the LiveKit plugin",
	}

	bot, ae := lkp.API.CreateBot(liveBot)
	if ae == nil {
		lkp.bot = bot
	}

	server := lkp.configuration.Servers[0]
	lkp.master = kitSDK.NewRoomServiceClient(server.Host, server.ApiKey, server.ApiSecret)
	return nil
}

func (lkp *LiveKitPlugin) OnDeactivate() error {
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

func (lkp *LiveKitPlugin) newRoom(userID, channelID, rootID, topic string) error {
	room, err := lkp.master.CreateRoom(context.Background(), &livekit.CreateRoomRequest{Name: topic, Metadata: channelID, EmptyTimeout: 300})
	lkp.API.LogInfo("room created at", room.CreationTime)
	post := &model.Post{
		UserId:    lkp.bot.UserId,
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

	newRoomPost, appErr := lkp.API.CreatePost(post)
	lkp.API.LogInfo("room post created with ID =", newRoomPost.Id)
	if appErr != nil {
		return appErr
	}
	return err
}

func (lkp *LiveKitPlugin) compileSlashCommand() (*model.Command, error) {
	// https://developers.mattermost.com/integrate/admin-guide/admin-slash-commands/
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
func (lkp *LiveKitPlugin) ServeHTTP(c *plugin.Context, w http.ResponseWriter, r *http.Request) {
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
			if _, err := lkp.API.GetChannelMember(rr.ChannelID, userID); err != nil {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}
			channel, appErr := lkp.API.GetChannel(rr.ChannelID)
			if appErr != nil {
				http.Error(w, "Forbidden", http.StatusForbidden)
				return
			}
			if channel.Type == model.ChannelTypeDirect || channel.Type == model.ChannelTypeGroup {
				// meetingID = generatePersonalMeetingName(user.Username)
				// meetingTopic = p.b.LocalizeWithConfig(l, &i18n.LocalizeConfig{
				// 	DefaultMessage: &i18n.Message{
				// 		ID:    "jitsi.start_meeting.personal_meeting_topic",
				// 		Other: "{{.Name}}'s Personal Meeting",
				// 	},
				// 	TemplateData: map[string]string{"Name": user.GetDisplayName(model.SHOW_NICKNAME_FULLNAME)},
				// })
				// meetingPersonal = true
			} else {
				// team, teamErr := lkp.API.GetTeam(channel.TeamId)
				// if teamErr != nil {
				// 	return "", teamErr
				// }
				// meetingTopic = p.b.LocalizeWithConfig(l, &i18n.LocalizeConfig{
				// 	DefaultMessage: &i18n.Message{
				// 		ID:    "jitsi.start_meeting.channel_meeting_topic",
				// 		Other: "{{.ChannelName}} Channel Meeting",
				// 	},
				// 	TemplateData: map[string]string{"ChannelName": channel.DisplayName},
				// })
				// meetingID = generateTeamChannelName(team.Name, channel.Name)
			}
			// lkp.API.SendEphemeralPost(lkp.bot.UserId, &model.Post{})
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
