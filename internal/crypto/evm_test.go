package crypto_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	hubcrypto "github.com/langgexyz/open-im-hub-server/internal/crypto"
)

func TestSignAndRecover(t *testing.T) {
	privKey, addr, err := hubcrypto.GenerateKey()
	require.NoError(t, err)
	require.NotEmpty(t, addr)

	msg := []byte("hello hub")
	sig, err := hubcrypto.Sign(msg, privKey)
	require.NoError(t, err)
	require.Len(t, sig, 65)

	recovered, err := hubcrypto.Ecrecover(msg, sig)
	require.NoError(t, err)
	require.Equal(t, addr, recovered)
}

func TestKeccak256Separator(t *testing.T) {
	h1 := hubcrypto.Keccak256([]byte("ab"))
	h2 := hubcrypto.Keccak256([]byte("a"), []byte{0x00}, []byte("b"))
	require.NotEqual(t, h1, h2)
}

func TestPrivKeyRoundtrip(t *testing.T) {
	privKey, addr, _ := hubcrypto.GenerateKey()
	hexKey := hubcrypto.PrivKeyToHex(privKey)
	restored, err := hubcrypto.PrivKeyFromHex(hexKey)
	require.NoError(t, err)
	restoredAddr, err := hubcrypto.PrivKeyToAddress(restored)
	require.NoError(t, err)
	require.Equal(t, addr, restoredAddr)
}
