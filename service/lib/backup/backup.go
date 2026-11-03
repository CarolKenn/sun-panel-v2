package backup

// 数据备份 / 恢复核心逻辑
//
// 备份包结构（zip）：
//
//	manifest.json          元信息：格式标识、版本、导出时间、各表行数
//	data/<table>.json      每张表的全量数据（按数据库实际列导出，不受 json tag 影响）
//	uploads/...            上传的附件（可选）
//
// 设计要点：
//   - 导出/导入均以「数据库真实列」为准，避免模型上的 json:"-" 导致字段丢失；
//   - 导入前先完整解析并校验，再开启事务，避免中途失败留下半套数据；
//   - 解压时严格校验路径，防止 zip-slip 目录穿越；
//   - 表名来自固定白名单，不接受外部输入。

import (
	"archive/zip"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"time"

	"sun-panel/global"
	"sun-panel/lib/cmn"
	"sun-panel/models"

	"gorm.io/gorm"
	"gorm.io/gorm/schema"
)

const (
	// FormatName 备份格式标识
	FormatName = "sun-panel-backup"
	// FormatVersion 当前备份格式版本
	FormatVersion = 1

	manifestEntry = "manifest.json"
	dataPrefix    = "data/"
	uploadsPrefix = "uploads/"

	// 解压条目上限，防 zip 炸弹
	maxEntries = 200000
	// 解压后总体积上限 4GiB
	maxUncompressedBytes = int64(4) << 30
	// 单次备份纳入的附件体积上限 2GiB
	maxUploadsBytes = int64(2) << 30
	// 单条记录批量写入的批大小
	insertBatchSize = 200
)

var (
	// ErrInvalidFormat 备份文件格式不正确
	ErrInvalidFormat = errors.New("备份文件格式不正确，请选择本面板导出的备份包")
	// ErrVersionTooNew 备份文件版本高于当前程序
	ErrVersionTooNew = errors.New("备份文件的版本高于当前面板版本，请先升级面板后再恢复")
	// ErrUnsafePath 备份包内含非法路径
	ErrUnsafePath = errors.New("备份包中存在非法的文件路径，已中止恢复")
	// ErrTooLarge 备份包解压后体积过大
	ErrTooLarge = errors.New("备份包解压后体积超过上限，已中止恢复")
	// ErrNoUser 备份包不含任何用户
	ErrNoUser = errors.New("备份包中没有用户数据，恢复后将无法登录，已中止恢复")
	// ErrRestoring 已有恢复任务在执行
	ErrRestoring = errors.New("已有恢复任务正在执行，请稍后再试")
)

// restoreLock 保证同一时间只有一个恢复任务在写数据库
var restoreLock sync.Mutex

// Manifest 备份元信息
type Manifest struct {
	Format         string         `json:"format"`
	Version        int            `json:"version"`
	CreatedAt      time.Time      `json:"createdAt"`
	AppVersion     string         `json:"appVersion"`
	DbDriver       string         `json:"dbDriver"`
	IncludeUploads bool           `json:"includeUploads"`
	Tables         map[string]int `json:"tables"`
	Warnings       []string       `json:"warnings,omitempty"`
}

// Result 恢复结果
type Result struct {
	Manifest        *Manifest      `json:"manifest"`
	Tables          map[string]int `json:"tables"`
	TotalRecords    int            `json:"totalRecords"`
	UploadsRestored int            `json:"uploadsRestored"`
	Warnings        []string       `json:"warnings,omitempty"`
}

// Info 当前数据概览
type Info struct {
	AppVersion   string         `json:"appVersion"`
	DbDriver     string         `json:"dbDriver"`
	Tables       map[string]int `json:"tables"`
	UploadsDir   string         `json:"uploadsDir"`
	UploadsCount int64          `json:"uploadsCount"`
	UploadsSize  int64          `json:"uploadsSize"`
}

// ExportOptions 导出选项
type ExportOptions struct {
	IncludeUploads bool
	UploadsDir     string
}

type tableSpec struct {
	Name  string
	Model interface{}
}

