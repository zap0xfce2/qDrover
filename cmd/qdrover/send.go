package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"qdrover/internal/adapters/herdr"
	"qdrover/internal/ports"
)

const herdrCallTimeout = 5 * time.Second

func resolveQuery(args []string, stdin io.Reader) (string, error) {
	if len(args) > 0 {
		return strings.Join(args, " "), nil
	}
	data, err := io.ReadAll(stdin)
	if err != nil {
		return "", fmt.Errorf("stdin lesen: %w", err)
	}
	text := strings.TrimSpace(string(data))
	if text == "" {
		return "", errors.New("keine Query übergeben (Argument oder stdin)")
	}
	return text, nil
}

func newSendCmd() *cobra.Command {
	var direction string

	cmd := &cobra.Command{
		Use:   "send [query]",
		Short: "Sendet einen Dispatch an eine Herdr-Pane",
		RunE: func(cmd *cobra.Command, args []string) error {
			text, err := resolveQuery(args, cmd.InOrStdin())
			if err != nil {
				return err
			}

			gateway := herdr.NewGateway()
			ctx, cancel := context.WithTimeout(context.Background(), herdrCallTimeout)
			defer cancel()

			return gateway.ResolveAndPromptWithPrefix(ctx, ports.Direction(direction), nil, text)
		},
	}
	cmd.Flags().StringVar(&direction, "direction", "", "up|down|left|right")
	return cmd
}
