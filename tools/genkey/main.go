package main

import (
	"fmt"
	"log"

	hubcrypto "github.com/langgexyz/open-im-hub-server/internal/crypto"
)

func main() {
	privKey, pubKey, err := hubcrypto.GenerateKey()
	if err != nil {
		log.Fatalf("generate key: %v", err)
	}
	fmt.Printf("HUB_PRIVATE_KEY=%s\n", hubcrypto.PrivKeyToHex(privKey))
	fmt.Printf("HUB_PUBLIC_KEY=%s  (硬编码进 App 客户端)\n", pubKey)
}
