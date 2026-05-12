package util

import (
	"testing"

	"gaiasec-nodeagent/pkg/pb"
	"google.golang.org/protobuf/proto"
)

func TestSanitizeProtoStringsNestedMessages(t *testing.T) {
	message := &pb.ProcessesResponse{
		Processes: []*pb.Process{
			{
				Name:    string([]byte{'n', 'g', 0xff, 'i', 'n', 'x'}),
				Cmdline: string([]byte{'/', 'b', 'i', 'n', '/', 0xfe, 's', 'h'}),
				User:    string([]byte{'r', 'o', 0xff, 'o', 't'}),
			},
		},
	}

	SanitizeProtoStrings(message)

	if _, err := proto.Marshal(message); err != nil {
		t.Fatalf("marshal sanitized processes response: %v", err)
	}

	process := message.Processes[0]
	if process.Name != "nginx" {
		t.Fatalf("Name = %q, want %q", process.Name, "nginx")
	}
	if process.Cmdline != "/bin/sh" {
		t.Fatalf("Cmdline = %q, want %q", process.Cmdline, "/bin/sh")
	}
	if process.User != "root" {
		t.Fatalf("User = %q, want %q", process.User, "root")
	}
}

func TestSanitizeProtoStringsRepeatedNestedFileNodes(t *testing.T) {
	message := &pb.FSListDirResponse{
		Files: []*pb.FileNode{
			{
				Name: "ok",
				Path: string([]byte{'/', 't', 'm', 'p', '/', 0xff, 'a'}),
				Link: string([]byte{'/', 'v', 'a', 'r', '/', 0xfe, 'b'}),
			},
		},
	}

	SanitizeProtoStrings(message)

	if _, err := proto.Marshal(message); err != nil {
		t.Fatalf("marshal sanitized fs response: %v", err)
	}

	node := message.Files[0]
	if node.Path != "/tmp/a" {
		t.Fatalf("Path = %q, want %q", node.Path, "/tmp/a")
	}
	if node.Link != "/var/b" {
		t.Fatalf("Link = %q, want %q", node.Link, "/var/b")
	}
}
