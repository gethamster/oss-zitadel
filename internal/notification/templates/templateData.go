package templates

import (
	"fmt"

	"github.com/zitadel/zitadel/internal/domain"
	"github.com/zitadel/zitadel/internal/i18n"
)

const (
	DefaultFontFamily      = "-apple-system, BlinkMacSystemFont, Segoe UI, Lato, Arial, Helvetica, sans-serif"
	DefaultFontColor       = "#22292f"
	DefaultBackgroundColor = "#fafafa"
	DefaultPrimaryColor    = "#5282C1"
)

type TemplateData struct {
	Title           string `json:"title,omitempty"`
	PreHeader       string `json:"preHeader,omitempty"`
	Subject         string `json:"subject,omitempty"`
	Greeting        string `json:"greeting,omitempty"`
	Text            string `json:"text,omitempty"`
	URL             string `json:"url,omitempty"`
	ButtonText      string `json:"buttonText,omitempty"`
	PrimaryColor    string `json:"primaryColor,omitempty"`
	BackgroundColor string `json:"backgroundColor,omitempty"`
	FontColor       string `json:"fontColor,omitempty"`
	LogoURL         string `json:"logoUrl,omitempty"`
	FontURL         string `json:"fontUrl,omitempty"`
	FontFaceFamily  string `json:"fontFaceFamily,omitempty"`
	FontFamily      string `json:"fontFamily,omitempty"`

	IncludeFooter bool   `json:"includeFooter,omitempty"`
	FooterText    string `json:"footerText,omitempty"`
}

func (data *TemplateData) Translate(translator *i18n.Translator, msgType string, args map[string]interface{}, langs ...string) {
	data.Title = translator.Localize(fmt.Sprintf("%s.%s", msgType, domain.MessageTitle), args, langs...)
	data.PreHeader = translator.Localize(fmt.Sprintf("%s.%s", msgType, domain.MessagePreHeader), args, langs...)
	data.Subject = translator.Localize(fmt.Sprintf("%s.%s", msgType, domain.MessageSubject), args, langs...)
	data.Greeting = translator.Localize(fmt.Sprintf("%s.%s", msgType, domain.MessageGreeting), args, langs...)
	data.Text = translator.Localize(fmt.Sprintf("%s.%s", msgType, domain.MessageText), args, langs...)
	data.ButtonText = translator.Localize(fmt.Sprintf("%s.%s", msgType, domain.MessageButtonText), args, langs...)
	// Footers are only configured through custom message texts.
	data.FooterText = translator.LocalizeOptional(fmt.Sprintf("%s.%s", msgType, domain.MessageFooterText), args, langs...)
	data.IncludeFooter = data.FooterText != ""
}
