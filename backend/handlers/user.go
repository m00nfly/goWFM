package handlers

import (
	"net/http"
	"strconv"

	"goWFM/models"
	"goWFM/services"

	"github.com/gin-gonic/gin"
)

func ListUsers(c *gin.Context) {
	rows, err := services.ListAllUsers()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "list users failed"})
		return
	}
	c.JSON(http.StatusOK, rows)
}

type CreateUserRequest struct {
	Username    string `json:"username" binding:"required"`
	Password    string `json:"password" binding:"required,min=6"`
	DisplayName string `json:"display_name"`
	Email       string `json:"email"`
	Permissions int    `json:"permissions"`
	IsAdmin     bool   `json:"is_admin"`
	TotpForced  bool   `json:"totp_forced"`
}

func CreateUser(c *gin.Context) {
	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := services.CreateUser(req.Username, req.Password, req.DisplayName, req.Email, req.IsAdmin, req.Permissions)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "create user failed: " + err.Error()})
		return
	}
	if req.TotpForced {
		if err := services.AdminEnableTOTP(user.ID); err != nil {
			services.DeleteUserByID(user.ID)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "create user TOTP policy failed"})
			return
		}
		user.TotpForced = true
	}

	services.CreateLog(c.GetInt64("userID"), models.ActionUserCreate, "", c.ClientIP(), map[string]interface{}{"username": req.Username})
	c.JSON(http.StatusOK, gin.H{
		"id":           user.ID,
		"username":     user.Username,
		"display_name": user.DisplayName,
		"email":        user.Email,
		"is_admin":     user.IsAdmin,
		"permissions":  user.Permissions,
		"totp_forced":  user.TotpForced,
	})
}

type UpdateUserRequest struct {
	DisplayName string `json:"display_name"`
	Email       string `json:"email"`
	Permissions int    `json:"permissions"`
	IsAdmin     *bool  `json:"is_admin"`
}

func UpdateUser(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	if id == 0 {
		c.JSON(http.StatusForbidden, gin.H{"error": "cannot modify system Guest account"})
		return
	}

	var req UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := services.GetUserByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	if user.IsAdmin && req.IsAdmin != nil && !*req.IsAdmin {
		adminCount, _ := services.AdminCount()
		if adminCount <= 1 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "cannot remove the last admin"})
			return
		}
	}

	isAdmin := user.IsAdmin
	if req.IsAdmin != nil {
		isAdmin = *req.IsAdmin
	}

	_, err = services.UpdateUserFields(id, req.DisplayName, req.Email, isAdmin, req.Permissions)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "update user failed"})
		return
	}

	services.CreateLog(c.GetInt64("userID"), models.ActionUserUpdate, "", c.ClientIP(), map[string]interface{}{"target_user_id": id})
	c.JSON(http.StatusOK, gin.H{"message": "user updated"})
}

func DeleteUser(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	if id == 0 {
		c.JSON(http.StatusForbidden, gin.H{"error": "cannot delete system Guest account"})
		return
	}

	user, err := services.GetUserByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	if user.IsAdmin {
		adminCount, _ := services.AdminCount()
		if adminCount <= 1 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "cannot delete the last admin"})
			return
		}
	}

	if err := services.DeleteUserByID(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "delete user failed"})
		return
	}

	services.DeleteAllUserSessions(id)
	services.CreateLog(c.GetInt64("userID"), models.ActionUserDelete, "", c.ClientIP(), map[string]interface{}{"target_user_id": id})
	c.JSON(http.StatusOK, gin.H{"message": "user deleted"})
}

type UpdateMeRequest struct {
	DisplayName string `json:"display_name"`
	Email       string `json:"email"`
}

func UpdateMe(c *gin.Context) {
	userID := c.GetInt64("userID")
	var req UpdateMeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := services.GetUserByID(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "user not found"})
		return
	}

	_, err = services.UpdateUserFields(userID, req.DisplayName, req.Email, user.IsAdmin, user.Permissions)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "update failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "profile updated"})
}

type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required,min=6"`
	TotpCode        string `json:"totp_code"`
}

func ChangePassword(c *gin.Context) {
	userID := c.GetInt64("userID")
	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := services.GetUserByID(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "user not found"})
		return
	}

	if !services.CheckPassword(user, req.CurrentPassword) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "current password is incorrect"})
		return
	}

	// 如果用户启用了 TOTP，修改密码需要验证 TOTP
	if user.TotpEnabled {
		if req.TotpCode == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "请输入 TOTP 验证码"})
			return
		}
		if err := services.VerifyTOTP(userID, req.TotpCode); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "TOTP 验证码错误"})
			return
		}
	}

	if err := services.UpdateUserPassword(userID, req.NewPassword); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "update password failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "password changed"})
}

// ---------- TOTP 用户设置端点 ----------

