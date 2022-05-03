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
	"github.com/mattermost/mattermost-server/v6/app"
	"github.com/mattermost/mattermost-server/v6/config"
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
	storeInstance     *config.Store
	serverInstance    *app.Server
	appInstance       *app.App
	Server            livekitSettings
	ROSID             string
	// appInstance	*app.App
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
	//Read-only scheme of roles
	// appConfig := lkp.API.GetUnsanitizedConfig()
	// fmt.Println(appConfig.SqlSettings.DataSource)
	// a, err := initDBCommandContext("config.json", false)
	// a.PatchChannelModerationsForChannel(&model.Channel{})
	// Client := model.NewAPIv4Client("http://localhost:8065")
	// Client.Login("Denis", "##332211qqwweE")
	// scheme, resp, err := Client.GetScheme("read_only_scheme")
	// if err == nil {
	// 	lkp.API.LogInfo("Read-only scheme obtained successefully")
	// 	lkp.ROSID = scheme.Id
	// 	fmt.Println(resp)
	// } else {
	// 	adminRole := model.Role{SchemeManaged: true}
	// 	patch := model.RolePatch{Permissions: &[]string{"add_reaction", "remove_reaction"}}
	// 	userRole, resp, err := Client.PatchRole(nil, &patch)
	// 	guestRole := model.Role{SchemeManaged: true}
	// 	readonlyScheme := model.Scheme{
	// 		Id:                      "read-only-scheme",
	// 		Name:                    "read-only",
	// 		DisplayName:             "",
	// 		Description:             "Read-only scheme",
	// 		DefaultChannelAdminRole: adminRole.Id,
	// 		DefaultChannelUserRole:  userRole.Id,
	// 		DefaultChannelGuestRole: guestRole.Id,
	// 		Scope:                   "channel",
	// 	}
	// 	// Client.UpdateChannelScheme()
	// 	scheme, resp, err = Client.CreateScheme(&readonlyScheme)
	// 	fmt.Println(readonlyScheme)
	// }

	configStore, err := config.NewStoreFromDSN("config.json", false, nil)
	if err != nil {
		lkp.API.LogWarn("failed to obtain configStore", "error", err.Error())
	}
	// defer configStore.Close()
	options := []app.Option{app.ConfigStore(configStore)}
	server, err := app.NewServer(options...)
	if err != nil {
		lkp.API.LogWarn("failed to obtain server", "error", err.Error())
	}
	// defer server.Shutdown()
	//
	// api := api4.Init(server)
	// if err != nil {
	// 	lkp.API.LogWarn("failed to obtain API instance", err.Error())
	// }
	// appInstance := app.New(app.ServerConnector(api.srv.Channels()))
	appInstance := app.New(app.ServerConnector(server.Channels()))
	// model.NewTestLicense("custom_permissions_schemes")
	// App.SetPhase2PermissionsMigrationStatus(true)
	readonlyScheme, ae := appInstance.GetSchemeByName("read_only")
	if ae == nil {
		lkp.API.LogInfo("read-only scheme found", "id", readonlyScheme.Id)
	} else {
		scheme := model.Scheme{
			Name:        "read_only",
			DisplayName: "Read-only scheme",
			Description: "Read-only scheme",
			// DefaultChannelAdminRole: adminRole.Id,
			// DefaultChannelUserRole:  userRole.Id,
			// DefaultChannelGuestRole: guestRole.Id,
			Scope: model.SchemeScopeChannel,
		}
		readonlyScheme, ae = appInstance.CreateScheme(&scheme)
		if ae == nil {
			lkp.API.LogInfo("read-only scheme created", "id", readonlyScheme.Id)
		} else {
			lkp.API.LogWarn("read-only scheme fails", "reason", ae.DetailedError)
		}
	}
	roleset, ae := appInstance.GetRolesByNames([]string{readonlyScheme.DefaultChannelAdminRole, readonlyScheme.DefaultChannelUserRole, readonlyScheme.DefaultChannelGuestRole})
	if ae == nil && len(roleset) == 3 {
		adminRole := roleset[0]
		userRole := roleset[1]
		guestRole := roleset[2]
		if userRole.Permissions[0] != "add_reaction" {
			patch := model.RolePatch{
				Permissions: &[]string{"add_reaction", "remove_reaction"},
			}
			patchedUserRole, ae := appInstance.PatchRole(userRole, &patch)
			if ae == nil {
				lkp.API.LogInfo("read-only User role patched", "id", patchedUserRole.Id)
				fmt.Println(patchedUserRole.Permissions)
			} else {
				lkp.API.LogInfo("read-only user role patch fails", "reason", ae.DetailedError)
			}
			patchedGuestRole, ae := appInstance.PatchRole(guestRole, &patch)
			if ae == nil {
				lkp.API.LogInfo("read-only Guest role patched", "id", patchedGuestRole.Id)
				fmt.Println(patchedGuestRole.Permissions)
			} else {
				lkp.API.LogInfo("read-only guest role patch fails", "reason", ae.DetailedError)
			}
		}
		fmt.Println("Read-only admin role id =", adminRole.Id)
	} else {
		lkp.API.LogInfo("read-only role set fails", "reason", ae.DetailedError)
	}
	testChannel, ae := appInstance.GetChannel("madgimq6iidium8dpc6w7hf5we")
	if ae == nil && testChannel.SchemeId != &readonlyScheme.Id {
		testChannel.SchemeId = &readonlyScheme.Id
		updatedChannel, ae := appInstance.UpdateChannelScheme(testChannel)
		if ae == nil {
			lkp.API.LogInfo("Channel got read-only SchemeID", "id", *updatedChannel.SchemeId)
		} else {
			lkp.API.LogInfo("setting SchemeID fails", "reason", ae.DetailedError)
		}
	} else {
		lkp.API.LogInfo("setting SchemeID fails", "reason", ae.DetailedError)
	}
	lkp.storeInstance = configStore
	lkp.serverInstance = server
	lkp.appInstance = appInstance

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
			serverURL := fmt.Sprintf("%s:%d", lkp.Server.Host, lkp.Server.Port)
			lkp.master = kitSDK.NewRoomServiceClient(serverURL, lkp.Server.ApiKey, lkp.Server.ApiValue)
			return nil
		}
	}

	return err
}

