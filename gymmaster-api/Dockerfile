# ---- build ----
FROM golang:1.25-alpine AS build
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
USER 1001
EXPOSE 8000
# Migration/seed chạy bằng lệnh riêng: gymmaster migrate up / gymmaster seed
ENTRYPOINT ["/app/gymmaster"]
CMD ["http"]
