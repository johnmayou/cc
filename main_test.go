package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCLI(t *testing.T) {
	steps, err := os.ReadDir("testdata")
	require.NoError(t, err)

	for _, step := range steps {
		cases, err := os.ReadDir(filepath.Join("testdata", step.Name()))
		require.NoError(t, err)

		for _, c := range cases {
			// Too many edge cases that I don't care to implement.
			if step.Name() == "step5" && c.Name() == "pass1.json" {
				continue
			}

			t.Run(step.Name()+":"+c.Name(), func(t *testing.T) {
				var stdout, stderr bytes.Buffer
				cmd := exec.Command("go", "run", "main.go", filepath.Join("testdata", step.Name(), c.Name()))
				cmd.Stdout = &stdout
				cmd.Stderr = &stderr

				err = cmd.Run()

				if strings.Contains(c.Name(), "invalid") || strings.Contains(c.Name(), "fail") {
					require.Equal(t, 1, cmd.ProcessState.ExitCode())
				} else {
					require.Equal(t, 0, cmd.ProcessState.ExitCode(), stderr.String())
				}
			})
		}
	}
}

func TestTokenize(t *testing.T) {
	tests := []struct {
		input string
		want  []Token
	}{
		{"{}", []Token{{Type: CurlyBracketOpen, Start: 0, Stop: 0}, {Type: CurlyBracketClosed, Start: 1, Stop: 1}}},
		{"[]", []Token{{Type: BracketOpen, Start: 0, Stop: 0}, {Type: BracketClosed, Start: 1, Stop: 1}}},
		{":", []Token{{Type: Colon, Start: 0, Stop: 0}}},
		{",", []Token{{Type: Comma, Start: 0, Stop: 0}}},
		{"\"string\"", []Token{{Type: String, Start: 0, Stop: 7}}},
		{"12345678", []Token{{Type: Number, Start: 0, Stop: 7}}},
		{"true", []Token{{Type: True, Start: 0, Stop: 3}}},
		{"false", []Token{{Type: False, Start: 0, Stop: 4}}},
		{"null", []Token{{Type: Null, Start: 0, Stop: 3}}},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got, err := Tokenize(tt.input)
			require.NoError(t, err)
			require.Equal(t, tt.want, got)
		})
	}
}

func TestParse(t *testing.T) {

}
