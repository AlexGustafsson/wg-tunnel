package main

import (
	"flag"
	"log/slog"
)

func main() {
	slog.SetLogLoggerLevel(slog.LevelDebug)

	privateKeyPath := flag.String("private-key", "", "")
	peerPublicKeyPath := flag.String("peer-public-key", "", "")

	listenPort := flag.Uint("l", 0, "")

	var exposed []string
	flag.Var((*StringSlice)(&exposed), "expose", "")

	flag.Parse()

	privateKey, err := readKey(*privateKeyPath)
	if err != nil {
		panic(err)
	}

	peerPublicKey, err := readKey(*peerPublicKeyPath)
	if err != nil {
		panic(err)
	}

	if *listenPort > 0 {
		listen(privateKey, peerPublicKey, exposed, uint16(*listenPort))
	} else {
		expose(privateKey, peerPublicKey, exposed, flag.Arg(0))
	}
}
