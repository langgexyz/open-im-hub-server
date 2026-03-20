package crypto

import (
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/crypto"
)

func GenerateKey() (*ecdsa.PrivateKey, string, error) {
	privKey, err := crypto.GenerateKey()
	if err != nil {
		return nil, "", err
	}
	addr, err := PrivKeyToAddress(privKey)
	if err != nil {
		return nil, "", err
	}
	return privKey, addr, nil
}

func PrivKeyToAddress(privKey *ecdsa.PrivateKey) (string, error) {
	pubKey, ok := privKey.Public().(*ecdsa.PublicKey)
	if !ok {
		return "", fmt.Errorf("invalid public key type")
	}
	return strings.ToLower(crypto.PubkeyToAddress(*pubKey).Hex()), nil
}

func Keccak256(data ...[]byte) []byte {
	return crypto.Keccak256(data...)
}

func Sign(message []byte, privKey *ecdsa.PrivateKey) ([]byte, error) {
	return crypto.Sign(Keccak256(message), privKey)
}

func Ecrecover(message, sig []byte) (string, error) {
	if len(sig) != 65 {
		return "", fmt.Errorf("invalid signature length: %d", len(sig))
	}
	pubKeyBytes, err := crypto.Ecrecover(Keccak256(message), sig)
	if err != nil {
		return "", err
	}
	pubKey, err := crypto.UnmarshalPubkey(pubKeyBytes)
	if err != nil {
		return "", err
	}
	return strings.ToLower(crypto.PubkeyToAddress(*pubKey).Hex()), nil
}

func PrivKeyToHex(privKey *ecdsa.PrivateKey) string {
	return hex.EncodeToString(crypto.FromECDSA(privKey))
}

func PrivKeyFromHex(hexKey string) (*ecdsa.PrivateKey, error) {
	return crypto.HexToECDSA(strings.TrimPrefix(hexKey, "0x"))
}

// PubKeyFromPrivHex 从私钥 hex 字符串返回对应的以太坊地址（公钥地址 hex）
func PubKeyFromPrivHex(privHex string) (string, error) {
	priv, err := PrivKeyFromHex(privHex)
	if err != nil {
		return "", err
	}
	addr, err := PrivKeyToAddress(priv)
	if err != nil {
		return "", err
	}
	return addr, nil
}
