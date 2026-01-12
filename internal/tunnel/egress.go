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
	"gvisor.dev/gvisor/pkg/tcpip/adapters/gonet"
)

const MTU = 1420

type Egress struct {
	tun       tun.Device
	network   *netstack.Net
	device    *device.Device
	forwards  map[uint16]string
	listeners map[uint16]*gonet.TCPListener
	wg        sync.WaitGroup
}

func NewEgress(privateKey string, server string, serverPublicKey string) (*Egress, error) {
	address := netip.MustParseAddr("10.0.8.2")

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
		"private_key=%s\npublic_key=%s\nallowed_ip=%s/32\nendpoint=%s\npersistent_keepalive_interval=25",
		privateKey,
		serverPublicKey,
		"10.0.8.1",
		server,
	)

	err = device.IpcSet(uapi)
	if err != nil {
		return nil, err
	}

	return &Egress{
		tun:       tun,
		network:   network,
		device:    device,
		forwards:  make(map[uint16]string),
		listeners: make(map[uint16]*gonet.TCPListener),
	}, nil
}

func (e *Egress) AddForward(port uint16, addr string) {
	e.forwards[port] = addr
}

func (e *Egress) Listen() error {
	err := e.device.Up()
	if err != nil {
		return err
	}

	for k := range e.forwards {
		listener, err := e.network.ListenTCPAddrPort(netip.MustParseAddrPort(fmt.Sprintf("10.0.8.2:%d", k)))
		if err != nil {
			return err
		}

		e.listeners[k] = listener
	}

	return nil
}

func (e *Egress) Serve() {
	for port, listener := range e.listeners {
		e.wg.Go(func() {
			log := slog.With(slog.Int("port", int(port)))
			defer listener.Close()

			for {
				conn, err := listener.Accept()
				if err != nil {
					log.Error("Failed to accept connection, closing")
					return
				}

				e.wg.Go(func() {
					e.serve(port, conn)
				})
			}
		})
	}

	e.wg.Wait()
}

func (e *Egress) serve(port uint16, conn net.Conn) {
	defer conn.Close()

	upstreamConn, err := net.Dial("tcp", e.forwards[port])
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

func (e *Egress) ListenAndServe() error {
	if err := e.Listen(); err != nil {
		return err
	}

	e.Serve()

	return nil
}

func (e *Egress) Shutdown() {
	for _, listener := range e.listeners {
		listener.Shutdown()
	}
	e.wg.Wait()
}

func (e *Egress) Close() error {
	errs := make([]error, 0, len(e.listeners))
	for _, listener := range e.listeners {
		errs = append(errs, listener.Close())
	}
	return errors.Join(errs...)
}
