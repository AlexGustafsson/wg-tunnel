package main

import (
	"encoding/base64"
	"encoding/hex"
	"flag"
	"io"
	"os"
	"strings"
)

var _ flag.Value = (*StringSlice)(nil)

type StringSlice []string

// Set implements flag.Value.
func (s *StringSlice) Set(v string) error {
	*s = append(*s, v)
	return nil
}

// String implements flag.Value.
func (s *StringSlice) String() string {
	return strings.Join(*s, ",")
}

func readKey(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()

	contents, err := io.ReadAll(base64.NewDecoder(base64.StdEncoding, file))
	if err != nil {
		return "", err
	}

	return hex.EncodeToString(contents), nil
}
