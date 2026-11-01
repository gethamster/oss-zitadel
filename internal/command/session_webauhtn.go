package command

import (
	"bytes"
	"context"
	"encoding/json"
	"unicode/utf8"

	"github.com/go-webauthn/webauthn/protocol"

	"github.com/zitadel/zitadel/internal/domain"
	"github.com/zitadel/zitadel/internal/eventstore"
	"github.com/zitadel/zitadel/internal/zerrors"
)

type humanWebAuthNTokens struct {
	human  *domain.Human
	tokens []*domain.WebAuthNToken
}

func (s *SessionCommands) getHumanWebAuthNTokens(ctx context.Context, userVerification domain.UserVerificationRequirement) (*humanWebAuthNTokens, error) {
	humanWritemodel, err := s.gethumanWriteModel(ctx)
	if err != nil {
		return nil, err
	}
	tokenReadModel, err := s.getHumanWebAuthNTokenReadModel(ctx, userVerification)
	if err != nil {
		return nil, err
	}
	return &humanWebAuthNTokens{
		human:  writeModelToHuman(humanWritemodel),
		tokens: readModelToWebAuthNTokens(tokenReadModel),
	}, nil
}

func (s *SessionCommands) getHumanWebAuthNTokenReadModel(ctx context.Context, userVerification domain.UserVerificationRequirement) (readModel HumanWebAuthNTokensReadModel, err error) {
	readModel = NewHumanU2FTokensReadModel(s.sessionWriteModel.UserID, s.sessionWriteModel.UserResourceOwner)
	if userVerification == domain.UserVerificationRequirementRequired {
		readModel = NewHumanPasswordlessTokensReadModel(s.sessionWriteModel.UserID, s.sessionWriteModel.UserResourceOwner)
	}
	err = s.eventstore.FilterToQueryReducer(ctx, readModel)
	if err != nil {
		return nil, err
	}
	return readModel, nil
}

func (c *Commands) CreateWebAuthNChallenge(userVerification domain.UserVerificationRequirement, rpid string, dst json.Unmarshaler) SessionCommand {
	return func(ctx context.Context, cmd *SessionCommands) ([]eventstore.Command, error) {
		var webAuthNLogin *domain.WebAuthNLogin
		var err error
		if cmd.sessionWriteModel.UserID == "" {
			if userVerification != domain.UserVerificationRequirementRequired {
				return nil, zerrors.ThrowPreconditionFailed(nil, "COMMAND-D4g2a", "Errors.User.UserIDMissing")
			}
			webAuthNLogin, err = c.webauthnConfig.BeginDiscoverableLogin(ctx, rpid)
		} else {
			var humanPasskeys *humanWebAuthNTokens
			humanPasskeys, err = cmd.getHumanWebAuthNTokens(ctx, userVerification)
			if err != nil {
				return nil, err
			}
			webAuthNLogin, err = c.webauthnConfig.BeginLogin(ctx, humanPasskeys.human, userVerification, rpid, humanPasskeys.tokens...)
		}
		if err != nil {
			return nil, err
		}
		if err = json.Unmarshal(webAuthNLogin.CredentialAssertionData, dst); err != nil {
			return nil, zerrors.ThrowInternal(err, "COMMAND-Yah6A", "Errors.Internal")
		}

		cmd.WebAuthNChallenged(ctx, webAuthNLogin.Challenge, webAuthNLogin.AllowedCredentialIDs, webAuthNLogin.UserVerification, rpid)
		return nil, nil
	}
}

func (c *Commands) CheckWebAuthN(credentialAssertionData json.Marshaler) SessionCommand {
	return func(ctx context.Context, cmd *SessionCommands) ([]eventstore.Command, error) {
		credentialAssertionData, err := json.Marshal(credentialAssertionData)
		if err != nil {
			return nil, zerrors.ThrowInternal(err, "COMMAND-ohG2o", "Errors.Internal")
		}
		challenge := cmd.sessionWriteModel.WebAuthNChallenge
		if challenge == nil {
			return nil, zerrors.ThrowPreconditionFailed(nil, "COMMAND-Ioqu5", "Errors.Session.WebAuthN.NoChallenge")
		}
		webAuthNTokens, err := cmd.assertionUser(ctx, challenge.UserVerification, credentialAssertionData)
		if err != nil {
			return nil, err
		}
		webAuthN := challenge.WebAuthNLogin(webAuthNTokens.human, credentialAssertionData)

		credential, err := c.webauthnConfig.FinishLogin(ctx, webAuthNTokens.human, webAuthN, credentialAssertionData, webAuthNTokens.tokens...)
		if err != nil && (credential == nil || credential.ID == nil) {
			return nil, err
		}
		_, token := domain.GetTokenByKeyID(webAuthNTokens.tokens, credential.ID)
		if token == nil {
			return nil, zerrors.ThrowPreconditionFailed(nil, "COMMAND-Aej7i", "Errors.User.WebAuthN.NotFound")
		}
		// Account discovery is only committed after the provider has verified the
		// credential, challenge and required user verification for that account.
		if cmd.sessionWriteModel.UserID == "" {
			human := webAuthNTokens.human
			if err := cmd.UserChecked(ctx, human.AggregateID, human.ResourceOwner, cmd.now(), &human.PreferredLanguage); err != nil {
				return nil, err
			}
		}
		cmd.WebAuthNChecked(ctx, cmd.now(), token.WebAuthNTokenID, credential.Authenticator.SignCount, credential.Flags.UserVerified)
		return nil, nil
	}
}

// assertionUser selects credentials for verification, without authenticating the
// user identified by an untrusted discoverable-credential handle.
func (s *SessionCommands) assertionUser(ctx context.Context, verification domain.UserVerificationRequirement, assertion []byte) (*humanWebAuthNTokens, error) {
	if s.sessionWriteModel.UserID != "" {
		return s.getHumanWebAuthNTokens(ctx, verification)
	}
	if verification != domain.UserVerificationRequirementRequired {
		return nil, zerrors.ThrowPreconditionFailed(nil, "COMMAND-M8k2a", "Errors.User.UserIDMissing")
	}
	parsed, err := protocol.ParseCredentialRequestResponseBody(bytes.NewReader(assertion))
	if err != nil {
		return nil, zerrors.ThrowInvalidArgument(err, "COMMAND-Y9w3a", "Errors.User.WebAuthN.ValidateLoginFailed")
	}
	handle := parsed.Response.UserHandle
	if len(handle) == 0 || len(handle) > 200 || !utf8.Valid(handle) {
		return nil, zerrors.ThrowInvalidArgument(nil, "COMMAND-V3f9b", "Errors.User.UserIDMissing")
	}
	// Resolve within this instance, and only accept an active human account.
	// The temporary model never changes the real session before verification.
	candidate := *s
	candidate.sessionWriteModel = &SessionWriteModel{UserID: string(handle)}
	return candidate.getHumanWebAuthNTokens(ctx, verification)
}
