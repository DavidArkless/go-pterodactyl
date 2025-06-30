package main

import (
	"context"
	"fmt"
	"github.com/davidarkless/go-pterodactyl"
	"github.com/davidarkless/go-pterodactyl/api"
	"log"
	"os"
	"strconv"
	"time"
)

// ptr is a tiny helper that returns a pointer to the value passed in. This is
// handy when a struct field is defined as *string or *bool but you have a
// ptr returns a pointer to the given value. Useful for creating pointers to literals or temporary values.
func ptr[T any](v T) *T { return &v }

// main creates a new server on a Pterodactyl panel using environment variables for configuration, then polls until the server is installed or a timeout occurs.
// 
// The function reads required environment variables, initializes the Pterodactyl client, constructs server creation options, and submits a server creation request. It then periodically checks the server's installation status, printing progress and exiting when the server is ready or if a timeout is reached. Fatal errors are logged and terminate execution.
func main() {

	baseURL := os.Getenv("PTERO_BASE_URL")
	apiKey := os.Getenv("PTERO_API_KEY")
	if baseURL == "" || apiKey == "" {
		log.Fatal("Set PTERO_BASE_URL and PTERO_API_KEY environment variables")
	}
	userID, err := strconv.Atoi(os.Getenv("PTERO_USER_ID"))
	if err != nil {
		log.Fatal("Set PTERO_USER_ID environment variable")
	}

	// Initialise SDK with a short default timeout; callers can override via ctx.
	client, err := pterodactyl.NewClient(baseURL, apiKey, pterodactyl.ApplicationKey, pterodactyl.WithTimeout(10*time.Second))
	if err != nil {
		log.Fatalf("new client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	startup := "java -Xms128M -Xmx{{SERVER_MEMORY}}M -jar {{SERVER_JARFILE}}"

	// --- build options ------------------------------------------------------
	opts := api.ServerCreateOptions{
		Name:        "example-mc-server-manual",
		Description: ptr("Provisioned via go-pterodactyl example"),
		User:        userID,
		Egg:         4,
		Nest:        1,
		DockerImage: "ghcr.io/pterodactyl/yolks:java_21",

		NodeID: ptr(1),
		Allocation: &struct {
			Default    int   `json:"default"`
			Additional []int `json:"additional,omitempty"`
		}{Default: 191},
		Limits: api.ServerLimits{
			Memory: 2048,
			Swap:   0,
			Disk:   10000,
			IO:     500,
			CPU:    0,
		},
		Startup: startup,
		Environment: &map[string]string{
			"MINECRAFT_VERSION": "1.20.4",
			"SERVER_JARFILE":    "server.jar",
			"JAVA_VERSION":      "17",
			"BUILD_NUMBER":      "latest",
		},
		StartWhenCreated: ptr(true),
	}

	srv, err := client.ApplicationAPI.Servers.Create(ctx, opts)
	if err != nil {
		log.Fatalf("create server: %v", err)
	}

	fmt.Printf("Server queued: %s\n", srv.UUID)

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			log.Fatalf("timed out waiting for install")
		case <-ticker.C:
			s, err := client.ApplicationAPI.Servers.Get(ctx, srv.ID)
			if err != nil {
				log.Printf("get server: %v", err)
				continue
			}
			if s.Container.Installed {
				fmt.Printf("✓ server %s installed and ready!\n", s.UUID)
				return
			}
			fmt.Print(".")
		}
	}
}
