package process

import "testing"

func TestSanitizeProcessStringRemovesInvalidUTF8(t *testing.T) {
	input := string([]byte{'n', 'g', 0xff, 'i', 'n', 'x'})

	got := sanitizeProcessString(input, "fallback")

	if got != "nginx" {
		t.Fatalf("sanitizeProcessString() = %q, want %q", got, "nginx")
	}
}

func TestSanitizeProcessStringFallsBackWhenEmptyAfterCleanup(t *testing.T) {
	input := string([]byte{0xff, 0xfe, ' '})

	got := sanitizeProcessString(input, "[unknown]")

	if got != "[unknown]" {
		t.Fatalf("sanitizeProcessString() = %q, want %q", got, "[unknown]")
	}
}

func TestParseJavaCommandWithJar(t *testing.T) {
	args := []string{
		"/usr/bin/java",
		"-Xms256m",
		"-javaagent:/tmp/agent.jar",
		"-jar",
		"/srv/app/order-service.jar",
		"--server.port=8080",
	}

	jvmArgs, mainClass, jarPath := parseJavaCommand(args)

	if mainClass != "" {
		t.Fatalf("mainClass = %q, want empty", mainClass)
	}
	if jarPath != "/srv/app/order-service.jar" {
		t.Fatalf("jarPath = %q, want %q", jarPath, "/srv/app/order-service.jar")
	}
	if len(jvmArgs) != 2 || jvmArgs[0] != "-Xms256m" || jvmArgs[1] != "-javaagent:/tmp/agent.jar" {
		t.Fatalf("unexpected jvmArgs: %#v", jvmArgs)
	}
}

func TestParseJavaCommandWithClasspathMainClass(t *testing.T) {
	args := []string{
		"/usr/bin/java",
		"-cp",
		"/srv/app/classes:/srv/app/libs/*",
		"-Dspring.profiles.active=prod",
		"com.demo.OrderApplication",
		"--server.port=8080",
	}

	jvmArgs, mainClass, jarPath := parseJavaCommand(args)

	if jarPath != "" {
		t.Fatalf("jarPath = %q, want empty", jarPath)
	}
	if mainClass != "com.demo.OrderApplication" {
		t.Fatalf("mainClass = %q, want %q", mainClass, "com.demo.OrderApplication")
	}
	if len(jvmArgs) != 3 {
		t.Fatalf("len(jvmArgs) = %d, want 3", len(jvmArgs))
	}
}
