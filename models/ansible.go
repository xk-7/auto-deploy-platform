package models

// RunAnsibleRequest 执行 Ansible 请求
type RunAnsibleRequest struct {
	Inventory string `json:"inventory" example:"/etc/ansible/hosts"`
	Playbook  string `json:"playbook" example:"site.yml"`
}
