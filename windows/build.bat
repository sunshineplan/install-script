@echo off
go build -ldflags "-s -w" UpdateGeph.go
go build -ldflags "-s -w" UpdateV2Ray.go
