package cmd

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/spf13/cobra"
)

const (
	KiB = 1024
	MiB = 1024 * KiB
)

var (
	bytes bool
	lines bool
	chars bool
	words bool
)

var rootCmd = &cobra.Command{
	Use:   "wc",
	Short: "word, line, character, and byte count",
	RunE: func(cmd *cobra.Command, args []string) error {
		if !(bytes || lines || chars || words) {
			bytes = true
			lines = true
			words = true
		}

		var err error
		var file *os.File
		var filename string
		if isPipedInput() {
			file = os.Stdin
		} else if len(args) == 1 {
			file, err = os.Open(args[0])
			if err != nil {
				return err
			}
			filename = args[0]
		} else {
			return errors.New("no input was found")
		}

		var nbytes, nlines, nchars, nwords int
		var inword bool

		buf := make([]byte, 8*KiB)
		for {
			n, err := file.Read(buf)
			if err == io.EOF {
				break
			} else if err != nil {
				return err
			}
			nbytes += n

			chunk := buf[:n]
			nchars += utf8.RuneCount(chunk)

			for _, b := range chunk {
				if unicode.IsSpace(rune(b)) {
					inword = false
				} else if !inword {
					nwords++
					inword = true
				}

				if b == '\n' {
					nlines++
				}
			}
		}

		var parts []string
		if lines {
			parts = append(parts, fmt.Sprintf("%7d", nlines))
		}
		if words {
			parts = append(parts, fmt.Sprintf("%7d", nwords))
		}
		if bytes {
			parts = append(parts, fmt.Sprintf("%7d", nbytes))
		}
		if chars {
			parts = append(parts, fmt.Sprintf("%7d", nchars))
		}
		if filename != "" {
			parts = append(parts, filename)
		}

		_, err = fmt.Println(" " + strings.Join(parts, " "))
		return err
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().BoolVarP(&bytes, "bytes", "c", false, "Number of bytes.")
	rootCmd.Flags().BoolVarP(&lines, "lines", "l", false, "Number of lines.")
	rootCmd.Flags().BoolVarP(&chars, "chars", "m", false, "Number of chars.")
	rootCmd.Flags().BoolVarP(&words, "words", "w", false, "Number of words.")
}

func isPipedInput() bool {
	stat, _ := os.Stdin.Stat()
	return (stat.Mode() & os.ModeCharDevice) == 0
}
