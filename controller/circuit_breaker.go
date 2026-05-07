package controller

import (
	"net/http"
	"strconv"

	"github.com/QuantumNous/new-api/service"
	"github.com/gin-gonic/gin"
)

func GetCircuitBreakerStatus(c *gin.Context) {
	statuses := service.GetAllCircuitBreakerStatus()
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    statuses,
	})
}

func ResetCircuitBreaker(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "无效的渠道 ID",
		})
		return
	}
	service.ResetCircuitBreaker(id)
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
	})
}