// tableSpecs 返回需要备份的表。顺序为「父表 -> 子表」，
// 恢复时按相反顺序清空，减少外键约束带来的问题。
// 表名与 gorm SingularTable 配置一致。
func tableSpecs() []tableSpec {
	return []tableSpec{
		{"user", &models.User{}},
		{"item_icon_group", &models.ItemIconGroup{}},
		{"item_icon", &models.ItemIcon{}},
		{"bookmark", &models.Bookmark{}},
		{"user_config", &models.UserConfig{}},
		{"module_config", &models.ModuleConfig{}},
		{"file", &models.File{}},
		{"system_setting", &models.SystemSetting{}},
	}
}

// ----------------------------------------------------------------------------
// 导出
// ----------------------------------------------------------------------------

// Export 将当前全部数据写入 w（zip 流）。
// 说明：数据直接从 w 流式输出，不落临时文件；调用前应确保 w 已设置好响应头。
func Export(w io.Writer, opt ExportOptions) error {
	zw := zip.NewWriter(w)

	mf := Manifest{
		Format:         FormatName,
		Version:        FormatVersion,
		CreatedAt:      time.Now(),
		AppVersion:     cmn.GetSysVersionInfo().Version,
		DbDriver:       string(global.DB_DRIVER),
		IncludeUploads: opt.IncludeUploads,
		Tables:         map[string]int{},
	}

	for _, spec := range tableSpecs() {
		rows, err := dumpTable(spec)
		if err != nil {
			_ = zw.Close()
			return fmt.Errorf("导出数据表 %s 失败: %w", spec.Name, err)
		}
		mf.Tables[spec.Name] = len(rows)
		if err := writeJSONEntry(zw, dataPrefix+spec.Name+".json", rows); err != nil {
			_ = zw.Close()
			return fmt.Errorf("写入数据表 %s 失败: %w", spec.Name, err)
		}
	}

	if opt.IncludeUploads && opt.UploadsDir != "" {
		if err := addUploads(zw, opt.UploadsDir, &mf.Warnings); err != nil {
			_ = zw.Close()
			return fmt.Errorf("打包附件失败: %w", err)
		}
	}

	if err := writeJSONEntry(zw, manifestEntry, mf); err != nil {
		_ = zw.Close()
		return fmt.Errorf("写入元信息失败: %w", err)
	}

	return zw.Close()
}

// dumpTable 按数据库真实列导出整表数据（含软删除记录）
func dumpTable(spec tableSpec) ([]map[string]interface{}, error) {
	rows := make([]map[string]interface{}, 0)
	if err := global.Db.Unscoped().Model(spec.Model).Find(&rows).Error; err != nil {
		return nil, err
	}
	return rows, nil
}

func writeJSONEntry(zw *zip.Writer, name string, v interface{}) error {
	hdr := &zip.FileHeader{
		Name:     name,
		Method:   zip.Deflate,
		Modified: time.Now(),
	}
	fw, err := zw.CreateHeader(hdr)
	if err != nil {
		return err
	}
	enc := json.NewEncoder(fw)
	enc.SetEscapeHTML(false)
	return enc.Encode(v)
}

// addUploads 把附件目录递归写入 zip
func addUploads(zw *zip.Writer, dir string, warnings *[]string) error {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return err
	}
	if _, err := os.Stat(absDir); err != nil {
		if os.IsNotExist(err) {
			// 没有附件目录，视为空
			return nil
		}
		return err
	}

	var total int64
	return filepath.Walk(absDir, func(p string, fi os.FileInfo, walkErr error) error {
		if walkErr != nil {
			*warnings = append(*warnings, "跳过无法读取的附件: "+p)
			return nil
		}
		if fi.IsDir() || !fi.Mode().IsRegular() {
			return nil
		}
		rel, err := filepath.Rel(absDir, p)
		if err != nil {
			return nil
		}
		if total+fi.Size() > maxUploadsBytes {
			*warnings = append(*warnings, "附件总体积超过上限，已跳过: "+filepath.ToSlash(rel))
			return nil
		}

		fr, err := os.Open(p)
		if err != nil {
			*warnings = append(*warnings, "跳过无法读取的附件: "+filepath.ToSlash(rel))
			return nil
		}
		defer fr.Close()

		hdr := &zip.FileHeader{
			Name:     uploadsPrefix + filepath.ToSlash(rel),
			Method:   zip.Deflate,
			Modified: fi.ModTime(),
		}
		fw, err := zw.CreateHeader(hdr)
		if err != nil {
			return err
		}
		n, err := io.Copy(fw, fr)
		total += n
		return err
	})
}

