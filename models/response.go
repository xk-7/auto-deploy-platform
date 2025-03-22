package models

// ErrorResponse 通用错误响应
type ErrorResponse struct {
	Code    int    `json:"code" example:"500"`
	Message string `json:"message" example:"Internal Server Error"`
}

// ContainerListResponse 成功返回的容器列表
type ContainerListResponse struct {
	Containers []ContainerInfo `json:"containers"`
}

// SuccessResponse 通用成功响应
type SuccessResponse struct {
	Code    int    `json:"code" example:"200"`
	Message string `json:"message" example:"Container stopped successfully"`
}

// CreateContainerResponse 创建容器成功响应
type CreateContainerResponse struct {
	Code    int    `json:"code" example:"200"`
	Message string `json:"message" example:"Container created"`
	ID      string `json:"id" example:"a1b2c3d4e5f6"`
}
