package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

// Container represents collected container information
type Container struct {
	ID           string
	Name         string
	State        string
	ImageID      string
	ImageName    string
	PID          string
	Runtime      string
	CreateTime   int64
	Ports        []PortMapping
	Mounts       []MountPoint
	Networks     []ContainerNetwork
	Labels       map[string]string
}

type PortMapping struct {
	ContainerPort int32
	Protocol      string
	HostIP        string
	HostPort      int32
}

type MountPoint struct {
	Source      string
	Destination string
	Type        string
	Driver      string
}

type ContainerNetwork struct {
	NetworkID   string
	NetworkName string
	IPAddress   string
	Gateway     string
	MacAddress  string
}

func main() {
	fmt.Println("=== Docker Container Collection Test ===\n")
	
	cli, err := client.NewClientWithOpts(
		client.FromEnv,
		client.WithAPIVersionNegotiation(),
	)
	if err != nil {
		log.Fatalf("✗ Failed to create Docker client: %v", err)
	}
	defer cli.Close()

	// Ping to verify connection
	ctx := context.Background()
	if _, err := cli.Ping(ctx); err != nil {
		log.Fatalf("✗ Failed to ping Docker daemon: %v", err)
	}

	fmt.Println("✓ Successfully connected to Docker daemon")

	// List all containers
	containers, err := cli.ContainerList(ctx, types.ContainerListOptions{All: true})
	if err != nil {
		log.Fatalf("✗ Failed to list containers: %v", err)
	}

	fmt.Printf("✓ Found %d containers\n\n", len(containers))

	// Collect detailed information
	result := make([]Container, 0, len(containers))
	for _, ctr := range containers {
		inspect, err := cli.ContainerInspect(ctx, ctr.ID)
		if err != nil {
			log.Printf("Warning: Failed to inspect container %s: %v", ctr.ID, err)
			continue
		}

		name := strings.TrimPrefix(ctr.Names[0], "/")
		
		// Parse ports
		ports := parsePorts(inspect.NetworkSettings.Ports)
		
		// Parse mounts
		mounts := parseMounts(inspect.Mounts)
		
		// Parse networks
		networks := parseNetworks(inspect.NetworkSettings.Networks)

		cont := Container{
			ID:           ctr.ID,
			Name:         name,
			State:        ctr.State,
			ImageID:      ctr.ImageID,
			ImageName:    inspect.Config.ImageName,
			PID:          fmt.Sprintf("%d", inspect.State.Pid),
			Runtime:      "docker",
			CreateTime:   ctr.Created.Unix(),
			Ports:        ports,
			Mounts:       mounts,
			Networks:     networks,
			Labels:       inspect.Config.Labels,
		}
		result = append(result, cont)
	}

	// Verify collected data
	fmt.Println("=== Data Verification ===\n")
	
	if len(result) == 0 {
		fmt.Println("✗ No containers collected")
		os.Exit(1)
	}
	
	fmt.Printf("✓ Collected %d containers\n", len(result))
	
	// Verify container fields
	for i, ctr := range result {
		if ctr.ID == "" {
			fmt.Printf("✗ Container %d: Missing ID\n", i)
			os.Exit(1)
		}
		if ctr.Name == "" {
			fmt.Printf("✗ Container %d: Missing Name\n", i)
			os.Exit(1)
		}
		if ctr.State == "" {
			fmt.Printf("✗ Container %d: Missing State\n", i)
			os.Exit(1)
		}
		if ctr.Runtime != "docker" {
			fmt.Printf("✗ Container %d: Invalid Runtime\n", i)
			os.Exit(1)
		}
	}
	
	fmt.Println("✓ All containers have required fields")
	
	// Check for containers with different states
	running := 0
	stopped := 0
	for _, ctr := range result {
		if ctr.State == "running" {
			running++
		} else {
			stopped++
		}
	}
	fmt.Printf("✓ Running containers: %d\n", running)
	fmt.Printf("✓ Stopped containers: %d\n", stopped)
	
	// Check for containers with ports
	withPorts := 0
	for _, ctr := range result {
		if len(ctr.Ports) > 0 {
			withPorts++
		}
	}
	fmt.Printf("✓ Containers with port mappings: %d\n", withPorts)
	
	// Check for containers with mounts
	withMounts := 0
	for _, ctr := range result {
		if len(ctr.Mounts) > 0 {
			withMounts++
		}
	}
	fmt.Printf("✓ Containers with mount points: %d\n", withMounts)
	
	// Check for containers with networks
	withNetworks := 0
	for _, ctr := range result {
		if len(ctr.Networks) > 0 {
			withNetworks++
		}
	}
	fmt.Printf("✓ Containers with networks: %d\n", withNetworks)
	
	// Display sample container details
	fmt.Println("\n=== Sample Container Details ===\n")
	
	displayLimit := 3
	if len(result) < displayLimit {
		displayLimit = len(result)
	}
	
	for i := 0; i < displayLimit; i++ {
		ctr := result[i]
		fmt.Printf("Container %d:\n", i+1)
		fmt.Printf("  ID: %s\n", ctr.ID[:12])
		fmt.Printf("  Name: %s\n", ctr.Name)
		fmt.Printf("  Image: %s\n", ctr.ImageName)
		fmt.Printf("  Status: %s\n", ctr.State)
		fmt.Printf("  PID: %s\n", ctr.PID)
		fmt.Printf("  Created: %s\n", time.Unix(ctr.CreateTime, 0).Format("2006-01-02 15:04:05"))
		
		if len(ctr.Ports) > 0 {
			fmt.Printf("  Ports:\n")
			for _, port := range ctr.Ports {
				fmt.Printf("    - %s %s:%d -> %d\n", port.Protocol, port.HostIP, port.HostPort, port.ContainerPort)
			}
		}
		
		if len(ctr.Networks) > 0 {
			fmt.Printf("  Networks:\n")
			for _, net := range ctr.Networks {
				fmt.Printf("    - %s: IP=%s\n", net.NetworkName, net.IPAddress)
			}
		}
		
		if len(ctr.Mounts) > 0 {
			fmt.Printf("  Mounts: %d mount(s)\n", len(ctr.Mounts))
		}
		
		if len(ctr.Labels) > 0 {
			fmt.Printf("  Labels: %d label(s)\n", len(ctr.Labels))
		}
		
		fmt.Println()
	}
	
	fmt.Println("=== Test PASSED ✓ ===")
}

func parsePorts(ports types.PortMap) []PortMapping {
	result := []PortMapping{}
	for containerPort, bindings := range ports {
		for _, binding := range bindings {
			result = append(result, PortMapping{
				ContainerPort: int32(containerPort.Int()),
				Protocol:      containerPort.Proto(),
				HostIP:        binding.HostIP,
				HostPort:      int32(binding.HostPort),
			})
		}
	}
	return result
}

func parseMounts(mounts []types.MountPoint) []MountPoint {
	result := make([]MountPoint, 0, len(mounts))
	for _, mount := range mounts {
		result = append(result, MountPoint{
			Source:      mount.Source,
			Destination: mount.Destination,
			Type:        mount.Type,
			Driver:      mount.Driver,
		})
	}
	return result
}

func parseNetworks(networks map[string]types.NetworkResource) []ContainerNetwork {
	result := make([]ContainerNetwork, 0, len(networks))
	for name, network := range networks {
		result = append(result, ContainerNetwork{
			NetworkID:   network.NetworkID,
			NetworkName: name,
			IPAddress:   network.IPAddress,
			Gateway:     network.Gateway,
			MacAddress:  "",
		})
	}
	return result
}
