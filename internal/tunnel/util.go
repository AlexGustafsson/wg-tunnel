package tunnel

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net"

	"golang.org/x/crypto/curve25519"
	"golang.zx2c4.com/wireguard/device"
	"gvisor.dev/gvisor/pkg/tcpip/adapters/gonet"
)

var defaultLogger = &device.Logger{Verbosef: verbosef, Errorf: errorf}

func verbosef(format string, args ...any) {
	slog.Debug(fmt.Sprintf(format, args...))
}

func errorf(format string, args ...any) {
	slog.Error(fmt.Sprintf(format, args...))
}

func GenerateKey() (string, string, error) {
	var privateKey device.NoisePrivateKey
	_, err := rand.Read(privateKey[:])
	if err != nil {
		return "", "", err
	}

	var publicKey device.NoisePublicKey

	apk := (*[device.NoisePublicKeySize]byte)(&publicKey)
	ask := (*[device.NoisePrivateKeySize]byte)(&privateKey)
	curve25519.ScalarBaseMult(apk, ask)

	return hex.EncodeToString(publicKey[:]), hex.EncodeToString(privateKey[:]), nil
}

func closeWrite(conn net.Conn) error {
	switch c := conn.(type) {
	case *net.TCPConn:
		return c.CloseWrite()
	case *gonet.TCPConn:
		return c.CloseWrite()
	}

	return nil
}
