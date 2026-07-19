package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"goWFM/config"
	"goWFM/services"

	"github.com/gin-gonic/gin"
)

func TestUpdateEmailTemplatePreservesSMTPSettingsAndActivation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	initSetupTestDB(t)
	cfg := config.DefaultEmail()
	cfg.Active = true
	cfg.SMTPHost = "smtp.example.com"
	cfg.SenderEmail = "sender@example.com"
	if err := services.UpdateEmailSettings(cfg); err != nil {
		t.Fatalf("save initial email settings: %v", err)
	}

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Params = gin.Params{{Key: "key", Value: services.ResetPasswordTemplateKey}}
	ctx.Request = httptest.NewRequest(http.MethodPut, "/api/admin/email/templates/reset_password", strings.NewReader(`{"subject":"自定义 {{.SiteName}}","html":"<p>{{.PoweredBy}}</p>"}`))
	ctx.Request.Header.Set("Content-Type", "application/json")
	UpdateEmailTemplate(ctx)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", recorder.Code, recorder.Body.String())
	}
	updated := config.GetEmail()
	if !updated.Active || updated.SMTPHost != cfg.SMTPHost || updated.SenderEmail != cfg.SenderEmail {
		t.Fatalf("SMTP settings changed while saving template: %+v", updated)
	}
	if updated.Templates[services.ResetPasswordTemplateKey].Subject != "自定义 {{.SiteName}}" {
		t.Fatal("template was not saved")
	}
}

func TestPasswordResetRequestRejectedWhenEmailInactive(t *testing.T) {
	gin.SetMode(gin.TestMode)
	config.InitDefaults()
	t.Cleanup(config.InitDefaults)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/auth/password-reset/request", strings.NewReader(`{"email":"user@example.com"}`))
	ctx.Request.Header.Set("Content-Type", "application/json")

	RequestPasswordReset(ctx)
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected %d, got %d", http.StatusServiceUnavailable, recorder.Code)
	}
	var result map[string]string
	if err := json.Unmarshal(recorder.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if result["error"] != passwordResetDisabledMessage {
		t.Fatalf("unexpected error: %q", result["error"])
	}
}

func TestEmailDeliveryChangesRequireAnotherTest(t *testing.T) {
	before := config.DefaultEmail()
	after := before
	if emailDeliverySettingsChanged(before, after) {
		t.Fatal("identical delivery settings should not count as changed")
	}
	after.SMTPHost = "smtp.example.com"
	if !emailDeliverySettingsChanged(before, after) {
		t.Fatal("SMTP host change should require another test")
	}
}

func TestEmailShareRejectedWhenSMTPInactive(t *testing.T) {
	gin.SetMode(gin.TestMode)
	config.InitDefaults()
	t.Cleanup(config.InitDefaults)
	share := config.GetShare()
	share.AllowEmailShare = true
	config.SetShare(share)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/shares/1/email", strings.NewReader(`{"email":"recipient@example.com"}`))
	ctx.Request.Header.Set("Content-Type", "application/json")

	EmailShareLink(ctx)
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected %d, got %d", http.StatusServiceUnavailable, recorder.Code)
	}
	var result map[string]string
	if err := json.Unmarshal(recorder.Body.Bytes(), &result); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if result["error"] != "SMTP 服务未激活，无法发送分享邮件" {
		t.Fatalf("unexpected error: %q", result["error"])
	}
}

func TestEmailShareRejectedWhenFeatureDisabled(t *testing.T) {
	gin.SetMode(gin.TestMode)
	config.InitDefaults()
	t.Cleanup(config.InitDefaults)
	email := config.GetEmail()
	email.Active = true
	config.SetEmail(email)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/shares/1/email", strings.NewReader(`{"email":"recipient@example.com"}`))
	ctx.Request.Header.Set("Content-Type", "application/json")
	EmailShareLink(ctx)

	if recorder.Code != http.StatusForbidden {
		t.Fatalf("expected %d, got %d", http.StatusForbidden, recorder.Code)
	}
}

func TestPasswordResetRequiresActiveSMTPAfterFeatureEnabled(t *testing.T) {
	gin.SetMode(gin.TestMode)
	config.InitDefaults()
	t.Cleanup(config.InitDefaults)
	security := config.GetSecurity()
	security.AllowEmailPasswordReset = true
	config.SetSecurity(security)

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	ctx.Request = httptest.NewRequest(http.MethodPost, "/api/auth/password-reset/request", strings.NewReader(`{"email":"user@example.com"}`))
	ctx.Request.Header.Set("Content-Type", "application/json")
	RequestPasswordReset(ctx)

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected %d, got %d", http.StatusServiceUnavailable, recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), "SMTP 服务未激活") {
		t.Fatalf("unexpected response: %s", recorder.Body.String())
	}
}
