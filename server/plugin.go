package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/livekit/protocol/auth"
	"github.com/livekit/protocol/livekit"
	kitSDK "github.com/livekit/server-sdk-go"
	pluginSDK "github.com/mattermost/mattermost-plugin-api"
	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/mattermost/mattermost-server/v6/plugin"
	"github.com/pkg/errors"
)

// References
// https://github.com/mattermost/mattermost-plugin-calls
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
	botUserID         string
	configurationLock sync.RWMutex
	configuration     *configuration
	master            *kitSDK.RoomServiceClient
	sdk               *pluginSDK.Client
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

// func getJoinToken(apiKey, apiSecret, room, identity string) (string, error) {}
// func (lkp *LiveKitPlugin) newRoom(userID, channelID, rootID, topic string) error {}

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

func (lkp *LiveKitPlugin) createPost(channelID, userID, text string, maxParticipants uint32) *model.AppError {
	post := &model.Post{
		UserId:    lkp.botUserID,
		ChannelId: channelID,
		Message:   text,
		Type:      "custom_livekit",
		Props: map[string]interface{}{
			"room_capacity": maxParticipants,
			"room_host":     userID,
			// "attachments": []*model.SlackAttachment{&model.SlackAttachment{}},
		},
	}
	// lkp.API.SendEphemeralPost(lkp.bot.UserId, post)
	newRoomPost, appErr := lkp.API.CreatePost(post)
	if appErr == nil {
		lkp.API.LogInfo("room created", "id", newRoomPost.Id)
		return nil
	}
	return appErr
}

func (lkp *LiveKitPlugin) writeErrorResponse(w http.ResponseWriter, text string) {}

