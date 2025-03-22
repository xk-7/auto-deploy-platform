package models

// FileInfo 单个文件信息
type FileInfo struct {
	Name    string `json:"name" example:"file.txt"`
	IsDir   bool   `json:"is_dir" example:"false"`
	Mode    string `json:"mode" example:"rw-r--r--"`
	Size    int64  `json:"size" example:"1024"`
	ModTime string `json:"mod_time" example:"2025-03-22 12:34:56"`
}

// ListFilesResponse 文件列表响应
type ListFilesResponse struct {
	Current string     `json:"current" example:"/data"`
	Files   []FileInfo `json:"files"`
}

// FileDeleteRequest 删除文件/文件夹请求
type FileDeleteRequest struct {
	Path string `json:"path" example:"/data"`
	Name string `json:"name" example:"file.txt"`
}

// MkdirRequest 创建目录请求
type MkdirRequest struct {
	Path string `json:"path" example:"/data"`
	Name string `json:"name" example:"new-folder"`
}

// FileConfigResponse 文件配置返回
type FileConfigResponse struct {
	BaseDir    string `json:"baseDir"`
	AllowAll   bool   `json:"allowAll"`
	ApiBaseUrl string `json:"apiBaseUrl"`
}

// RenameRequest 文件重命名请求
type RenameRequest struct {
	Path    string `json:"path" example:"/data"`
	OldName string `json:"old_name" example:"file.txt"`
	NewName string `json:"new_name" example:"file-renamed.txt"`
}

// BatchDeleteRequest 批量删除请求
type BatchDeleteRequest struct {
	Path  string   `json:"path" example:"/data"`
	Names []string `json:"names" example:"[\"file1.txt\", \"folder2\"]"`
}

// FileContentResponse 文件内容返回
type FileContentResponse struct {
	Content string `json:"content"`
}

// SaveFileRequest 保存文件内容请求
type SaveFileRequest struct {
	Path    string `json:"path" example:"/data/file.txt"`
	Content string `json:"content" example:"新的文件内容"`
}

type ChmodRequest struct {
	Path string `json:"path" example:"/data/file.txt"`
	Mode string `json:"mode" example:"755"`
}

type CompressRequest struct {
	Path  string   `json:"path" example:"/data"`
	Names []string `json:"names" example:"[\"file1.txt\", \"folder\"]"`
	Type  string   `json:"type" example:"zip"` // 可选 zip/tar.gz
}

type ExtractRequest struct {
	Path string `json:"path" example:"/data"`
}

// 移动文件
type MoveFileRequest struct {
	SourcePath string `json:"source_path" example:"/tmp/file-manager/a.txt"`
	TargetDir  string `json:"target_dir" example:"/tmp/file-manager/subfolder"`
}

// 批量下载
type BatchDownloadRequest struct {
	Path  string   `json:"path" example:"/tmp/file-manager"`
	Names []string `json:"names" example:"[\"a.txt\", \"b.txt\"]"`
}

// 批量权限修改
type BatchChmodRequest struct {
	Path  string   `json:"path" example:"/tmp/file-manager"`
	Names []string `json:"names" example:"[\"file1\", \"folder1\"]"`
	Mode  string   `json:"mode" example:"755"`
}
