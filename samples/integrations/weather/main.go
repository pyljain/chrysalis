package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

var (
	host = flag.String("host", "localhost", "host to connect to/listen on")
	port = flag.Int("port", 9000, "port number to connect to/listen on")
)

type Input struct {
	City string `json:"city" jsonschema:"the name of the city to fetch weather for"`
}

type Output struct {
	Temperature string `json:"temperature" jsonschema:"temperature in the given city"`
	Conditions  string `json:"conditions" jsonschema:"weather conditions in the given city"`
}

func GetWeather(ctx context.Context, req *mcp.CallToolRequest, input Input) (
	*mcp.CallToolResult,
	Output,
	error,
) {
	return nil, Output{Temperature: fmt.Sprintf("%d C", 21), Conditions: "Sunny"}, nil
}

func main() {
	flag.Parse()
	server := mcp.NewServer(&mcp.Implementation{Name: "Weather API", Version: "v1.0.0"}, nil)
	mcp.AddTool(server, &mcp.Tool{Name: "GetWeather", Description: "Gets weather for the given city"}, GetWeather)

	handler := mcp.NewStreamableHTTPHandler(func(req *http.Request) *mcp.Server {
		return server
	}, nil)

	log.Printf("MCP server listening on %d", *port)

	// Start the HTTP server.
	if err := http.ListenAndServe(fmt.Sprintf(":%d", *port), handler); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}
