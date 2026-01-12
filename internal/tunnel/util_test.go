package tunnel

import (
	"fmt"
	"testing"
)

func TestGenerateKey(t *testing.T) {
	privateKey, publicKey, err := GenerateKey()
	if err != nil {
		t.Fatalf("failed to generate keys: %v", err)
	}

	fmt.Println(privateKey)
	fmt.Println(publicKey)
}
