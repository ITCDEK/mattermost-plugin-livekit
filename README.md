--- For using Makefile.go, install mage:

```Shell
cd ~
wget https://github.com/magefile/mage/releases/download/v1.13.0/mage_1.13.0_Linux-64bit.tar.gz
sudo tar -C /usr/local/go/bin -xzf mage_1.13.0_Linux-64bit.tar.gz
rm mage_1.13.0_Linux-64bit.tar.gz
```

To use it, run `mage build` and `mage deploy`.
To build static executable, run `mage -compile make`.

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
