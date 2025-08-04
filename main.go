package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/spf13/cobra"
)

func fetchCommand() *cobra.Command {
	var opts PullOptions
	var auth string

	var cmd = &cobra.Command{
		Use: "ctr-fetch <image> <path>",

		Args: cobra.ExactArgs(2),

		RunE: func(cmd *cobra.Command, args []string) error {
			start := time.Now()
			if auth != "" {
				username, password, err := parseAuth(auth)
				if err != nil {
					return err
				}
				opts.Username = username
				opts.Password = password
			}
			opts.Stdout = cmd.OutOrStdout()

			imageName, destPath := args[0], args[1]
			err := ensureDir(destPath)
			if err != nil {
				return fmt.Errorf("ensure dest directory: %w", err)
			}

			pullResult, err := PullImage(imageName, opts)
			if err != nil {
				return fmt.Errorf("pull image error: %w", err)
			}

			size, err := ExtractDirectory(pullResult, destPath)
			if err != nil {
				return fmt.Errorf("extract layers error: %v", err)
			}

			sizeStr := humanize.Bytes(size)
			fmt.Println()
			fmt.Printf("Fetch done, extracted %s data, took %v\n", sizeStr, time.Since(start))
			return nil
		},
	}

	flags := cmd.Flags()

	flags.StringVarP(&auth, "auth", "", "", "The auth string, format is 'user:password'")
	flags.StringVarP(&opts.Token, "token", "", "", "Pull registry token")
	flags.BoolVarP(&opts.Insecure, "insecure", "", false, "Skip TLS verification while pulling")
	flags.StringVarP(&opts.BaseDir, "base-dir", "b", "", "The base directory to store downloaded images, default is /tmp/ctr-fetch")
	flags.BoolVarP(&opts.Force, "force", "f", false, "Force re-download the image even if it is already present in the base directory")

	return cmd
}

func parseAuth(auth string) (string, string, error) {
	fields := strings.Split(auth, ":")
	if len(fields) <= 1 {
		return "", "", fmt.Errorf("invalid auth %q, should be 'user:password'", auth)
	}
	user := fields[0]
	password := strings.Join(fields[1:], ":")
	return user, password, nil
}

func main() {
	cmd := fetchCommand()

	cmd.CompletionOptions = cobra.CompletionOptions{
		HiddenDefaultCmd: true,
	}
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true

	err := cmd.Execute()
	if err != nil {
		fmt.Printf("error: %v\n", err)
		os.Exit(1)
	}
}
