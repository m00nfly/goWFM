package services

import (
	"fmt"
	"net/textproto"
	"strings"
	"testing"

	"goWFM/config"
)

func TestRenderResetPasswordEmail(t *testing.T) {
	subject, body, err := RenderResetPasswordEmail(config.DefaultResetPasswordTemplate(), ResetPasswordTemplateData{
		SiteName: "Team Files", Username: "alice", ResetURL: "https://files.example.com/login?reset_token=abc", ExpiresMinutes: 15,
	})
	if err != nil {
		t.Fatalf("render template: %v", err)
	}
	if subject != "重置您的 Team Files 密码" {
		t.Fatalf("unexpected subject: %q", subject)
	}
	for _, expected := range []string{"alice", "15", "https://files.example.com/login?reset_token=abc"} {
		if !strings.Contains(body, expected) {
			t.Fatalf("body does not contain %q", expected)
		}
	}
}

func TestRenderResetPasswordEmailRejectsUnknownVariable(t *testing.T) {
	_, _, err := RenderResetPasswordEmail(config.EmailTemplate{Subject: "{{.Unknown}}", HTML: "<p>ok</p>"}, ResetPasswordTemplateData{})
	if err == nil {
		t.Fatal("expected unknown template variable to fail")
	}
}

func TestSMTPErrorDetailsUnwrapsServerResponse(t *testing.T) {
	err := fmt.Errorf("SMTP 认证失败: %w", &textproto.Error{Code: 535, Msg: "5.7.8 Authentication credentials invalid"})
	code, message, ok := SMTPErrorDetails(err)
	if !ok || code != 535 || message != "5.7.8 Authentication credentials invalid" {
		t.Fatalf("unexpected SMTP details: code=%d message=%q ok=%v", code, message, ok)
	}
}

func TestEffectiveSenderNameFallsBackToSiteName(t *testing.T) {
	config.InitDefaults()
	defer config.InitDefaults()
	basic := config.GetBasic()
	basic.SiteName = "Team Files"
	config.SetBasic(basic)
	if got := EffectiveSenderName(""); got != "Team Files" {
		t.Fatalf("expected site name fallback, got %q", got)
	}
	if got := EffectiveSenderName("Notification Center"); got != "Notification Center" {
		t.Fatalf("expected explicit sender name, got %q", got)
	}
}
