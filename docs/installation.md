# Installation

We have multiple ways to install Kompose. Our preferred method is downloading the binary from the latest GitHub release.

#### GitHub release

Kompose is released via GitHub on a three-week cycle, you can see all current releases on the [GitHub release page](https://github.com/kubernetes/kompose/releases).

__Linux and macOS:__

```sh
# Linux
curl -L https://github.com/AliyunContainerService/kompose/releases/download/v0.0.3/kompose-linux-amd64 -o kompose

# macOS
curl -L https://github.com/AliyunContainerService/kompose/releases/download/v0.0.3/kompose-darwin-amd64 -o kompose

chmod +x kompose
sudo mv ./kompose /usr/local/bin/kompose
```

__Windows:__

Download from [GitHub](https://github.com/AliyunContainerService/kompose/releases/download/v0.0.3/kompose-windows-amd64.exe) and add the binary to your PATH.

#### Go

Installing using `go get` pulls from the master branch with the latest development changes.

```sh
go get -u github.com/AliyunContainerService/kompose
```
