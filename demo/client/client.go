package client

import (
	"embed"

	embed_fs "github.com/go-xlite/wbx/comm/adapter_fs/embed_fs"
)

//go:embed dist/*
var Content embed.FS

type Client struct {
	AppG *embed_fs.EmbedFS
	AppW *embed_fs.EmbedFS
	// Application specific fields can be added here
}

func NewClient() *Client {
	cl := &Client{AppG: embed_fs.NewEmbedFS(&Content)}
	cl.AppG.SetBasePath("dist/app_g")
	cl.AppW = embed_fs.NewEmbedFS(&Content)
	cl.AppW.SetBasePath("dist/app_w")
	return cl
}
