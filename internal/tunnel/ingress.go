package tunnel

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/netip"
	"sync"

	"golang.zx2c4.com/wireguard/conn"
	"golang.zx2c4.com/wireguard/device"
	"golang.zx2c4.com/wireguard/tun"
	"golang.zx2c4.com/wireguard/tun/netstack"
)

type Ingress struct {
	tun       tun.Device
	network   *netstack.Net
	device    *device.Device
	forwards  map[string]uint16
	listeners map[net.Listener]uint16
	wg        sync.WaitGroup
}

func NewIngress(privateKey string, peerPublicKey string, listenPort uint16) (*Ingress, error) {
	address := netip.MustParseAddr("10.0.8.1")

	tun, network, err := netstack.CreateNetTUN(
		[]netip.Addr{address},
		[]netip.Addr{},
		MTU,
	)
	if err != nil {
		return nil, err
	}

	device := device.NewDevice(tun, conn.NewDefaultBind(), defaultLogger)

	uapi := fmt.Sprintf(
		"private_key=%s\nlisten_port=%d\npublic_key=%s\nallowed_ip=%s/32",
		privateKey,
		listenPort,
		peerPublicKey,
		"10.0.8.2",
	)

	err = device.IpcSet(uapi)
	if err != nil {
		return nil, err
	}

	return &Ingress{
		tun:       tun,
		network:   network,
		device:    device,
		forwards:  make(map[string]uint16),
		listeners: make(map[net.Listener]uint16),
	}, nil
}

func (i *Ingress) AddForward(addr string, port uint16) {
	i.forwards[addr] = port
}

func (i *Ingress) Listen() error {
	err := i.device.Up()
	if err != nil {
		return err
	}

	for addr, port := range i.forwards {
		listener, err := net.Listen("tcp", addr)
		if err != nil {
			return err
		}
		i.listeners[listener] = port
	}

	return nil
}

func (i *Ingress) Serve() {
	for listener, port := range i.listeners {
		i.wg.Go(func() {
			log := slog.With(slog.Int("port", int(port)))
			defer listener.Close()

			for {
				conn, err := listener.Accept()
				if err != nil {
					log.Error("Failed to accept connection, closing")
					return
				}

				i.wg.Go(func() {
					i.serve(port, conn)
				})
			}
		})
	}

	i.wg.Wait()
}

func (i *Ingress) serve(port uint16, conn net.Conn) {
	defer conn.Close()

	_ = conn.(*net.TCPConn).SetLinger(0)

	upstreamConn, err := i.network.Dial("tcp", fmt.Sprintf("10.0.8.2:%d", port))
	if err != nil {
		slog.Error("Failed to dial upstream", slog.Any("error", err))
		return
	}
	defer upstreamConn.Close()

	var wg sync.WaitGroup

	var readError error
	wg.Go(func() {
		_, readError = io.Copy(conn, upstreamConn)
		if readError != nil {
			conn.Close()
			upstreamConn.Close()
		}
	})

	var writeError error
	wg.Go(func() {
		_, writeError = io.Copy(upstreamConn, conn)
		if writeError != nil {
			conn.Close()
			upstreamConn.Close()
		}
	})

	wg.Wait()

	if readError != nil || writeError != nil {
		slog.Warn("Failed to serve connection", slog.Any("error", errors.Join(readError, writeError)))
	}
}

func (i *Ingress) ListenAndServe() error {
	if err := i.Listen(); err != nil {
		return err
	}

	i.Serve()

	return nil
}

func (i *Ingress) Close() error {
	errs := make([]error, 0, len(i.listeners))
	for listener := range i.listeners {
		errs = append(errs, listener.Close())
	}
	return errors.Join(errs...)
}
