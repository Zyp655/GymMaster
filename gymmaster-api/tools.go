//go:build tools

// Giữ sẵn các thư viện các module sẽ dùng trong go.mod (để `go mod tidy` không xóa
// khi chưa có code import). Xóa dòng tương ứng khi code thật đã import.
package main

import (
	_ "github.com/google/uuid"
)
