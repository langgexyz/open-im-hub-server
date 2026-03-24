package auth

import (
	"crypto/ecdsa"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	hubcrypto "github.com/langgexyz/open-im-hub-server/internal/crypto"
)

type credentialPayload struct {
	UID   string `json:"uid"`
	AppID string `json:"app_id"`
	Exp   int64  `json:"exp"`
}

// IssueCredential signs a credential binding uid + app_id + exp using hub_private_key.
// Format: base64url(payload) + "." + hex(sign(keccak256(base64url(payload)), hub_private_key))
func IssueCredential(uid, appID string, privKey *ecdsa.PrivateKey, ttlSeconds int64) (string, error) {
	payload := credentialPayload{
		UID:   uid,
		AppID: appID,
		Exp:   time.Now().Add(time.Duration(ttlSeconds) * time.Second).Unix(),
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	payloadB64 := base64.RawURLEncoding.EncodeToString(payloadJSON)
	sig, err := hubcrypto.Sign([]byte(payloadB64), privKey)
	if err != nil {
		return "", fmt.Errorf("sign credential: %w", err)
	}
	return payloadB64 + "." + hex.EncodeToString(sig), nil
}

// PubKeyFromHex derives the Ethereum address (hub public key) from a hub_private_key hex string.
// Useful for tests to obtain the expected public key from a known private key.
func PubKeyFromHex(privHex string) (string, error) {
	return hubcrypto.PubKeyFromPrivHex(privHex)
}

// VerifyCredential verifies the credential signature and expiry, returning uid and app_id.
// hubPublicKey is the Ethereum address (e.g. "0xabcd..."), case-insensitive.
func VerifyCredential(credStr, hubPublicKey string) (uid, appID string, err error) {
	parts := strings.SplitN(credStr, ".", 2)
	if len(parts) != 2 {
		return "", "", fmt.Errorf("malformed credential")
	}
	payloadB64, sigHex := parts[0], parts[1]

	payloadBytes, err := base64.RawURLEncoding.DecodeString(payloadB64)
	if err != nil {
		return "", "", fmt.Errorf("invalid payload encoding")
	}
	var p credentialPayload
	if err := json.Unmarshal(payloadBytes, &p); err != nil {
		return "", "", fmt.Errorf("invalid payload json")
	}
	if time.Now().Unix() > p.Exp {
		return "", "", fmt.Errorf("credential expired")
	}
	sig, err := hex.DecodeString(sigHex)
	if err != nil || len(sig) != 65 {
		return "", "", fmt.Errorf("invalid signature format")
	}
	recovered, err := hubcrypto.Ecrecover([]byte(payloadB64), sig)
	if err != nil || !strings.EqualFold(recovered, hubPublicKey) {
		return "", "", fmt.Errorf("signature verification failed")
	}
	return p.UID, p.AppID, nil
}
