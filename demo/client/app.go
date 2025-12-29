package client

import "embed"

//go:embed dist/*
var Content embed.FS

type Client struct {
	Content embed.FS
	// Application specific fields can be added here
}

func NewClient() *Client {
	return &Client{Content: Content}
}
