package cmd

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/net/ghttp"
	"github.com/gogf/gf/v2/net/goai"
	"github.com/gogf/gf/v2/os/gcmd"

	"gymmaster-api/internal/config"
	"gymmaster-api/internal/cron"
	"gymmaster-api/internal/middleware"
	"gymmaster-api/internal/module"
	"gymmaster-api/internal/pkg/token"
	"gymmaster-api/internal/repository"
	"gymmaster-api/internal/router"
	"gymmaster-api/internal/seed"
	"gymmaster-api/internal/service"
	"gymmaster-api/internal/storage/postgres"
	"gymmaster-api/migrations"
	"gymmaster-api/resource/i18n"
)

var (
	// Main: `gymmaster` hoặc `gymmaster http` – chạy HTTP server.
	Main = gcmd.Command{
		Name:  "gymmaster",
		Usage: "gymmaster [http|migrate|seed]",
		Brief: "GymMaster API – hệ thống quản lý trung tâm Fitness",
		Func:  serve,
	}

	HTTP = gcmd.Command{
		Name:  "http",
		Brief: "Chạy HTTP server",
		Func:  serve,
	}

	// Migrate: `gymmaster migrate up | down [steps] | version`.
	Migrate = gcmd.Command{
		Name:  "migrate",
		Usage: "gymmaster migrate up | down [steps] | version",
		Brief: "Quản lý schema bằng golang-migrate",
	}

	MigrateUp = gcmd.Command{
		Name:  "up",
		Brief: "Áp dụng tất cả migration chưa chạy",
		Func: func(ctx context.Context, parser *gcmd.Parser) error {
			dsn := config.GetConfig().DB.DSN()
			if err := migrations.Up(dsn); err != nil {
				return err
			}
			return printVersion(ctx, dsn)
		},
	}

	MigrateDown = gcmd.Command{
		Name:  "down",
		Usage: "gymmaster migrate down [steps]",
		Brief: "Rollback [steps] migration gần nhất (mặc định 1)",
		Arguments: []gcmd.Argument{
			{Name: "steps", IsArg: true, Brief: "Số migration cần rollback"},
		},
		Func: func(ctx context.Context, parser *gcmd.Parser) error {
			dsn := config.GetConfig().DB.DSN()
			steps := 1
			if n, err := strconv.Atoi(parser.GetArg(3).String()); err == nil && n > 0 {
				steps = n
			}
			if err := migrations.Down(dsn, steps); err != nil {
				return err
			}
			return printVersion(ctx, dsn)
		},
	}

	MigrateVersion = gcmd.Command{
		Name:  "version",
		Brief: "Xem version schema hiện tại",
		Func: func(ctx context.Context, parser *gcmd.Parser) error {
			return printVersion(ctx, config.GetConfig().DB.DSN())
		},
	}

	// Seed: `gymmaster seed` – tạo tài khoản OWNER + chi nhánh mẫu.
	Seed = gcmd.Command{
		Name:  "seed",
		Brief: "Tạo dữ liệu khởi tạo (tài khoản OWNER, chi nhánh mẫu)",
		Func: func(ctx context.Context, parser *gcmd.Parser) error {
			cfg := config.GetConfig()
			db, err := postgres.Open(cfg.DB)
			if err != nil {
				return err
			}
			return seed.Run(ctx, db, cfg)
		},
	}
)

func init() {
	if err := Migrate.AddCommand(&MigrateUp, &MigrateDown, &MigrateVersion); err != nil {
		panic(err)
	}
	if err := Main.AddCommand(&HTTP, &Migrate, &Seed); err != nil {
		panic(err)
	}
}

func printVersion(ctx context.Context, dsn string) error {
	v, dirty, err := migrations.Version(dsn)
	if err != nil {
		return err
	}
	g.Log().Infof(ctx, "migrate: version=%d dirty=%v", v, dirty)
	return nil
}

