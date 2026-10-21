package repository

import "strings"

// likeEscaper thoát ký tự đại diện của LIKE/ILIKE (ký tự thoát mặc định của PostgreSQL là '\').
var likeEscaper = strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)

// likeContains trả mẫu "%kw%" đã thoát % _ \ – tìm "50%" hay "a_b" khớp đúng ký tự, không phải đại diện.
func likeContains(kw string) string {
	return "%" + likeEscaper.Replace(kw) + "%"
}
