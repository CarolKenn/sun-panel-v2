package system

import (
	"sun-panel/api/api_v1"

	"github.com/gin-gonic/gin"
)

func InitPingRouter(router *gin.RouterGroup) {
	ping := api_v1.ApiGroupApp.ApiSystem.Ping
	{
		router.GET("system/ping", ping.Get)
	}
}
