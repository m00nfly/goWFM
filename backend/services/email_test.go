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
		SiteName: "Team Files", PoweredBy: "goWFM", Username: "alice", ResetURL: "https://files.example.com/login?reset_token=abc", ExpiresMinutes: 15,
	})
	if err != nil {
		t.Fatalf("render template: %v", err)
	}
	if subject != "重置您的 Team Files 密码" {
		t.Fatalf("unexpected subject: %q", subject)
	}
	for _, expected := range []string{"alice", "15", "https://files.example.com/login?reset_token=abc", "Team Files", `Powered by <a href="https://gowfm.dev"`, ">goWFM</a>"} {
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

func TestRenderShareNotificationEmail(t *testing.T) {
	subject, body, err := RenderShareNotificationEmail(config.DefaultShareNotificationTemplate(), ShareNotificationTemplateData{
		SiteName: "Team Files", PoweredBy: "goWFM", Sharer: "Alice", ShareName: "季度资料", FileCount: 4, ShareURL: "https://files.example.com/share/abc",
	})
	if err != nil {
		t.Fatalf("render share template: %v", err)
	}
	if subject != "Alice 向您分享了「季度资料」" {
		t.Fatalf("unexpected subject: %q", subject)
	}
	for _, expected := range []string{"Alice", "季度资料", "4", "https://files.example.com/share/abc", "Team Files", `Powered by <a href="https://gowfm.dev"`, ">goWFM</a>"} {
		if !strings.Contains(body, expected) {
			t.Fatalf("body does not contain %q", expected)
		}
	}
}

func TestRenderShareNotificationEmailRejectsResetVariables(t *testing.T) {
	_, _, err := RenderShareNotificationEmail(config.EmailTemplate{Subject: "{{.Username}}", HTML: "<p>ok</p>"}, ShareNotificationTemplateData{})
	if err == nil {
		t.Fatal("expected template variable from another notification type to fail")
	}
}

func TestDefaultEmailStartsInactiveWithRequiredTemplates(t *testing.T) {
	cfg := config.DefaultEmail()
	if cfg.Active {
		t.Fatal("default email configuration must be inactive")
	}
	for _, key := range []string{ResetPasswordTemplateKey, ShareNotificationTemplateKey} {
		if _, ok := cfg.Templates[key]; !ok {
			t.Fatalf("missing default template %q", key)
		}
	}
}

func TestSendShareNotificationEmailRequiresActiveSMTP(t *testing.T) {
	config.InitDefaults()
	t.Cleanup(config.InitDefaults)
	share := config.GetShare()
	share.AllowEmailShare = true
	config.SetShare(share)
	err := SendShareNotificationEmail("recipient@example.com", "Alice", "资料", 1, "token")
	if err == nil || !strings.Contains(err.Error(), "SMTP 服务未激活") {
		t.Fatalf("expected inactive SMTP error, got %v", err)
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