func (lkp *LiveKitPlugin) OnDeactivate() error {
	lkp.storeInstance.Close()
	lkp.serverInstance.Shutdown()
	lkp.appInstance = nil
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
		fmt.Fprint(w, "Hello, world! This hook is not implemented yet.")
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
			server := lkp.Server
			accessToken := auth.NewAccessToken(server.ApiKey, server.ApiValue)
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
		settings := struct {
			Server livekitSettings
			ROSID  string
		}{Server: lkp.Server, ROSID: lkp.ROSID}
		json.NewEncoder(w).Encode(settings)
	case "/scheme":
		patch := struct {
			SchemeID  string `json:"scheme_id"`
			ChannelID string `json:"channel_id"`
		}{}
		var reply map[string]string
		jsonErr := json.NewDecoder(r.Body).Decode(&patch)
		if jsonErr == nil {
			// session, ae := lkp.API.GetSession(c.SessionId)
			// if lkp.appInstance.SessionHasPermissionTo(*session, model.PermissionManageSystem) && ae == nil {
			// }
			channel, chErr := lkp.API.GetChannel(patch.ChannelID)
			scheme, schErr := lkp.appInstance.GetScheme(patch.SchemeID)
			if chErr == nil && schErr == nil {
				channel.SchemeId = &scheme.Id
				newChannel, ae := lkp.appInstance.UpdateChannel(channel)
				if ae == nil {
					lkp.API.LogInfo("Channel scheme set to", "id", newChannel.SchemeId)
					reply["result"] = "ok"
				} else {
					lkp.API.LogWarn("Setting scheme failed", "error", ae.DetailedError)
					reply["result"] = ae.DetailedError
				}
			} else {
				reply["result"] = chErr.DetailedError + "|" + schErr.DetailedError
			}
		} else {
			reply["result"] = jsonErr.Error()
		}
		json.NewEncoder(w).Encode(reply)
	default:
		http.NotFound(w, r)
	}
}

// See https://developers.mattermost.com/extend/plugins/server/reference/
