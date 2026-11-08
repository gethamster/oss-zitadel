package templates

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/zitadel/logging"
	"golang.org/x/text/language"

	"github.com/zitadel/zitadel/internal/i18n"
)

func TestVerificationEmailTranslation(t *testing.T) {
	var logs bytes.Buffer
	originalOutput := logging.New().Logger.Out
	logging.SetOutput(&logs)
	t.Cleanup(func() { logging.SetOutput(originalOutput) })

	translator := i18n.NewNotificationTranslator(language.English, []language.Tag{language.English})
	const code = "test-verification-secret"
	args := map[string]interface{}{"DisplayName": "Test user", "Code": code}
	var data TemplateData

	data.Translate(translator, "VerifyEmail", args, "en")
	require.Contains(t, data.Text, code)
	require.False(t, data.IncludeFooter)
	require.Empty(t, logs.String())

	require.Equal(t, "VerifyEmail.Missing", translator.Localize("VerifyEmail.Missing", args, "en"))
	require.Contains(t, logs.String(), "VerifyEmail.Missing")
	require.NotContains(t, logs.String(), code)

	require.NoError(t, translator.AddMessages(language.English, i18n.Message{
		ID: "VerifyEmail.Footer", Text: "Thanks, {{.DisplayName}}",
	}))
	data.Translate(translator, "VerifyEmail", args, "en")
	require.True(t, data.IncludeFooter)
	require.Equal(t, "Thanks, Test user", data.FooterText)
}
