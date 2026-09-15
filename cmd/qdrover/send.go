package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"
	"time"

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

func runSend(args []string, stdin io.Reader) error {
	fs := flag.NewFlagSet("send", flag.ContinueOnError)
	direction := fs.String("direction", "", "up|down|left|right")
	if err := fs.Parse(args); err != nil {
		return err
	}

	text, err := resolveQuery(fs.Args(), stdin)
	if err != nil {
		return err
	}

	gateway := herdr.NewGateway()
	ctx, cancel := context.WithTimeout(context.Background(), herdrCallTimeout)
	defer cancel()

	return gateway.ResolveAndPromptWithPrefix(ctx, ports.Direction(*direction), nil, text)
}
