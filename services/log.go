package services

import (
	"encoding/json"
	"fmt"
	"strings"

	"goWFM/db"
	"goWFM/models"
)

func CreateLog(userID int64, action, targetPath, ipAddress string, details map[string]interface{}) error {
	var detailsJSON string
	if details != nil {
		b, _ := json.Marshal(details)
		detailsJSON = string(b)
	}
	_, err := db.DB.Exec(
		`INSERT INTO operation_logs (user_id, action, target_path, details, ip_address) VALUES (?, ?, ?, ?, ?)`,
		userID, action, targetPath, detailsJSON, ipAddress,
	)
	return err
}

func CreateLoginLog(userID int64, ipAddress string, success bool) error {
	action := models.ActionLogin
	if !success {
		action = models.ActionLoginFail
	}
	return CreateLog(userID, action, "", ipAddress, nil)
}

func QueryLogs(startTime, endTime, userID, action, targetPath string, page, pageSize int) ([]map[string]interface{}, int, error) {
	var conditions []string
	var args []interface{}

	if startTime != "" {
		conditions = append(conditions, "created_at >= ?")
		args = append(args, startTime)
	}
	if endTime != "" {
		conditions = append(conditions, "created_at <= ?")
		args = append(args, endTime)
	}
	if userID != "" {
		conditions = append(conditions, "user_id = ?")
		args = append(args, userID)
	}
	if action != "" {
		conditions = append(conditions, "action = ?")
		args = append(args, action)
	}
	if targetPath != "" {
		conditions = append(conditions, "target_path LIKE ?")
		args = append(args, "%"+targetPath+"%")
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	var total int
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM operation_logs %s", whereClause)
	db.DB.QueryRow(countQuery, args...).Scan(&total)

	offset := (page - 1) * pageSize
	query := fmt.Sprintf(
		`SELECT id, user_id, action, target_path, details, ip_address, created_at FROM operation_logs %s ORDER BY created_at DESC LIMIT ? OFFSET ?`,
		whereClause,
	)
	queryArgs := append(args, pageSize, offset)

	rows, err := db.DB.Query(query, queryArgs...)
	if err != nil {
		return nil, 0, err
	}

	type rawLog struct {
		id         int64
		userID     int64
		action     string
		targetPath string
		details    string
		ipAddr     string
		createdAt  string
	}

	var rawLogs []rawLog
	for rows.Next() {
		var r rawLog
		rows.Scan(&r.id, &r.userID, &r.action, &r.targetPath, &r.details, &r.ipAddr, &r.createdAt)
		rawLogs = append(rawLogs, r)
	}
	rows.Close()

	result := make([]map[string]interface{}, 0, len(rawLogs))
	for _, r := range rawLogs {
		username := ""
		u, _ := GetUserByID(r.userID)
		if u != nil {
			username = u.Username
		}

		result = append(result, map[string]interface{}{
			"id":          r.id,
			"user_id":     r.userID,
			"username":    username,
			"action":      r.action,
			"target_path": r.targetPath,
			"details":     r.details,
			"ip_address":  r.ipAddr,
			"created_at":  r.createdAt,
		})
	}

	return result, total, nil
}
