package command

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"golang.org/x/text/language"

	"github.com/zitadel/zitadel/internal/domain"
	"github.com/zitadel/zitadel/internal/eventstore"
	"github.com/zitadel/zitadel/internal/repository/session"
	"github.com/zitadel/zitadel/internal/repository/user"
	webauthnconfig "github.com/zitadel/zitadel/internal/webauthn"
)

func TestSessionDiscoverablePasskey(t *testing.T) {
	for _, test := range []struct {
		name      string
		handle    string
		challenge string
		flags     byte
		boundUser string
		wantError bool
	}{
		{name: "discover and verify", handle: "user1", flags: 5},
		{name: "wrong challenge", handle: "user1", challenge: "wrong", flags: 5, wantError: true},
		{name: "missing user verification", handle: "user1", flags: 1, wantError: true},
		{name: "another user's handle", handle: "user2", flags: 5, wantError: true},
		{name: "cannot switch a bound user", handle: "user2", boundUser: "user1", flags: 5, wantError: true},
		{name: "empty handle", flags: 5, wantError: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx := context.Background()
			key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
			require.NoError(t, err)
			// COSE EC2 / ES256 public key, with 32-byte P-256 coordinates.
			publicKey := append([]byte{0xa5, 1, 2, 3, 0x26, 0x20, 1, 0x21, 0x58, 32}, key.X.FillBytes(make([]byte, 32))...)
			publicKey = append(publicKey, 0x22, 0x58, 32)
			publicKey = append(publicKey, key.Y.FillBytes(make([]byte, 32))...)
			credentialID := []byte("credential1")
			provider := &Commands{webauthnConfig: &webauthnconfig.Config{DisplayName: "Test"}}
			wm := NewSessionWriteModel("session1", "instance1")
			cmd := &SessionCommands{sessionWriteModel: wm}
			var options json.RawMessage
			_, err = provider.CreateWebAuthNChallenge(domain.UserVerificationRequirementRequired, "localhost", &options)(ctx, cmd)
			require.NoError(t, err)
			require.Len(t, cmd.eventCommands, 1)
			require.NoError(t, AppendAndReduce(wm, cmd.eventCommands[0].(*session.WebAuthNChallengedEvent)))
			require.NotNil(t, wm.WebAuthNChallenge)
			var request struct {
				PublicKey struct {
					Challenge        string `json:"challenge"`
					AllowCredentials []any  `json:"allowCredentials"`
					UserVerification string `json:"userVerification"`
				} `json:"publicKey"`
			}
			require.NoError(t, json.Unmarshal(options, &request))
			require.Empty(t, request.PublicKey.AllowCredentials)
			require.Equal(t, "required", request.PublicKey.UserVerification)
			challenge := request.PublicKey.Challenge
			if test.challenge != "" {
				challenge = test.challenge
			}
			clientData, err := json.Marshal(map[string]any{"type": "webauthn.get", "challenge": challenge, "origin": "https://localhost"})
			require.NoError(t, err)
			rpHash := sha256.Sum256([]byte("localhost"))
			authData := append(rpHash[:], test.flags, 0, 0, 0, 0)
			binary.BigEndian.PutUint32(authData[33:], 1)
			clientHash := sha256.Sum256(clientData)
			signed := sha256.Sum256(append(authData, clientHash[:]...))
			signature, err := ecdsa.SignASN1(rand.Reader, key, signed[:])
			require.NoError(t, err)
			encode := base64.RawURLEncoding.EncodeToString
			assertion, err := json.Marshal(map[string]any{
				"id": encode(credentialID), "rawId": encode(credentialID), "type": "public-key",
				"response": map[string]any{"clientDataJSON": encode(clientData), "authenticatorData": encode(authData), "signature": encode(signature), "userHandle": encode([]byte(test.handle))},
			})
			require.NoError(t, err)
			wm.UserID = test.boundUser
			var expected []expect
			if test.handle != "" || test.boundUser != "" {
				ownerID := "user1"
				if test.boundUser == "" && test.handle == "user2" {
					ownerID = "user2"
					otherKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
					require.NoError(t, err)
					publicKey = append([]byte{0xa5, 1, 2, 3, 0x26, 0x20, 1, 0x21, 0x58, 32}, otherKey.X.FillBytes(make([]byte, 32))...)
					publicKey = append(publicKey, 0x22, 0x58, 32)
					publicKey = append(publicKey, otherKey.Y.FillBytes(make([]byte, 32))...)
				}
				aggregate := &user.NewAggregate(ownerID, "org1").Aggregate
				expected = []expect{
					expectFilter(eventFromEventPusher(user.NewHumanAddedEvent(ctx, aggregate, "", "", "", "", "", language.English, domain.GenderUnspecified, "", true))),
					expectFilter(
						eventFromEventPusher(user.NewHumanPasswordlessAddedEvent(ctx, aggregate, "token1", "registration", "localhost")),
						eventFromEventPusher(user.NewHumanPasswordlessVerifiedEvent(ctx, aggregate, "token1", "Passkey", "none", credentialID, publicKey, nil, 0, "")),
					),
				}
			}
			cmd = &SessionCommands{sessionWriteModel: wm, eventstore: eventstoreExpect(t, expected...), now: time.Now}
			_, err = provider.CheckWebAuthN(json.RawMessage(assertion))(ctx, cmd)
			if test.wantError {
				require.Error(t, err)
				require.Empty(t, cmd.eventCommands, "an invalid assertion must not authenticate a session")
				require.Equal(t, test.boundUser, wm.UserID)
				return
			}
			require.NoError(t, err)
			require.Equal(t, "user1", wm.UserID)
			require.Len(t, cmd.eventCommands, 3)
			require.IsType(t, &session.UserCheckedEvent{}, cmd.eventCommands[0])
			require.IsType(t, &session.WebAuthNCheckedEvent{}, cmd.eventCommands[1])
			for _, event := range cmd.eventCommands {
				require.NoError(t, AppendAndReduce(wm, event.(eventstore.Event)))
			}
			require.Nil(t, wm.WebAuthNChallenge, "a successful assertion consumes the challenge")
			cmd.eventCommands = nil
			_, err = provider.CheckWebAuthN(json.RawMessage(assertion))(ctx, cmd)
			require.Error(t, err, "the same assertion cannot be replayed")
			require.Empty(t, cmd.eventCommands)
		})
	}
}

func TestDiscoverablePasskeyRejectsU2F(t *testing.T) {
	provider := &Commands{webauthnConfig: &webauthnconfig.Config{DisplayName: "Test"}}
	for _, verification := range []domain.UserVerificationRequirement{domain.UserVerificationRequirementDiscouraged, domain.UserVerificationRequirementPreferred} {
		cmd := &SessionCommands{sessionWriteModel: NewSessionWriteModel("session1", "instance1")}
		var options json.RawMessage
		_, err := provider.CreateWebAuthNChallenge(verification, "localhost", &options)(context.Background(), cmd)
		require.Error(t, err)
		require.Empty(t, cmd.eventCommands)
	}
}
