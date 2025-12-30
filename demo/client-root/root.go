package clientroot

import "embed"

//go:embed dist/*
var content embed.FS

type ClientRoot struct {
	Content embed.FS
	// Application specific fields can be added here
}

func NewClientRoot() *ClientRoot {
	return &ClientRoot{Content: content}
}
