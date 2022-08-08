package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/livekit/protocol/auth"
	"github.com/livekit/protocol/livekit"
	"github.com/mattermost/mattermost-server/v6/model"
	"github.com/mattermost/mattermost-server/v6/plugin"
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

//client4.doFetch requires JSON response
type fetchResponse struct {
	Status string      `json:"status"`
	Error  string      `json:"error,omitempty"`
	Data   interface{} `json:"data,omitempty"`
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

// ServeHTTP demonstrates a plugin that handles HTTP requests by greeting the world.
func (lkp *LiveKitPlugin) ServeHTTP(c *plugin.Context, w http.ResponseWriter, r *http.Request) {
	reply := fetchResponse{Status: "error"}
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
		tokenRequest := struct {
			PostID string `json:"post_id"`
		}{}
		err := json.NewDecoder(r.Body).Decode(&tokenRequest)
		if err == nil {
			post, postAE := lkp.API.GetPost(tokenRequest.PostID)
			_, cmAE := lkp.API.GetChannelMember(post.ChannelId, userID)
			tokenUser, userAE := lkp.API.GetUser(userID)
			if postAE != nil || cmAE != nil || userAE != nil {
				reply.Error = fmt.Sprintf("%s\n%s\n%s", postAE.DetailedError, cmAE.DetailedError, userAE.DetailedError)
				json.NewEncoder(w).Encode(reply)
				return
			}
			lkp.API.LogInfo("room token requested", "post_id", tokenRequest.PostID)
			roomList, err := lkp.master.ListRooms(
				context.Background(),
				&livekit.ListRoomsRequest{Names: []string{tokenRequest.PostID}},
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
						Name:            tokenRequest.PostID,
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
			accessToken := auth.NewAccessToken(lkp.configuration.ApiKey, lkp.configuration.ApiValue)
			grant := &auth.VideoGrant{RoomJoin: true, Room: room.Name}
			userName := tokenUser.GetDisplayName("full_name")
			accessToken.AddGrant(grant).SetValidFor(time.Hour * 12).SetIdentity(userID).SetName(userName)
			jwt, err := accessToken.ToJWT()
			if err == nil {
				reply.Status = "OK"
				reply.Data = jwt
				json.NewEncoder(w).Encode(reply)
				return
			}
		}
		reply.Error = err.Error()
		json.NewEncoder(w).Encode(reply)
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
						reply.Status = "OK"
					} else {
						reply.Error = appErr.DetailedError
					}
					json.NewEncoder(w).Encode(reply)
					return
				}
				http.Error(w, appErr.DetailedError, http.StatusForbidden)
				return
			}
			http.Error(w, appErr.DetailedError, http.StatusBadRequest)
		} else {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
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
				reply.Status = "OK"
				json.NewEncoder(w).Encode(reply)
				return
			}
			err = fmt.Errorf("%s", appErr.DetailedError)
		}
		reply.Error = err.Error()
		json.NewEncoder(w).Encode(reply)
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
