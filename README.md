# Mattermost plugin for LiveKit

This [Mattermost](https://github.com/mattermost/mattermost-server) plug-in provides integration with the [LiveKit](https://github.com/livekit/livekit) audio- and video-conferencing server.

## Installation guide

Go to the Releases section and download file named `com.mattermost.plugin-livekit-0.x.x.tar.gz`. Then upload this bundle using System console GUI on your Mattermost server.
As of now, these two settings will get you going: `Host` (ie. livekit.myhost.org) and `Host port` (that's 7880 by default).

## Developer's guide
--- For using Makefile.go, install mage:

```Shell
cd ~
wget https://github.com/magefile/mage/releases/download/v1.13.0/mage_1.13.0_Linux-64bit.tar.gz
sudo tar -C /usr/local/go/bin -xzf mage_1.13.0_Linux-64bit.tar.gz
rm mage_1.13.0_Linux-64bit.tar.gz
```

To use it, run `mage build` and `mage deploy`.
To build static executable, run `mage -compile make`.
To execute build and deploy in one go with static executable, use `./make install`.

--- To set up Node.js

```Shell
curl -o- https://raw.githubusercontent.com/nvm-sh/nvm/v0.39.1/install.sh | bash
nvm install --lts
cd webapp && npm i --legacy-peer-deps
```

--- To install \ update Go

```Shell
sudo rm -rf /usr/local/go
wget "https://go.dev/dl/$(curl 'https://go.dev/VERSION?m=text').linux-amd64.tar.gz" && sudo tar -C /usr/local -xzf go*.linux-amd64.tar.gz
```
--- Check out the Livekit playground

https://livekit.io/playground
