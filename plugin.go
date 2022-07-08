package root

import (
	// _ "embed" // Need to embed manifest file
	// "encoding/json"
	// "strings"

	"github.com/mattermost/mattermost-server/v6/model"
)

// var manifestString string

var Manifest model.Manifest

func init() {
	Manifest.Name = "VoiceMatters"
	Manifest.Version = "0.1.0"
	Manifest.MinServerVersion = "6.0.0"
	Manifest.Id = "com.mattermost.plugin-livekit"
	Manifest.IconPath = "assets/bot-icon.svg"
	var properties map[string]interface{}
	Manifest.Props = properties
	exeList := map[string]string{
		"linux-amd64": "server/dist/plugin-linux-amd64",
	}
	Manifest.Server = &model.ManifestServer{Executables: exeList}
	Manifest.SettingsSchema = &model.PluginSettingsSchema{
		Header: "For the detailed settings description, please visit https://github.com/ITCDEK/mattermost-plugin-livekit",
		Footer: "---- VoiceMatters settings ----",
		Settings: []*model.PluginSetting{
			// {
			// 	DisplayName: "Signaling server URI",
			// 	Key:         "LiveKitString",
			// 	Type:        "text",
			// 	Default:     "",
			// 	Placeholder: "http(s)://livekit-01.on.me:7880",
			// 	HelpText:    "Signaling server URI for selected region",
			// },
			// {
			// 	DisplayName: "TURN server URI",
			// 	Key:         "TurnString",
			// 	Type:        "text",
			// 	Default:     "",
			// 	Placeholder: "http(s)://turn-01.on.me:5349, 443",
			// 	HelpText:    "If enabled, specify TURN server URI for the region",
			// },
			// {
			// 	DisplayName: "API secret key",
			// 	Key:         "SecretString",
			// 	Type:        "text",
			// 	Default:     "",
			// 	Placeholder: "API7vTUvag3wqvW: mqtMUyGxMzzw7tUfGmlYIo4utTs66svftv8MRh1HTxp",
			// 	HelpText:    "The LiveKit API key to create user credentials",
			// },
			{
				DisplayName: "Server #1",
				Key:         "Server1",
				Type:        "text",
				Default:     "{host: '172.16.42.228', secure: false, port: 7880, turnhost: '172.16.42.228', turnport: 5349, turnsecure: false, turnudp: 443, apikey: 'API7vTUvag3wqvW', apivalue: 'mqtMUyGxMzzw7tUfGmlYIo4utTs66svftv8MRh1HTxp'}",
				Placeholder: "{host: 'livekit.on.me', port: 7880}",
				HelpText:    "JSON-formatted settings, see plugin documentation on GitHub",
			},
		},
	}
	Manifest.Description = "LiveKit audio and video conferencing plugin for Mattermost"
	Manifest.HomepageURL = "https://github.com/ITCDEK/mattermost-plugin-livekit"
	Manifest.SupportURL = "https://github.com/ITCDEK/mattermost-plugin-livekit/issues"
	Manifest.ReleaseNotesURL = "https://github.com/ITCDEK/mattermost-plugin-livekit/releases/tag/v0.1.0"
	// _ = json.NewDecoder(strings.NewReader(manifestString)).Decode(&Manifest)
}
