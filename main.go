package main

import (
	"os"

	"github.com/adrianliechti/wingman-realtime-proxy/pkg/server"
	"github.com/adrianliechti/wingman-realtime-proxy/provider"
	"github.com/adrianliechti/wingman-realtime-proxy/provider/azure"
	"github.com/adrianliechti/wingman-realtime-proxy/provider/openai"
)

func main() {
	port := os.Getenv("PORT")

	if port == "" {
		port = "8080"
	}

	var handler provider.Handler

	if h, err := azure.NewFromEnvironment(); err == nil {
		handler = h
	}

	if h, err := openai.NewFromEnvironment(); err == nil {
		handler = h
	}

	if handler == nil {
		panic("No provider configured")
	}

	server := server.New(handler)

	if err := server.ListenAndServe(":" + port); err != nil {
		panic(err)
	}
}