// ServeHTTP demonstrates a plugin that handles HTTP requests by greeting the world.
func (lkp *LiveKitPlugin) ServeHTTP(c *plugin.Context, w http.ResponseWriter, r *http.Request) {
	clientResponse := struct {
		Status string
		Error  string
	}{Status: "error"}
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
	case "/join":
		var room *livekit.Room
		mvpRequest := struct {
			PostID string `json:"post_id"`
		}{}
		err := json.NewDecoder(r.Body).Decode(&mvpRequest)
		if err == nil {
			post, ae := lkp.API.GetPost(mvpRequest.PostID)
			if ae != nil {
				http.Error(w, ae.DetailedError, http.StatusNotFound)
				return
			}
			_, ae = lkp.API.GetChannelMember(post.ChannelId, userID)
			if ae != nil {
				http.Error(w, ae.DetailedError, http.StatusExpectationFailed)
				return
			}
			tokenUser, ae := lkp.API.GetUser(userID)
			if ae != nil {
				http.Error(w, ae.DetailedError, http.StatusExpectationFailed)
				return
			}
			lkp.API.LogInfo("room token requested", "post_id", mvpRequest.PostID)
			roomList, err := lkp.master.ListRooms(
				context.Background(),
				&livekit.ListRoomsRequest{Names: []string{mvpRequest.PostID}},
			)
			if err == nil && len(roomList.Rooms) > 0 {
				room = roomList.Rooms[0]
				lkp.API.LogInfo("room found", "name", room.Name)
			} else {
				n := post.GetProp("room_capacity").(float64)
				castedN := uint32(n)
				newRoom, err := lkp.master.CreateRoom(
					context.Background(),
					&livekit.CreateRoomRequest{
						Name:            mvpRequest.PostID,
						Metadata:        userID,
						EmptyTimeout:    300,
						MaxParticipants: castedN,
					},
				)
				if err == nil {
					room = newRoom
					lkp.API.LogInfo("room created", "name", room.Name)
				} else {
					lkp.API.LogError("room creation failed", "reason", err.Error())
				}
			}
			lkp.API.LogInfo("room assigned", "pointer", room, "name", room.Name)
			accessToken := auth.NewAccessToken(lkp.configuration.ApiKey, lkp.configuration.ApiValue)
			grant := &auth.VideoGrant{RoomJoin: true, Room: room.Name}
			userName := tokenUser.GetDisplayName("full_name")
			accessToken.AddGrant(grant).SetValidFor(time.Hour * 12).SetIdentity(userID).SetName(userName)
			tokenReply, err := accessToken.ToJWT()
			if err == nil {
				json.NewEncoder(w).Encode(tokenReply)
				return
			}
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
	case "/rooms":
		roomList, err := lkp.master.ListRooms(context.Background(), &livekit.ListRoomsRequest{})
		if err == nil {
			json.NewEncoder(w).Encode(roomList)
		} else {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	case "/host":
		// https://github.com/matterpoll/matterpoll/blob/master/server/plugin/api.go#L324
		// https://github.com/matterpoll/matterpoll/blob/master/server/plugin/api.go#L484
		// lkp.API.SendEphemeralPost(userID, &model.Post{})
	case "/create":
		// https://stackoverflow.com/questions/57096382/response-from-interactive-button-post-is-ignored-in-mattermost
		roomRequest := struct {
			ChannelID string `json:"channel_id"`
			Capacity  uint32 `json:"capacity"`
			Message   string `json:"message"`
		}{}
		err := json.NewDecoder(r.Body).Decode(&roomRequest)
		if err == nil {
			channel, appErr := lkp.API.GetChannel(roomRequest.ChannelID)
			if appErr == nil {
				member, appErr := lkp.API.GetChannelMember(channel.Id, userID)
				if appErr == nil {
					info := fmt.Sprintf("User %s requested new live room for channel %s", member.UserId, member.ChannelId)
					lkp.API.LogInfo(info)
					appErr = lkp.createPost(roomRequest.ChannelID, member.UserId, roomRequest.Message, roomRequest.Capacity)
					if appErr == nil {
						clientResponse.Status = "OK"
						json.NewEncoder(w).Encode(clientResponse)
					} else {
						clientResponse.Error = appErr.DetailedError
						json.NewEncoder(w).Encode(clientResponse)
					}
					return
				}
				http.Error(w, appErr.DetailedError, http.StatusForbidden)
				return
			}
			http.Error(w, appErr.DetailedError, http.StatusBadRequest)
		} else {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
	case "/old":
		var tokenRequest map[string]string
		err := json.NewDecoder(r.Body).Decode(&tokenRequest)
		roomPost, ae := lkp.API.GetPost(tokenRequest["postID"])
		roomName, ok := roomPost.GetProp("room_name").(string)
		if err == nil && ae == nil && ok {
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
				props := roomPost.GetProps()
				props[userID] = tokenReply
				roomPost.SetProps(props)
				_, ae = lkp.API.UpdatePost(roomPost)
				if ae == nil {
					json.NewEncoder(w).Encode(tokenReply)
				}
				return
			}
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
	case "/delete":
		var deleteRequest map[string]string
		err := json.NewDecoder(r.Body).Decode(&deleteRequest)
		postID, found := deleteRequest["post_id"]
		if err == nil && found {
			info := fmt.Sprintf("User %s requested room deletion from post [%s]", userID, postID)
			lkp.API.LogInfo(info)
			appErr := lkp.API.DeletePost(postID)
			// postHost := post.GetProp("room_host").(string)
			if appErr == nil {
				clientResponse.Status = "OK"
				json.NewEncoder(w).Encode(clientResponse)
				return
			}
			err = fmt.Errorf("%s", appErr.DetailedError)
		}
		clientResponse.Error = err.Error()
		json.NewEncoder(w).Encode(clientResponse)
	case "/settings":
		copy := *lkp.configuration
		copy.ApiKey = "n/a"
		copy.ApiValue = "n/a"
		json.NewEncoder(w).Encode(copy)
	default:
		http.NotFound(w, r)
	}
}

// See https://developers.mattermost.com/extend/plugins/server/reference/
