package comm

import (
	mime "github.com/go-xlite/wbx/comm/mime"
	servercore "github.com/go-xlite/wbx/comm/server_core"
)

type ServerCore = servercore.ServerCore

var NewServerCore = servercore.NewServerCore

type mim struct {
	GetType           func(ext string) string
	IsStaticExtension func(ext string) bool
	Type              *mime.MimeTypes
}

var Mime = mim{
	GetType:           mime.GetMimeType,
	IsStaticExtension: mime.IsStaticExtension,
	Type:              &mime.Mime,
}
