package client

import (
	"embed"

	embed_fs "github.com/go-xlite/wbx/comm/adapter_fs/embed_fs"
)

//go:embed dist/*
var Content embed.FS

type Client struct {
	Content *embed_fs.EmbedFS
	// Application specific fields can be added here
}

func NewClient() *Client {
	cl := &Client{Content: embed_fs.NewEmbedFS(&Content)}
	cl.Content.SetBasePath("dist")
	return cl
}
