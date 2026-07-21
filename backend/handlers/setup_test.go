package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"goWFM/config"
	"goWFM/db"
	"goWFM/services"

	"github.com/gin-gonic/gin"
)

func initSetupTestDB(t *testing.T) {
	t.Helper()
	if err := db.Init(filepath.Join(t.TempDir(), "setup.db")); err != nil {
		t.Fatalf("init database: %v", err)
	}
	t.Cleanup(db.Close)
	if err := services.LoadAllConfigs(); err != nil {
		t.Fatalf("load configs: %v", err)
	}
}

func TestConfigInfoPublishesCustomBrandPanel(t *testing.T) {
	gin.SetMode(gin.TestMode)
	initSetupTestDB(t)

	appearance := config.GetAppearance()
	appearance.CustomBrandPanelEnabled = true
	appearance.CustomBrandPanelContent = "## Team files"
	if err := services.UpdateAppearanceSettings(appearance); err != nil {
		t.Fatalf("update appearance settings: %v", err)
	}

	recorder := performJSONRequest(t, http.MethodGet, "/api/config/info", nil, GetConfigInfo)
	if recorder.Code != http.StatusOK {
		t.Fatalf("config info status = %d", recorder.Code)
	}
	response := decodeJSONResponse(t, recorder)
	if enabled, _ := response["custom_brand_panel_enabled"].(bool); !enabled {
		t.Fatal("expected custom brand panel to be enabled")
	}
	if content, _ := response["custom_brand_panel_content"].(string); content != "## Team files" {
		t.Fatalf("custom brand panel content = %q", content)
	}
}

func performJSONRequest(t *testing.T, method, path string, body any, handler gin.HandlerFunc) *httptest.ResponseRecorder {
	t.Helper()
	payload, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	recorder := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(recorder)
	context.Request = httptest.NewRequest(method, path, bytes.NewReader(payload))
	context.Request.Header.Set("Content-Type", "application/json")
	handler(context)
	return recorder
}

func decodeJSONResponse(t *testing.T, recorder *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var response map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return response
}

func TestSetupUsesCustomAdminUsernameAndConfigInfoReportsStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)
	initSetupTestDB(t)

	before := performJSONRequest(t, http.MethodGet, "/api/config/info", nil, GetConfigInfo)
	if before.Code != http.StatusOK {
		t.Fatalf("config info before setup status = %d", before.Code)
	}
	if needsSetup, _ := decodeJSONResponse(t, before)["needs_setup"].(bool); !needsSetup {
		t.Fatal("expected needs_setup before creating an administrator")
	}

	setup := performJSONRequest(t, http.MethodPost, "/api/setup", SetupRequest{
		SiteName:      "Team Files",
		DataRootPath:  filepath.Join(t.TempDir(), "data"),
		ServerPort:    8080,
		AdminUsername: "owner.account",
		AdminPassword: "secure-password",
		AdminEmail:    "owner@example.com",
		MaxUploadSize: 1024,
	}, PostSetup)
	if setup.Code != http.StatusOK {
		t.Fatalf("setup status = %d, body = %s", setup.Code, setup.Body.String())
	}
	if got := decodeJSONResponse(t, setup)["admin_username"]; got != "owner.account" {
		t.Fatalf("admin_username = %v", got)
	}
	admin, err := services.GetUserByUsername("owner.account")
	if err != nil {
		t.Fatalf("get custom administrator: %v", err)
	}
	if !admin.IsAdmin {
		t.Fatal("created user is not an administrator")
	}

	after := performJSONRequest(t, http.MethodGet, "/api/config/info", nil, GetConfigInfo)
	if needsSetup, _ := decodeJSONResponse(t, after)["needs_setup"].(bool); needsSetup {
		t.Fatal("expected setup to be complete after creating an administrator")
	}
}

func TestSetupRejectsReservedOrWhitespaceAdminUsername(t *testing.T) {
	gin.SetMode(gin.TestMode)
	initSetupTestDB(t)

	for _, username := range []string{"Guest", "team owner"} {
		recorder := performJSONRequest(t, http.MethodPost, "/api/setup", SetupRequest{
			DataRootPath:  filepath.Join(t.TempDir(), "data"),
			AdminUsername: username,
			AdminPassword: "secure-password",
			AdminEmail:    "owner@example.com",
		}, PostSetup)
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("username %q status = %d, body = %s", username, recorder.Code, recorder.Body.String())
		}
	}
}
