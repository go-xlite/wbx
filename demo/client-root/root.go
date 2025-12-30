package clientroot

import (
	"embed"

	embed_fs "github.com/go-xlite/wbx/comm/adapter_fs/embed_fs"
)

//go:embed dist/*
var content embed.FS

type ClientRoot struct {
	Content *embed_fs.EmbedFS
	// Application specific fields can be added here
}

func NewClientRoot() *ClientRoot {
	clr := &ClientRoot{Content: embed_fs.NewEmbedFS(&content)}
	clr.Content.SetBasePath("dist")
	return clr
}
