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
