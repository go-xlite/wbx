package app

import "embed"

//go:embed dist/*
var Content embed.FS

type App struct {
	Content embed.FS
	// Application specific fields can be added here
}

func NewApp() *App {
	return &App{Content: Content}
}
