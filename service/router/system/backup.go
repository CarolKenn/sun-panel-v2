package system

import (
	"sun-panel/api/api_v1"
	"sun-panel/api/api_v1/middleware"

	"github.com/gin-gonic/gin"
)

// InitBackupRouter 数据备份与恢复
//
// 备份内容包含全部业务数据（含用户口令哈希、邮件服务密码、附件），
// 因此必须同时通过登录校验与管理员校验，且恢复接口额外要求二次密码确认。
func InitBackupRouter(router *gin.RouterGroup) {
	backupApi := api_v1.ApiGroupApp.ApiSystem.BackupApi

	private := router.Group("", middleware.LoginInterceptor, middleware.AdminInterceptor)
	{
		private.GET("/system/backup/info", backupApi.Info)
		private.POST("/system/backup/export", backupApi.Export)
		private.POST("/system/backup/import", backupApi.Import)
	}
}