// ----------------------------------------------------------------------------
// 恢复
// ----------------------------------------------------------------------------

// Restore 从备份包恢复数据。
// 语义：覆盖式恢复——先清空受管表，再写入备份内容；附件为「覆盖写入」，不删除多余文件。
func Restore(zipPath, uploadsDir string, restoreUploads bool) (*Result, error) {
	// 串行化恢复操作，避免两次恢复交叉写入
	if !restoreLock.TryLock() {
		return nil, ErrRestoring
	}
	defer restoreLock.Unlock()

	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		return nil, ErrInvalidFormat
	}
	defer zr.Close()

	if len(zr.File) == 0 {
		return nil, ErrInvalidFormat
	}

	index := make(map[string]*zip.File, len(zr.File))
	for _, f := range zr.File {
		name := filepath.ToSlash(f.Name)
		if strings.HasSuffix(name, "/") {
			continue
		}
		index[name] = f
	}

	// 阶段一：读取并校验元信息
	mfFile, ok := index[manifestEntry]
	if !ok {
		return nil, ErrInvalidFormat
	}
	mf := &Manifest{}
	if err := decodeZipJSON(mfFile, mf); err != nil {
		return nil, ErrInvalidFormat
	}
	if mf.Format != FormatName {
		return nil, ErrInvalidFormat
	}
	if mf.Version > FormatVersion {
		return nil, ErrVersionTooNew
	}

	// 阶段二：完整解析所有数据表（此时不动数据库）
	parsed := make(map[string][]map[string]json.RawMessage, len(tableSpecs()))
	for _, spec := range tableSpecs() {
		f, ok := index[dataPrefix+spec.Name+".json"]
		if !ok {
			// 兼容缺表：视为空表
			continue
		}
		rows := make([]map[string]json.RawMessage, 0)
		if err := decodeZipJSON(f, &rows); err != nil {
			return nil, fmt.Errorf("数据表 %s 解析失败: %w", spec.Name, err)
		}
		parsed[spec.Name] = rows
	}

	result := &Result{
		Manifest: mf,
		Tables:   map[string]int{},
		Warnings: []string{},
	}

	// 阶段三：恢复前的前置校验
	// 注意：以下校验必须在动数据库之前全部完成。
	// 否则一旦校验失败，数据库已被清空但内容写不回去，会造成不可逆的数据丢失。
	if len(parsed["user"]) == 0 {
		return nil, ErrNoUser
	}
	if restoreUploads && uploadsDir != "" {
		if err := validateUploads(index); err != nil {
			return nil, err
		}
	}

	// 阶段四：事务内清空并写入
	err = global.Db.Transaction(func(tx *gorm.DB) error {
		// 先按子表 -> 父表清空
		specs := tableSpecs()
		for i := len(specs) - 1; i >= 0; i-- {
			if err := tx.Session(&gorm.Session{AllowGlobalUpdate: true}).Unscoped().Delete(specs[i].Model).Error; err != nil {
				return fmt.Errorf("清空数据表 %s 失败: %w", specs[i].Name, err)
			}
		}

		// 再按父表 -> 子表写入
		for _, spec := range specs {
			rows := parsed[spec.Name]
			n, warns, err := insertTable(tx, spec, rows)
			if err != nil {
				return err
			}
			result.Tables[spec.Name] = n
			result.TotalRecords += n
			result.Warnings = append(result.Warnings, warns...)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	// 阶段五：附件（路径与体积已在事务前校验，此处失败仅按警告处理）
	if restoreUploads && uploadsDir != "" {
		n, warns, err := extractUploads(index, uploadsDir)
		if err != nil {
			return nil, err
		}
		result.UploadsRestored = n
		result.Warnings = append(result.Warnings, warns...)
	}

	return result, nil
}

// validateUploads 在动数据库之前，先校验备份包内所有附件条目的路径与体积
func validateUploads(index map[string]*zip.File) error {
	count := 0
	var total int64
	for name, f := range index {
		if !strings.HasPrefix(name, uploadsPrefix) {
			continue
		}
		rel := strings.TrimPrefix(name, uploadsPrefix)
		if rel == "" || strings.HasPrefix(rel, "__MACOSX/") || path.Base(rel) == ".DS_Store" {
			continue
		}
		if _, err := safeRelPath(rel); err != nil {
			return err
		}
		if count >= maxEntries {
			return ErrTooLarge
		}
		total += int64(f.UncompressedSize64)
		if total > maxUncompressedBytes {
			return ErrTooLarge
		}
		count++
	}
	return nil
}

// insertTable 将解析好的行写入目标表
func insertTable(tx *gorm.DB, spec tableSpec, raws []map[string]json.RawMessage) (int, []string, error) {
	warnings := []string{}
	if len(raws) == 0 {
		return 0, warnings, nil
	}

	fields, err := modelFields(spec.Model)
	if err != nil {
		return 0, warnings, err
	}
	byColumn := make(map[string]*schema.Field, len(fields))
	for _, f := range fields {
		byColumn[f.DBName] = f
	}

	rows := make([]map[string]interface{}, 0, len(raws))
	unknown := map[string]bool{}
	for _, raw := range raws {
		row := make(map[string]interface{}, len(raw))
		for col, val := range raw {
			f, ok := byColumn[col]
			if !ok {
				if !unknown[col] {
					unknown[col] = true
					warnings = append(warnings, fmt.Sprintf("数据表 %s 存在当前版本不识别的字段 %s，已忽略", spec.Name, col))
				}
				continue
			}
			v, err := decodeValue(val, f.FieldType)
			if err != nil {
				return 0, warnings, fmt.Errorf("数据表 %s 字段 %s 解析失败: %w", spec.Name, col, err)
			}
			row[col] = v
		}
		if len(row) == 0 {
			continue
		}
		rows = append(rows, row)
	}

	for start := 0; start < len(rows); start += insertBatchSize {
		end := start + insertBatchSize
		if end > len(rows) {
			end = len(rows)
		}
		batch := rows[start:end]
		if err := tx.Table(spec.Name).Create(&batch).Error; err != nil {
			return 0, warnings, fmt.Errorf("写入数据表 %s 失败: %w", spec.Name, err)
		}
	}

	return len(rows), warnings, nil
}

// extractUploads 解压附件，严格校验路径
func extractUploads(index map[string]*zip.File, destDir string) (int, []string, error) {
	warnings := []string{}
	absDest, err := filepath.Abs(destDir)
	if err != nil {
		return 0, warnings, err
	}
	if err := os.MkdirAll(absDest, 0o755); err != nil {
		return 0, warnings, err
	}

	count := 0
	var total int64

	for name, f := range index {
		if !strings.HasPrefix(name, uploadsPrefix) {
			continue
		}
		rel := strings.TrimPrefix(name, uploadsPrefix)
		if rel == "" || strings.HasPrefix(rel, "__MACOSX/") || path.Base(rel) == ".DS_Store" {
			continue
		}

		safeRel, err := safeRelPath(rel)
		if err != nil {
			return count, warnings, err
		}
		target := filepath.Join(absDest, safeRel)
		if !strings.HasPrefix(target, absDest+string(os.PathSeparator)) {
			return count, warnings, ErrUnsafePath
		}

		if total+int64(f.UncompressedSize64) > maxUncompressedBytes {
			return count, warnings, ErrTooLarge
		}
		if count >= maxEntries {
			return count, warnings, ErrTooLarge
		}

		rc, err := f.Open()
		if err != nil {
			warnings = append(warnings, "跳过无法读取的附件: "+rel)
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			rc.Close()
			return count, warnings, err
		}
		out, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o644)
		if err != nil {
			rc.Close()
			warnings = append(warnings, "跳过无法写入的附件: "+rel)
			continue
		}
		n, copyErr := io.Copy(out, rc)
		out.Close()
		rc.Close()
		if copyErr != nil {
			warnings = append(warnings, "附件写入不完整: "+rel)
			continue
		}
		total += n
		count++
	}

	return count, warnings, nil
}

// ----------------------------------------------------------------------------
// 工具函数
// ----------------------------------------------------------------------------

// GetInfo 统计当前数据量，用于前端展示
func GetInfo(uploadsDir string) (*Info, error) {
	info := &Info{
		AppVersion: cmn.GetSysVersionInfo().Version,
		DbDriver:   string(global.DB_DRIVER),
		Tables:     map[string]int{},
		UploadsDir: uploadsDir,
	}

	for _, spec := range tableSpecs() {
		var count int64
		if err := global.Db.Unscoped().Model(spec.Model).Count(&count).Error; err != nil {
			return nil, err
		}
		info.Tables[spec.Name] = int(count)
	}

	if uploadsDir != "" {
		if abs, err := filepath.Abs(uploadsDir); err == nil {
			_ = filepath.Walk(abs, func(_ string, fi os.FileInfo, err error) error {
				if err != nil || fi == nil || fi.IsDir() {
					return nil
				}
				info.UploadsCount++
				info.UploadsSize += fi.Size()
				return nil
			})
		}
	}

	return info, nil
}

// FlushCaches 恢复后清空内存缓存，避免读到旧配置与旧登录态
func FlushCaches() {
	if global.SystemSetting != nil {
		global.SystemSetting.Cache.Flush()
	}
	if global.UserToken != nil {
		global.UserToken.Flush()
	}
	if global.CUserToken != nil {
		global.CUserToken.Flush()
	}
	if global.VerifyCodeCachePool != nil {
		global.VerifyCodeCachePool.Flush()
	}
}

func decodeZipJSON(f *zip.File, v interface{}) error {
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()
	dec := json.NewDecoder(io.LimitReader(rc, maxUncompressedBytes))
	return dec.Decode(v)
}

// decodeValue 按目标字段类型还原 JSON 值，保证时间/整数等类型正确
func decodeValue(raw json.RawMessage, t reflect.Type) (interface{}, error) {
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}
	ptr := reflect.New(t)
	if err := json.Unmarshal(raw, ptr.Interface()); err != nil {
		return nil, err
	}
	return ptr.Elem().Interface(), nil
}

