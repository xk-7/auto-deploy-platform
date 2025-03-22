package models

// ListComposeResponse Compose 应用列表响应
type ListComposeResponse struct {
	Apps []string `json:"apps" example:"[\"app1\", \"app2\"]"`
}

// ComposeContainerInfo 单个容器信息
type ComposeContainerInfo struct {
	ID     string `json:"id" example:"a1b2c3d4e5f6"`
	Name   string `json:"name" example:"/app1-web"`
	Image  string `json:"image" example:"nginx:latest"`
	Status string `json:"status" example:"Up 3 minutes"`
	Ports  string `json:"ports" example:"8080:80 443:443"`
}

// ComposeStatusResponse Compose 应用状态响应
type ComposeStatusResponse struct {
	Apps map[string][]ComposeContainerInfo `json:"apps"`
}

// ComposeActionRequest Compose 应用操作请求
type ComposeActionRequest struct {
	Name string `json:"name" example:"my-app"`
}