func SetupTOTP(c *gin.Context) {
	userID := c.GetInt64("userID")
	user, err := services.GetUserByID(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "user not found"})
		return
	}
	if user.TotpEnabled {
		c.JSON(http.StatusBadRequest, gin.H{"error": "TOTP 已启用，请先禁用后再重新设置"})
		return
	}

	qrBase64, err := services.SetupTOTP(userID, user.Username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "生成 TOTP 密钥失败"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"qr_code": "data:image/png;base64," + qrBase64,
	})
}

func VerifyTOTPSetup(c *gin.Context) {
	userID := c.GetInt64("userID")
	var req struct {
		Code string `json:"code" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "请输入验证码"})
		return
	}

	codes, err := services.VerifyTOTPSetup(userID, req.Code)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	services.CreateLog(userID, models.ActionTOTPEnable, "", c.ClientIP(), nil)

	c.JSON(http.StatusOK, gin.H{
		"message":        "TOTP 已启用",
		"recovery_codes": codes,
	})
}

func DisableMyTOTP(c *gin.Context) {
	userID := c.GetInt64("userID")
	user, err := services.GetUserByID(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "user not found"})
		return
	}
	if user.TotpForced {
		c.JSON(http.StatusForbidden, gin.H{"error": "管理员已强制启用 TOTP，无法自行禁用"})
		return
	}

	if err := services.DisableTOTP(userID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "disable totp failed"})
		return
	}

	services.CreateLog(userID, models.ActionTOTPDisable, "", c.ClientIP(), nil)

	c.JSON(http.StatusOK, gin.H{"message": "TOTP 已禁用"})
}

func GetMyTOTPStatus(c *gin.Context) {
	userID := c.GetInt64("userID")
	user, err := services.GetUserByID(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "user not found"})
		return
	}

	remaining := 0
	if user.TotpEnabled {
		remaining = services.GetRecoveryCodesRemaining(userID)
	}

	c.JSON(http.StatusOK, gin.H{
		"totp_enabled":             user.TotpEnabled,
		"totp_forced":              user.TotpForced,
		"setup_required":           user.TotpResetRequired || (user.TotpForced && !user.TotpEnabled),
		"reset_required":           user.TotpResetRequired,
		"recovery_codes_remaining": remaining,
	})
}

// ---------- 管理员 TOTP 管理端点 ----------

func AdminDisableTOTP(c *gin.Context) {
	targetID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	if targetID == 0 {
		c.JSON(http.StatusForbidden, gin.H{"error": "cannot modify system Guest account"})
		return
	}

	user, err := services.GetUserByID(targetID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	if !user.TotpEnabled && !user.TotpForced {
		c.JSON(http.StatusBadRequest, gin.H{"error": "该用户未启用 TOTP"})
		return
	}

	if err := services.AdminDisableTOTP(targetID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "disable totp failed"})
		return
	}

	userID := c.GetInt64("userID")
	services.CreateLog(userID, models.ActionTOTPDisable, "", c.ClientIP(),
		map[string]interface{}{"target_user_id": targetID})

	c.JSON(http.StatusOK, gin.H{"message": "TOTP 已禁用"})
}

// AdminUpdateTOTP 管理员启用/禁用用户的 TOTP
func AdminUpdateTOTP(c *gin.Context) {
	targetID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	if targetID == 0 {
		c.JSON(http.StatusForbidden, gin.H{"error": "cannot modify system Guest account"})
		return
	}

	var req struct {
		TotpForced  *bool `json:"totp_forced"`
		TotpEnabled *bool `json:"totp_enabled"` // 兼容旧版管理端
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	forced := req.TotpForced
	if forced == nil {
		forced = req.TotpEnabled
	}
	if forced == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "totp_forced is required"})
		return
	}

	userID := c.GetInt64("userID")
	if *forced {
		if err := services.AdminEnableTOTP(targetID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "enable totp failed"})
			return
		}
		services.CreateLog(userID, models.ActionTOTPEnable, "", c.ClientIP(),
			map[string]interface{}{"target_user_id": targetID})
		c.JSON(http.StatusOK, gin.H{"message": "已要求用户启用 TOTP"})
	} else {
		if err := services.AdminUnforceTOTP(targetID); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "取消强制 TOTP 失败"})
			return
		}
		services.CreateLog(userID, models.ActionUserUpdate, "", c.ClientIP(),
			map[string]interface{}{"target_user_id": targetID, "totp_forced": false})
		c.JSON(http.StatusOK, gin.H{"message": "已取消强制 TOTP，用户可自主关闭"})
	}
}

func AdminResetTOTP(c *gin.Context) {
	targetID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || targetID == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}
	user, err := services.GetUserByID(targetID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	if !user.TotpEnabled {
		c.JSON(http.StatusBadRequest, gin.H{"error": "该用户未启用 TOTP"})
		return
	}
	if err = services.AdminResetTOTP(targetID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "重置 TOTP 失败"})
		return
	}
	services.CreateLog(c.GetInt64("userID"), models.ActionTOTPDisable, "", c.ClientIP(),
		map[string]interface{}{"target_user_id": targetID, "action": "reset"})
	c.JSON(http.StatusOK, gin.H{"message": "TOTP 密钥已重置，用户下次登录时必须重新绑定"})
}
