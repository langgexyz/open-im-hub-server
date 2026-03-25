package main

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	hubcrypto "github.com/langgexyz/open-im-hub-server/internal/crypto"
)

func main() {
	privKey, err := hubcrypto.PrivKeyFromHex("a57f16a73b624b06325a070bb149e996cd8f431a988a12cc10f914003c7de95e")
	if err != nil {
		panic(err)
	}
	payload, _ := json.Marshal(map[string]any{
		"app_uid": "user_test_001",
		"exp":     time.Now().Add(time.Hour).Unix(),
	})
	b64 := base64.RawURLEncoding.EncodeToString(payload)
	sig, err := hubcrypto.Sign([]byte(b64), privKey)
	if err != nil {
		panic(err)
	}
	fmt.Println(b64 + "." + hex.EncodeToString(sig))
}
