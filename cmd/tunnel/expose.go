package main

import (
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/AlexGustafsson/wg-tunnel/internal/tunnel"
)

func expose(privateKey string, peerPublicKey string, exposed []string, address string) {
	egress, err := tunnel.NewEgress(privateKey, address, peerPublicKey)
	if err != nil {
		slog.Error("Failed to create egress", slog.Any("error", err))
		os.Exit(1)
	}

	for _, exposed := range exposed {
		parts := strings.Split(exposed, ":")
		if len(parts) != 3 {
			slog.Error("Invalid expose syntax")
			os.Exit(1)
		}
		localPort := parts[0]
		localAddress := parts[1]
		peerPort, err := strconv.ParseUint(parts[2], 10, 16)
		if err != nil {
			slog.Error("Invalid expose syntax", slog.Any("error", err))
			os.Exit(1)
		}

		egress.AddForward(uint16(peerPort), localAddress+":"+localPort)
	}

	err = egress.Listen()
	if err != nil {
		slog.Error("Failed to listen", slog.Any("error", err))
		return
	}

	done := make(chan struct{})
	go func() {
		defer close(done)
		egress.Serve()
	}()

	signals := make(chan os.Signal, 2)
	signal.Notify(signals, syscall.SIGTERM, syscall.SIGINT)

	caught := 0
loop:
	for {
		select {
		case <-done:
			break loop
		case <-signals:
			caught++
			if caught == 1 {
				slog.Info("Caught signal, exiting gracefully")
				egress.Close()
			} else {
				slog.Info("Caught signal, exiting now")
				os.Exit(1)
			}
		}
	}
}
