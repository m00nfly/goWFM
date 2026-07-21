package handlers

import (
	"net/http"
	"strconv"

	"goWFM/services"

	"github.com/gin-gonic/gin"
)

func GetDashboard(c *gin.Context) {
	data, err := services.GetDashboardData()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "load dashboard failed"})
		return
	}
	c.JSON(http.StatusOK, data)
}

func GetDashboardActivity(c *gin.Context) {
	days, _ := strconv.Atoi(c.DefaultQuery("days", "7"))
	activity, effectiveDays, maxDays, err := services.GetDashboardActivity(days)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "load dashboard activity failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"activity": activity, "days": effectiveDays, "max_days": maxDays})
}

func GetStorageScanStatus(c *gin.Context) {
	c.JSON(http.StatusOK, services.GetDashboardScanStatus())
}

func TriggerStorageScan(c *gin.Context) {
	if !services.TriggerDashboardStorageScan("manual") {
		c.JSON(http.StatusConflict, gin.H{"error": "storage scan already running"})
		return
	}
	c.JSON(http.StatusAccepted, gin.H{"message": "storage scan started"})
}
