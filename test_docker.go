package main

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

func main() {
	cli, err := client.NewClientWithOpts(
		client.FromEnv,
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		log.Fatalf("Failed to create Docker client: %v", err)
	}
	defer cli.Close()

	// Ping to verify connection
	ctx := context.Background()
	if _, err := cli.Ping(ctx); err != nil {
		log.Fatalf("Failed to ping Docker daemon: %v", err)
	}

	fmt.Println("✓ Successfully connected to Docker daemon")

	// List all containers
	containers, err := cli.ContainerList(ctx, types.ContainerListOptions{All: true})
	if err != nil {
		log.Fatalf("Failed to list containers: %v", err)
	}

	fmt.Printf("\n=== Found %d containers ===\n\n", len(containers))

	for _, ctr := range containers {
		inspect, err := cli.ContainerInspect(ctx, ctr.ID)
		if err != nil {
			log.Printf("Warning: Failed to inspect container %s: %v", ctr.ID, err)
			continue
		}

		name := strings.TrimPrefix(ctr.Names[0], "/")
		fmt.Printf("ID: %s\n", ctr.ID[:12])
		fmt.Printf("Name: %s\n", name)
		fmt.Printf("Image: %s\n", inspect.Config.Image)
		fmt.Printf("Status: %s\n", ctr.State)
		fmt.Printf("Created: %s\n", ctr.Created.Format("2006-01-02 15:04:05"))
		fmt.Printf("PID: %d\n", inspect.State.Pid)
		
		// Ports
		if len(ctr.Ports) > 0 {
			fmt.Printf("Ports:\n")
			for _, port := range ctr.Ports {
				if port.PublicPort != 0 {
					fmt.Printf("  - %s %d -> %d\n", port.Type, port.PublicPort, port.PrivatePort)
				}
			}
		}
		
		// Networks
		if len(inspect.NetworkSettings.Networks) > 0 {
			fmt.Printf("Networks:\n")
			for netName, net := range inspect.NetworkSettings.Networks {
				fmt.Printf("  - %s: IP=%s, MAC=%s\n", netName, net.IPAddress, net.MacAddress)
			}
		}
		
		// Mounts
		if len(inspect.Mounts) > 0 {
			fmt.Printf("Mounts:\n")
			for _, mount := range inspect.Mounts {
				fmt.Printf("  - %s: %s -> %s (%s)\n", mount.Type, mount.Source, mount.Destination, mount.Driver)
			}
		}
		
		// Labels
		if len(inspect.Config.Labels) > 0 {
			fmt.Printf("Labels:\n")
			for k, v := range inspect.Config.Labels {
				fmt.Printf("  - %s: %s\n", k, v)
			}
		}
		
		fmt.Println()
	}
}
