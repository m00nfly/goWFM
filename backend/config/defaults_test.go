package config

import (
	"strings"
	"testing"
)

func TestUpgradeBuiltinEmailTemplatesPreservesCustomTemplates(t *testing.T) {
	latest := DefaultResetPasswordTemplate()
	legacy := latest
	legacy.HTML = strings.Replace(legacy.HTML, DefaultEmailFooterHTML, "", 1)
	legacy.HTML = strings.Replace(legacy.HTML, "\n\n  </div>", "\n  </div>", 1)
	inlineFooter := legacy
	inlineFooter.HTML = strings.Replace(
		inlineFooter.HTML,
		"    </div>\n  </div>",
		legacyInlineEmailFooterHTML+"\n    </div>\n  </div>",
		1,
	)
	custom := EmailTemplate{Subject: "自定义", HTML: "<p>custom</p>"}
	templates := map[string]EmailTemplate{
		"reset_password":     legacy,
		"share_notification": custom,
	}

	UpgradeBuiltinEmailTemplates(templates)
	if templates["reset_password"] != latest {
		t.Fatal("legacy built-in template was not upgraded")
	}
	if templates["share_notification"] != custom {
		t.Fatal("custom template must not be overwritten")
	}

	templates["reset_password"] = inlineFooter
	UpgradeBuiltinEmailTemplates(templates)
	if templates["reset_password"] != latest {
		t.Fatal("inline-footer built-in template was not upgraded")
	}
}