func serve(ctx context.Context, parser *gcmd.Parser) error {
	cfg := config.GetConfig()

	// Thông báo lỗi validate mặc định (gvalid) bằng tiếng Việt – file ngôn ngữ nhúng từ resource/i18n.
	if err := i18n.Init(); err != nil {
		g.Log().Warningf(ctx, "i18n: không nạp được thông báo tiếng Việt: %v", err)
	}

	db, err := postgres.Open(cfg.DB)
	if err != nil {
		return fmt.Errorf("kết nối database: %w", err)
	}
	tokenMaker, err := token.NewJWTMaker(cfg.JWT.Secret, cfg.JWT.Issuer, cfg.JWT.AccessTTL)
	if err != nil {
		return err
	}
	// CORS so khớp chính xác origin (scheme + host + cổng) → đọc nguyên chuỗi CORS_ORIGINS (config chỉ giữ host).
	corsOrigins := middleware.ParseCORSOrigins(os.Getenv("CORS_ORIGINS"))
	if len(corsOrigins) == 0 {
		corsOrigins = cfg.Server.CorsOrigins
	}
	mw := middleware.New(tokenMaker, corsOrigins)

	// Môi trường production không công khai tài liệu API (/swagger, /api.json).
	production := cfg.IsProduction()
	s := g.Server()
	s.SetPort(cfg.Server.Port)
	s.SetDumpRouterMap(!production)
	if !production {
		s.SetOpenApiPath("/api.json")
		s.SetSwaggerPath("/swagger")
		enhanceOpenAPI(s)
		// Swagger UI do GoFrame phục vụ bằng hook HookBeforeServe (chạy trước middleware) nên không nhận
		// SecurityHeaders → gắn nosniff + Referrer-Policy ở hook riêng (đăng ký trước hook của GoFrame).
		// Không đặt X-Frame-Options/CSP vì UI cần script/CSS từ CDN.
		s.BindHookHandler("/swagger/*", ghttp.HookBeforeServe, func(r *ghttp.Request) {
			h := r.Response.Header()
			h.Set("X-Content-Type-Options", "nosniff")
			h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
		})
	}

	s.Use(mw.CORS(), mw.SecurityHeaders(), mw.RequestLog(), mw.Response())
	// File tải lên (ảnh hội viên, sản phẩm...) – module upload ghi vào thư mục này.
	if err := os.MkdirAll(cfg.Upload.Dir, 0o755); err != nil {
		return fmt.Errorf("tạo thư mục upload: %w", err)
	}
	s.AddStaticPath("/uploads", cfg.Upload.Dir)
	s.BindHandler("GET:/health", func(r *ghttp.Request) {
		r.Response.WriteJson(g.Map{"status": "ok"})
	})

	// Bán hàng: InvoiceService dùng chung (tạo MỘT lần) + SubscriptionService (đăng ký handler MEMBERSHIP/FEE).
	salesTx := postgres.NewTxManager(db)
	promotions := service.NewPromotionService(repository.NewPromotionRepository(db), service.NewAuditService(db))
	invoices := service.NewInvoiceService(repository.NewInvoiceRepository(db), promotions, salesTx, service.NewAuditService(db))
	subscriptions := service.NewSubscriptionService(repository.NewSubscriptionRepository(db), invoices, salesTx,
		service.NewAuditService(db), service.NewNotifier(db))

	router.Register(s, &module.Deps{
		Config:     cfg,
		DB:         db,
		Tx:         postgres.NewTxManager(db),
		TokenMaker: tokenMaker,
		MW:         mw,
		Settings:   service.NewSettingService(db),
		Audit:      service.NewAuditService(db),
		Notifier:   service.NewNotifier(db),

		Promotions:    promotions,
		Invoices:      invoices,
		Subscriptions: subscriptions,
	})

	// Job định kỳ: các module đã cron.Register(...) trong router.Register ở trên. Chỉ bật khi CRON_ENABLED=true.
	if cfg.Cron.Enabled {
		if err := cron.Default(db).Start(ctx); err != nil {
			return err
		}
		defer cron.Default(db).Stop()
	} else {
		g.Log().Info(ctx, "cron: scheduler tắt (CRON_ENABLED=false) – chỉ chạy job thủ công qua API")
	}

	s.Run()
	return nil
}

// enhanceOpenAPI khai báo envelope chung và Bearer auth cho Swagger UI.
func enhanceOpenAPI(s *ghttp.Server) {
	oai := s.GetOpenApi()
	oai.Info.Title = "GymMaster API"
	oai.Info.Description = "API hệ thống số hóa quy trình quản lý trung tâm Fitness"
	oai.Info.Version = "v1"
	oai.Config.CommonResponse = middleware.DefaultHandlerResponse{}
	oai.Config.CommonResponseDataField = "Data"
	oai.Components.SecuritySchemes = goai.SecuritySchemes{
		"bearerAuth": goai.SecuritySchemeRef{Value: &goai.SecurityScheme{Type: "http", Scheme: "bearer", BearerFormat: "JWT"}},
	}
	oai.Security = &goai.SecurityRequirements{{"bearerAuth": {}}}
}
