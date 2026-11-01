package eventstore

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"testing"
	"time"

	"github.com/go-jose/go-jose/v4"
	"github.com/stretchr/testify/require"
	"github.com/zitadel/oidc/v3/pkg/oidc"
	"github.com/zitadel/oidc/v3/pkg/op"

	http_util "github.com/zitadel/zitadel/internal/api/http"
)

type verifierTestKey struct{ public ed25519.PublicKey }

func (key verifierTestKey) VerifySignature(_ context.Context, signature *jose.JSONWebSignature) ([]byte, error) {
	return signature.Verify(key.public)
}

func TestJWTTokenVerifierEd25519(t *testing.T) {
	public, private, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	_, otherPrivate, err := ed25519.GenerateKey(rand.Reader)
	require.NoError(t, err)
	ctx := http_util.WithDomainContext(context.Background(), http_util.NewDomainCtx("id.example", "id.example", "https"))
	verifier := (&TokenVerifierRepo{}).jwtTokenVerifier(ctx)
	verifier.KeySet = verifierTestKey{public}
	for _, test := range []struct {
		name    string
		issuer  string
		expires time.Time
		key     ed25519.PrivateKey
		valid   bool
	}{
		{"valid", "https://id.example", time.Now().Add(time.Minute), private, true},
		{"wrong issuer", "https://other.example", time.Now().Add(time.Minute), private, false},
		{"expired", "https://id.example", time.Now().Add(-time.Minute), private, false},
		{"wrong key", "https://id.example", time.Now().Add(time.Minute), otherPrivate, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.EdDSA, Key: test.key}, nil)
			require.NoError(t, err)
			payload, err := json.Marshal(map[string]any{"iss": test.issuer, "sub": "service", "jti": "token", "aud": []string{"api"}, "iat": time.Now().Unix(), "exp": test.expires.Unix()})
			require.NoError(t, err)
			signed, err := signer.Sign(payload)
			require.NoError(t, err)
			token, err := signed.CompactSerialize()
			require.NoError(t, err)
			claims, err := op.VerifyAccessToken[*oidc.AccessTokenClaims](ctx, token, verifier)
			if test.valid {
				require.NoError(t, err)
				require.Equal(t, "service", claims.Subject)
			} else {
				require.Error(t, err)
			}
		})
	}
}