// modelFields 解析出模型对应的数据库列
func modelFields(model interface{}) ([]*schema.Field, error) {
	stmt := &gorm.Statement{DB: global.Db}
	if err := stmt.Parse(model); err != nil {
		return nil, err
	}
	fields := make([]*schema.Field, 0, len(stmt.Schema.Fields))
	for _, f := range stmt.Schema.Fields {
		if f.DBName == "" {
			continue
		}
		if !isColumnType(f.FieldType) {
			continue
		}
		fields = append(fields, f)
	}
	return fields, nil
}

var (
	timeType    = reflect.TypeOf(time.Time{})
	scannerType = reflect.TypeOf((*sql.Scanner)(nil)).Elem()
)

// isColumnType 判断字段是否对应真实数据库列（排除关联结构体）
func isColumnType(t reflect.Type) bool {
	if t.Kind() != reflect.Struct {
		return true
	}
	if t == timeType {
		return true
	}
	return reflect.PointerTo(t).Implements(scannerType)
}

// safeRelPath 校验并归一化 zip 内的相对路径，防止目录穿越
func safeRelPath(name string) (string, error) {
	n := strings.ReplaceAll(name, "\\", "/")
	if n == "" || strings.HasPrefix(n, "/") || strings.Contains(n, ":") {
		return "", ErrUnsafePath
	}
	clean := path.Clean(n)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") {
		return "", ErrUnsafePath
	}
	return filepath.FromSlash(clean), nil
}
