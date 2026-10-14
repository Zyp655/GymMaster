# gymmaster-api

Backend của GymMaster, viết bằng **Go 1.25**, **GoFrame v2**, **GORM** và **PostgreSQL 16**. Kiến trúc lấy theo `strongbody-api` và có bổ sung một số cải tiến (xem `../docs/PLAN.md` §3.2).

## Chạy nhanh

```bash
cp .env.example .env                 # nhớ sửa JWT_SECRET
docker compose -f ../docker-compose.yml up -d postgres
make migrate-up                      # tạo schema
make seed                            # admin@gymmaster.vn / Admin@123 + 3 chi nhánh mẫu
make run                             # http://localhost:8000
```

- Swagger UI: http://localhost:8000/swagger
- OpenAPI JSON: http://localhost:8000/api.json
- Health check: http://localhost:8000/health

Gõ `make help` để xem tất cả các lệnh.

## Cấu trúc thư mục

```
main.go                      → internal/cmd (lệnh: http | migrate up/down/version | seed)
api/<module>/v1/*.go         Req/Res + g.Meta (path, method, tags, summary, perm) = route + swagger + validate
api/<module>/<module>.go     interface I<Module>V1 (sinh bằng `make ctrl`)
internal/
  controller/<module>/       <module>_new.go + <module>_v1_<action>.go (mỏng, chỉ gọi service)
  service/                   nghiệp vụ; chỉ phụ thuộc interface repository và context
  repository/                truy cập DB qua GORM; lấy kết nối bằng postgres.Conn(ctx, db) để tham gia transaction
  entity/                    model GORM (BaseEntity: id, timestamps, soft delete, created_by/updated_by)
  dto/                       PageReq/PageRes, Map
  middleware/                CORS, RequestLog, Response (envelope), Authentication (JWT), Authorization (perm)
  module/                    lắp ráp repo → service → controller cho từng module + đăng ký route
  router/                    các nhóm /v1/auth, /v1/admin, ...
  storage/postgres/          Open, TxManager
  pkg/                       token (JWT), hash (bcrypt, sha256), rescode (mã lỗi), reqctx (user hiện tại)
  seed/                      dữ liệu khởi tạo (chạy lại nhiều lần vẫn an toàn)
migrations/                  golang-migrate (nhúng vào binary)
```

## Quy ước

### Thêm một API
1. Khai báo Req/Res trong `api/<module>/v1`.
   - Thông tin route (`path`, `method`, `tags`, `summary`) và quyền yêu cầu (`perm:"<module>.<action>"`) đặt trong tag `g.Meta`.
   - Quy tắc validate đặt trong tag `v:"rule1|rule2#Thông báo 1|Thông báo 2"`, thông báo viết bằng tiếng Việt.
2. Viết repository (interface + struct private + `NewXRepository`) và service (interface + struct + `NewXService`).
3. Controller gọi service. Chạy `make ctrl` để sinh interface, hoặc tự viết theo mẫu có sẵn.
4. Đăng ký module trong `internal/module/<module>.go` và gọi hàm đó trong `internal/router/router.go`.
5. Viết unit test cho service, dùng repository giả lập (fake).

### Response
Mọi response đều có dạng `{code, message, data}`:
- `code = 0` là thành công.
- Lỗi nghiệp vụ dùng mã trong `pkg/rescode`, và HTTP status = `code / 100`.
- Lỗi validate có `code = 51`, trả HTTP 400.
- Lỗi 5xx không trả chi tiết nội bộ ra ngoài; chi tiết chỉ được ghi vào log.

### Phân quyền
- `Authentication` kiểm tra JWT rồi đặt payload (`user`, `roles`, `perms`, `branches`) vào context.
- `Authorization` đọc tag `perm` của API đang được gọi. Role `OWNER` luôn được phép.
- Trong service, lấy người dùng hiện tại bằng `reqctx.Auth(ctx)` hoặc `reqctx.UserID(ctx)`.

### Database
- Không dùng AutoMigrate. Mọi thay đổi schema phải qua migration: `make migrate-create name=...`, viết cả file `up` và `down`.
- Mã nghiệp vụ (code) là unique, nhưng chỉ tính trên các bản ghi chưa bị xóa mềm (partial unique index).
- Nghiệp vụ ghi vào nhiều bảng phải bọc trong `TxManager.WithTx`.

### Refresh token
- Lưu trong DB dưới dạng SHA-256, không lưu bản gốc.
- Mỗi lần refresh sẽ cấp token mới và thu hồi token cũ (xoay vòng).
- Nếu một token đã bị thu hồi mà vẫn được dùng lại, hệ thống thu hồi toàn bộ phiên đăng nhập của user đó (phát hiện token bị đánh cắp).
