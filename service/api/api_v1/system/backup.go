package system

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"sun-panel/api/api_v1/common/apiReturn"
	"sun-panel/api/api_v1/common/base"
	"sun-panel/global"
	"sun-panel/lib/backup"
	"sun-panel/lib/cmn"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

// 恢复请求体上限（含附件时才可能这么大）
const maxBackupUploadSize = int64(4) << 30

type BackupApi struct {
}

// Info 当前数据概览：各表行数、附件数量与体积
func (a *BackupApi) Info(c *gin.Context) {
	sourcePath := global.Config.GetValueString("base", "source_path")
	info, err := backup.GetInfo(sourcePath, uploadTempDir())
	if err != nil {
		apiReturn.ErrorDatabase(c, err.Error())
		return
	}
	apiReturn.SuccessData(c, info)
}

// uploadTempDir 上传临时目录。它通常是 source_path 的子目录，
// 统计与打包时都要排除，否则会把上传中断残留的临时文件算进去。
func uploadTempDir() []string {
	if p := global.Config.GetValueString("base", "source_temp_path"); p != "" {
		return []string{p}
	}
	return nil
}

// Export 导出全站数据为 zip（含附件可选）
//
// 说明：直接以流的方式写响应体，避免大文件落临时盘。
// 若导出中途失败，客户端会得到一个不完整的 zip，恢复时会因格式校验被拒绝，不会写坏数据。
func (a *BackupApi) Export(c *gin.Context) {
	type Req struct {
		IncludeUploads bool `json:"includeUploads"`
	}
	req := Req{IncludeUploads: true}
	// 允许空请求体，默认导出附件
	if c.Request != nil && c.Request.ContentLength > 0 {
		if err := c.ShouldBindBodyWith(&req, binding.JSON); err != nil {
			apiReturn.ErrorParamFomat(c, err.Error())
			return
		}
	}

	sourcePath := global.Config.GetValueString("base", "source_path")
	fileName := fmt.Sprintf("sun-panel-backup-%s.zip", time.Now().Format("20060102-150405"))

	c.Header("Content-Type", "application/zip")
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, fileName))
	c.Header("Cache-Control", "no-store")
	c.Header("X-Content-Type-Options", "nosniff")
	c.Status(http.StatusOK)

	if err := backup.Export(c.Writer, backup.ExportOptions{
		IncludeUploads: req.IncludeUploads,
		UploadsDir:     sourcePath,
		ExcludeDirs:    uploadTempDir(),
	}); err != nil {
		global.Logger.Errorf("导出备份失败: %v", err)
	}
}

// Import 从备份包覆盖恢复数据
//
// 该操作会清空并重写全部业务数据，因此要求再次输入当前账号密码，
// 用于防止会话被劫持后的静默破坏。
func (a *BackupApi) Import(c *gin.Context) {
	userInfo, exist := base.GetCurrentUserInfo(c)
	if !exist {
		apiReturn.ErrorByCode(c, 1000)
		return
	}

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBackupUploadSize)

	// 二次校验当前账号密码
	password := c.PostForm("password")
	if password == "" || cmn.PasswordEncryption(password) != userInfo.Password {
		apiReturn.Error(c, "当前账号密码校验失败，已取消恢复")
		return
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		apiReturn.Error(c, "请选择要恢复的备份文件")
		return
	}

	tmp, err := os.CreateTemp("", "sun-panel-restore-*.zip")
	if err != nil {
		apiReturn.ErrorDatabase(c, "创建临时文件失败: "+err.Error())
		return
	}
	defer os.Remove(tmp.Name())
	defer tmp.Close()

	src, err := fileHeader.Open()
	if err != nil {
		apiReturn.Error(c, "读取上传文件失败: "+err.Error())
		return
	}
	defer src.Close()

	if _, err := io.Copy(tmp, src); err != nil {
		apiReturn.Error(c, "备份文件保存失败: "+err.Error())
		return
	}

	restoreUploads := c.PostForm("restoreUploads") == "true"
	sourcePath := global.Config.GetValueString("base", "source_path")

	result, err := backup.Restore(tmp.Name(), sourcePath, restoreUploads)
	if err != nil {
		global.Logger.Errorf("恢复备份失败: %v", err)
		apiReturn.Error(c, err.Error())
		return
	}

	// 数据已被替换，清空内存缓存避免读到旧配置与旧登录态
	backup.FlushCaches()
	global.Logger.Infof("数据恢复完成，共写入 %d 条记录，附件 %d 个", result.TotalRecords, result.UploadsRestored)

	apiReturn.SuccessData(c, gin.H{
		"tables":          result.Tables,
		"totalRecords":    result.TotalRecords,
		"uploadsRestored": result.UploadsRestored,
		"warnings":        result.Warnings,
		"manifest":        result.Manifest,
		// 用户表已被替换，当前会话可能失效，前端应引导重新登录
		"needRelogin": true,
	})
}
