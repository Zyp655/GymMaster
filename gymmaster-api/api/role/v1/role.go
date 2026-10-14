package v1

import "github.com/gogf/gf/v2/frame/g"

type Role struct {
	ID          uint     `json:"id"`
	Code        string   `json:"code"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	IsSystem    bool     `json:"is_system"`
	Permissions []string `json:"permissions"`
}

type Permission struct {
	ID     uint   `json:"id"`
	Code   string `json:"code"`
	Module string `json:"module"`
	Name   string `json:"name"`
}

type ListPermissionsReq struct {
	g.Meta `path:"/permissions" method:"get" tags:"Vai trò & quyền" summary:"Danh sách quyền" perm:"role.view"`
}
type ListPermissionsRes []*Permission

type ListReq struct {
	g.Meta `path:"/roles" method:"get" tags:"Vai trò & quyền" summary:"Danh sách vai trò" perm:"role.view"`
}
type ListRes []*Role

type GetReq struct {
	g.Meta `path:"/roles/{id}" method:"get" tags:"Vai trò & quyền" summary:"Chi tiết vai trò" perm:"role.view"`
	ID     uint `json:"id" in:"path" v:"required|min:1"`
}
type GetRes = Role

type CreateReq struct {
	g.Meta      `path:"/roles" method:"post" tags:"Vai trò & quyền" summary:"Thêm vai trò" perm:"role.create"`
	Code        string   `json:"code"        v:"required|regex:^[A-Z][A-Z0-9_]{1,49}$#Vui lòng nhập mã vai trò|Mã gồm chữ in hoa, số, '_' và bắt đầu bằng chữ"`
	Name        string   `json:"name"        v:"required|max-length:100#Vui lòng nhập tên vai trò|Tên tối đa 100 ký tự"`
	Description string   `json:"description" v:"max-length:255#Mô tả tối đa 255 ký tự"`
	Permissions []string `json:"permissions"`
}
type CreateRes = Role

type UpdateReq struct {
	g.Meta      `path:"/roles/{id}" method:"put" tags:"Vai trò & quyền" summary:"Cập nhật vai trò và phân quyền" perm:"role.update"`
	ID          uint     `json:"id"          in:"path" v:"required|min:1"`
	Name        string   `json:"name"        v:"required|max-length:100#Vui lòng nhập tên vai trò|Tên tối đa 100 ký tự"`
	Description string   `json:"description" v:"max-length:255#Mô tả tối đa 255 ký tự"`
	Permissions []string `json:"permissions"`
}
type UpdateRes = Role

type DeleteReq struct {
	g.Meta `path:"/roles/{id}" method:"delete" tags:"Vai trò & quyền" summary:"Xóa vai trò" perm:"role.delete"`
	ID     uint `json:"id" in:"path" v:"required|min:1"`
}
type DeleteRes struct{}
