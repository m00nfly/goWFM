package services

import (
	"testing"

	"goWFM/config"
)

func TestEmailDependentSettingsRequireActiveSMTP(t *testing.T) {
	config.InitDefaults()
	t.Cleanup(config.InitDefaults)

	security := config.GetSecurity()
	security.AllowEmailPasswordReset = true
	if err := UpdateSecuritySettings(security); err == nil {
		t.Fatal("expected password reset setting to require active SMTP")
	}

	share := config.GetShare()
	share.AllowEmailShare = true
	if err := UpdateShareSettings(share); err == nil {
		t.Fatal("expected email share setting to require active SMTP")
	}
}
