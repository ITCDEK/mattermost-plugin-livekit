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
	configurationLock sync.RWMutex
	configuration     *configuration
	bot               *model.Bot
	master            *kitSDK.RoomServiceClient
	Server            livekitSettings
}

// func initDBCommandContext(configDSN string, readOnlyConfigStore bool) (*app.App, error) {
// 	if err := utils.TranslationsPreInit(); err != nil {
// 		return nil, err
// 	}
// 	model.AppErrorInit(i18n.T)

// 	s, err := app.NewServer(
// 		app.Config(configDSN, readOnlyConfigStore, nil),
// 		app.StartSearchEngine,
// 		app.StartMetrics,
// 	)
// 	if err != nil {
// 		return nil, err
// 	}

// 	a := app.New(app.ServerConnector(s.Channels()))

// 	if model.BuildEnterpriseReady == "true" {
// 		a.Srv().LoadLicense()
// 	}

// 	return a, nil
// }

func (lkp *LiveKitPlugin) OnActivate() error {
	lkp.API.LogInfo("Activating...")
	configuration := lkp.getConfiguration()
	// validate configuration here
	lkp.configuration = configuration

	var err error
	//Bot
	liveBot := &model.Bot{
		UserId:      "livekit_id",
		Username:    "livekit2",
		DisplayName: "Live Bot",
		Description: "A bot account created by the LiveKit plugin",
	}

	bot, ae := lkp.API.GetBot(liveBot.UserId, true)
	if ae == nil {
		lkp.bot = bot
	} else {
		bot, ae = lkp.API.CreateBot(liveBot)
		if ae == nil {
			lkp.bot = bot
		} else {
			err = fmt.Errorf(ae.Error())
			return err
		}
	}
	command, err := lkp.compileSlashCommand()
	if err == nil {
		err = lkp.API.RegisterCommand(command)
		if err == nil {
			serverURL := fmt.Sprintf("%s:%d", lkp.configuration.Host, lkp.configuration.Port)
			lkp.master = kitSDK.NewRoomServiceClient(serverURL, lkp.configuration.ApiKey, lkp.configuration.ApiValue)
			return nil
		}
	}

	return err
}

func (lkp *LiveKitPlugin) OnDeactivate() error {
	return nil
}

// func getJoinToken(apiKey, apiSecret, room, identity string) (string, error) {}
// func (lkp *LiveKitPlugin) newRoom(userID, channelID, rootID, topic string) error {}

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
	userID := r.Header.Get("Mattermost-User-ID")
	if userID == "" {
		http.Error(w, "Not authorized", http.StatusUnauthorized)
		return
	}
	info := fmt.Sprintf("Got %s request on %s", r.Method, r.URL.Path)
	lkp.API.LogInfo(info)
	switch r.URL.Path {
	case "/webhook":
		http.Error(w, "Not authorized", http.StatusUnauthorized)
	case "/host":
		// https://github.com/matterpoll/matterpoll/blob/master/server/plugin/api.go#L324
		// https://github.com/matterpoll/matterpoll/blob/master/server/plugin/api.go#L484
		// lkp.API.SendEphemeralPost(userID, &model.Post{})
	case "/room":
		// https://stackoverflow.com/questions/57096382/response-from-interactive-button-post-is-ignored-in-mattermost
		roomRequest := struct {
			ChannelID string `json:"channel_id"`
			PartyOf   int    `json:"party"`
			Personal  bool   `json:"personal"`
			Topic     string `json:"topic"`
			RootID    string `json:"root_id"`
		}{}
		err := json.NewDecoder(r.Body).Decode(&roomRequest)
		if err == nil {
			channel, appErr := lkp.API.GetChannel(roomRequest.ChannelID)
			if appErr == nil {
				member, appErr := lkp.API.GetChannelMember(channel.Id, userID)
				if appErr == nil {
					info := fmt.Sprintf("User %s requested new live room for channel %s", member.UserId, member.ChannelId)
					lkp.API.LogInfo(info)
					room, err := lkp.master.CreateRoom(
						context.Background(),
						&livekit.CreateRoomRequest{
							Name:         roomRequest.Topic,
							Metadata:     roomRequest.ChannelID,
							EmptyTimeout: 300,
						},
					)
					if err == nil {
						lkp.API.LogInfo("room created at", room.CreationTime)
						post := &model.Post{
							UserId:    lkp.bot.UserId,
							ChannelId: channel.Id,
							RootId:    roomRequest.RootID,
							Message:   "I have started a meeting",
							Type:      "custom_livekit",
							Props: map[string]interface{}{
								"room_capacity": room.MaxParticipants,
								"room_name":     room.Name,
								"room_sid":      room.Sid,
								"room_host":     member.UserId,
								"room_server":   0,
								// "meeting_status":           zoom.WebhookStatusStarted,
								// "meeting_personal":         false,
								// "attachments":              []*model.SlackAttachment{&slackAttachment},
							},
						}
						// lkp.API.SendEphemeralPost(lkp.bot.UserId, post)
						newRoomPost, appErr := lkp.API.CreatePost(post)
						if appErr == nil {
							lkp.API.LogInfo("room post created with ID =", newRoomPost.Id)
							http.Error(w, "OK", http.StatusOK)
						} else {
							lkp.API.LogInfo(appErr.DetailedError)
							http.Error(w, appErr.DetailedError, http.StatusInternalServerError)
						}
						return
					}
				}
			}
			http.Error(w, "Forbidden", http.StatusForbidden)
		} else {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
	case "/join":
		var tokenRequest map[string]string
		err := json.NewDecoder(r.Body).Decode(&tokenRequest)
		roomName, found := tokenRequest["roomID"]
		if err == nil && found {
			info := fmt.Sprintf("User %s requested new access token for [%s]", userID, roomName)
			lkp.API.LogInfo(info)
			// options := livekit.ListRoomsRequest{Names: []string{}}
			// listing, err := lkp.master.ListRooms(context.Background(), &options)
			// if err == nil {}
			accessToken := auth.NewAccessToken(lkp.configuration.ApiKey, lkp.configuration.ApiValue)
			grant := &auth.VideoGrant{RoomJoin: true, Room: roomName}
			accessToken.AddGrant(grant).SetIdentity(userID).SetValidFor(time.Hour)
			tokenReply, err := accessToken.ToJWT()
			if err == nil {
				json.NewEncoder(w).Encode(tokenReply)
				return
			}
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
	case "/delete":
		var deleteRequest map[string]string
		err := json.NewDecoder(r.Body).Decode(&deleteRequest)
		postID, found := deleteRequest["postID"]
		if err == nil && found {
			info := fmt.Sprintf("User %s requested room deletion from post [%s]", userID, postID)
			lkp.API.LogInfo(info)
			post, appErr := lkp.API.GetPost(postID)
			postHost := post.GetProp("room_host").(string)
			roomName := post.GetProp("room_name").(string)
			if appErr == nil && postHost == userID {
				deletionRequest := livekit.DeleteRoomRequest{Room: roomName}
				result, err := lkp.master.DeleteRoom(context.Background(), &deletionRequest)
				if err == nil {
					appErr = lkp.API.DeletePost(post.Id)
					if appErr == nil {
						http.Error(w, "OK", http.StatusOK)
						return
					}
				}
				lkp.API.LogInfo(result.String())
			}
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
	case "/settings":
		json.NewEncoder(w).Encode(lkp.configuration)
	default:
		http.NotFound(w, r)
	}
}

// See https://developers.mattermost.com/extend/plugins/server/reference/
