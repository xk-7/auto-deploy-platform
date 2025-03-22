package models

// ContainerInfo 容器信息
type ContainerInfo struct {
	ID      string `json:"id" example:"a1b2c3d4e5f6"`
	Name    string `json:"name" example:"/my-container"`
	Status  string `json:"status" example:"Up 2 hours"`
	Image   string `json:"image" example:"nginx:latest"`
	Created int64  `json:"created" example:"1678901234"`
}

// CreateContainerRequest 创建容器请求
type CreateContainerRequest struct {
	Name    string `json:"name" example:"my-container"`
	Image   string `json:"image" example:"nginx:latest"`
	Ports   string `json:"ports" example:"8080:80,8443:443"` // hostPort:containerPort
	Volumes string `json:"volumes" example:"/host/path:/container/path"`
	Envs    string `json:"envs" example:"ENV_VAR1=value1,ENV_VAR2=value2"`
	CPU     string `json:"cpu" example:"0.5"`        // 单位核
	Memory  string `json:"memory" example:"512m"`    // 单位 m/g
	Restart string `json:"restart" example:"always"` // Restart 策略
	Network string `json:"network" example:"bridge"` // host/bridge
}
