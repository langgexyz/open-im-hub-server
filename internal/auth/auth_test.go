package auth_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/langgexyz/open-im-hub-server/internal/auth"
	hubcrypto "github.com/langgexyz/open-im-hub-server/internal/crypto"
)

const testPrivHex = "ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80" // 已知测试私钥

func TestHubToken(t *testing.T) {
	secret := "testsecret"
	token, err := auth.IssueHubToken("10001", "alice@example.com", secret, 3600)
	require.NoError(t, err)
	require.NotEmpty(t, token)

	claims, err := auth.VerifyHubToken(token, secret)
	require.NoError(t, err)
	require.Equal(t, "10001", claims.UID)
	require.Equal(t, "alice@example.com", claims.Email)
}

func TestHubTokenExpired(t *testing.T) {
	secret := "testsecret"
	token, _ := auth.IssueHubToken("1", "a@b.com", secret, -1) // 过期
	_, err := auth.VerifyHubToken(token, secret)
	require.Error(t, err)
}

func TestCredential(t *testing.T) {
	priv, err := hubcrypto.PrivKeyFromHex(testPrivHex)
	require.NoError(t, err)
	pub, _ := hubcrypto.PubKeyFromPrivHex(testPrivHex)

	cred, err := auth.IssueCredential("10001", "app-abc123", priv, 3600)
	require.NoError(t, err)

	uid, appID, err := auth.VerifyCredential(cred, pub)
	require.NoError(t, err)
	require.Equal(t, "10001", uid)
	require.Equal(t, "app-abc123", appID)
}
