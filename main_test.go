package main

import (
	"fmt"
	"os/exec"
	"strings"
	"testing"
)

func TestCLI(t *testing.T) {
	tests := []struct {
		cmd  string
		want string
	}{
		{"go run main.go -c test.txt", "  342190 test.txt"},
		{"go run main.go -l test.txt", "    7145 test.txt"},
		{"go run main.go -w test.txt", "   58164 test.txt"},
		{"go run main.go -m test.txt", "  339292 test.txt"},
		{"go run main.go test.txt", "    7145   58164  342190 test.txt"},
		{"cat test.txt | go run main.go -l", "    7145"},
	}

	for num, tt := range tests {
		t.Run(fmt.Sprintf("CLI test #%d", num), func(t *testing.T) {
			out, err := exec.Command("sh", "-c", tt.cmd).Output()
			if err != nil {
				t.Fatalf("command failed: %v", err)
			}
			got := strings.TrimRight(string(out), "\n")
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}
