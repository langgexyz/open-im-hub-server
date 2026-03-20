package handler

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"time"

	hubcrypto "github.com/langgexyz/open-im-hub-server/internal/crypto"
)

// VerifyCredential 验证 Hub Server 签发的 user_credential，返回 app_uid
func VerifyCredential(tokenStr, hubPublicKey string) (string, error) {
	parts := strings.SplitN(tokenStr, ".", 2)
	if len(parts) != 2 {
		return "", errors.New("malformed credential")
	}
	payloadB64, sigHex := parts[0], parts[1]
	payloadBytes, err := base64.RawURLEncoding.DecodeString(payloadB64)
	if err != nil {
		return "", errors.New("invalid payload encoding")
	}
	var payload struct {
		AppUID string `json:"app_uid"`
		Exp    int64  `json:"exp"`
	}
	if err := json.Unmarshal(payloadBytes, &payload); err != nil {
		return "", errors.New("invalid payload json")
	}
	if time.Now().Unix() > payload.Exp {
		return "", errors.New("credential expired")
	}
	sig, err := hex.DecodeString(sigHex)
	if err != nil || len(sig) != 65 {
		return "", errors.New("invalid signature format")
	}
	recovered, err := hubcrypto.Ecrecover([]byte(payloadB64), sig)
	if err != nil || !strings.EqualFold(recovered, hubPublicKey) {
		return "", errors.New("signature verification failed")
	}
	return payload.AppUID, nil
}
