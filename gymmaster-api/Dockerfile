# ---- build ----
# Phiên bản Go phải >= dòng `go` trong go.mod (hiện 1.26.0).
FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /out/gymmaster .

# ---- runtime ----
FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata && adduser -D -u 1001 app
ENV TZ=Asia/Ho_Chi_Minh
WORKDIR /app
COPY --from=build /out/gymmaster /app/gymmaster
# Thư mục upload (UPLOAD_DIR mặc định resource/uploads) phải thuộc user 1001: khi mount named volume
# rỗng vào đây lần đầu, Docker sao chép quyền sở hữu của thư mục trong image sang volume.
RUN mkdir -p /app/resource/uploads && chown -R 1001:1001 /app/resource
USER 1001
EXPOSE 8000
# Migration/seed chạy bằng lệnh riêng: gymmaster migrate up / gymmaster seed
ENTRYPOINT ["/app/gymmaster"]
CMD ["http"]
