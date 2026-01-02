package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"

	"github.com/na2na-p/cargohold/internal/config"
	"github.com/na2na-p/cargohold/internal/domain"
	"github.com/na2na-p/cargohold/internal/handler"
	"github.com/na2na-p/cargohold/internal/handler/auth"
	authMiddleware "github.com/na2na-p/cargohold/internal/handler/middleware"
	"github.com/na2na-p/cargohold/internal/infrastructure"
	"github.com/na2na-p/cargohold/internal/infrastructure/logging"
	"github.com/na2na-p/cargohold/internal/infrastructure/oidc"
	"github.com/na2na-p/cargohold/internal/infrastructure/postgres"
	"github.com/na2na-p/cargohold/internal/infrastructure/redis"
	"github.com/na2na-p/cargohold/internal/infrastructure/s3"
	infraurl "github.com/na2na-p/cargohold/internal/infrastructure/url"
	"github.com/na2na-p/cargohold/internal/usecase"
)

const (
	defaultPort     = "8080"
	readTimeout     = 30 * time.Second
	writeTimeout    = 30 * time.Second
	shutdownTimeout = 10 * time.Second
	idleTimeout     = 120 * time.Second
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level:       slog.LevelInfo,
		ReplaceAttr: logging.MaskSensitiveAttrs,
	}))
	slog.SetDefault(logger)

	if err := run(); err != nil {
		slog.Error("application failed", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	pool, err := postgres.NewPostgresConnection(postgres.PostgresConfig{
		Host:     cfg.Database.Host,
		Port:     cfg.Database.Port,
		User:     cfg.Database.User,
		Password: cfg.Database.Password,
		Database: cfg.Database.DBName,
		SSLMode:  cfg.Database.SSLMode,
	})
	if err != nil {
		return err
	}
	defer pool.Close()
	slog.Info("PostgreSQL connection established")

	redisConn, err := redis.NewRedisConnection(redis.RedisConfig{
		Host:     cfg.Redis.Host,
		Port:     cfg.Redis.Port,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	if err != nil {
		return err
	}
	defer func() { _ = redisConn.Close() }()
	redisClient := redis.NewRedisClient(redisConn)
	slog.Info("Redis connection established")

	s3Conn, err := s3.NewS3Connection(s3.S3Config{
		Endpoint:        cfg.S3.Endpoint,
		AccessKeyID:     cfg.S3.AccessKeyID,
		SecretAccessKey: cfg.S3.SecretAccessKey,
		Region:          cfg.S3.Region,
	})
	if err != nil {
		return err
	}
	s3Client := s3.NewS3Client(s3Conn, cfg.S3.BucketName)
	slog.Info("S3 connection established")

	lfsRepo := postgres.NewLFSObjectRepository(pool)
	policyRepo := postgres.NewAccessPolicyRepository(pool)
	repoAllowlistRepo := postgres.NewRepositoryAllowlistRepository(pool)

	var githubProvider *oidc.GitHubOIDCProvider
	if cfg.OIDC.GitHub.Enabled {
		githubProvider, err = oidc.NewGitHubOIDCProvider(
			cfg.OIDC.GitHub.Audience,
			redisClient,
			cfg.OIDC.GitHub.JWKSURL,
		)
		if err != nil {
			return err
		}
		slog.Info("GitHub OIDC provider initialized")
	}

	var githubOAuthUC *usecase.GitHubOAuthUseCase
	if cfg.OAuth.GitHub.Enabled {
		githubOAuthProvider, err := oidc.NewGitHubOAuthProvider(
			cfg.OAuth.GitHub.ClientID,
			cfg.OAuth.GitHub.ClientSecret,
			"",
		)
		if err != nil {
			return err
		}

		oauthProviderAdapter := oidc.NewGitHubOAuthProviderAdapter(githubOAuthProvider)
		oauthStateStore := redis.NewOAuthStateStore(redisClient)
		sessionStoreAdapter := redis.NewSessionStoreAdapterWithDefaults(redisClient)

		allowedRedirectURIs, err := domain.NewAllowedRedirectURIs(cfg.OAuth.GitHub.AllowedRedirectURIs)
		if err != nil {
			return fmt.Errorf("failed to create AllowedRedirectURIs: %w", err)
		}

		githubOAuthUC, err = usecase.NewGitHubOAuthUseCase(
			oauthProviderAdapter,
			sessionStoreAdapter,
			oauthStateStore,
			allowedRedirectURIs,
		)
		if err != nil {
			return err
		}
		slog.Info("GitHub OAuth provider initialized")
	}

	cacheKeyGenerator := redis.NewCacheKeyGenerator()
	cacheConfig := redis.NewCacheConfig()
	storageKeyGenerator := s3.NewStorageKeyGenerator()
	proxyActionURLGenerator := infraurl.NewProxyActionURLGenerator()

	cachingRepo := infrastructure.NewCachingLFSObjectRepository(
		lfsRepo,
		redisClient,
		cacheKeyGenerator,
		cacheConfig,
	)

	cachingRepoAllowlist := infrastructure.NewCachingRepositoryAllowlist(repoAllowlistRepo, redisClient)
	authUC := usecase.NewAuthUseCase(githubProvider, cachingRepoAllowlist, redisClient, cacheKeyGenerator)
	accessAuthService := domain.NewAccessAuthorizationService(policyRepo)
	batchUC := usecase.NewBatchUseCase(cachingRepo, proxyActionURLGenerator, policyRepo, storageKeyGenerator, accessAuthService)
	verifyUC := usecase.NewVerifyUseCase(cachingRepo, cachingRepo)
	proxyUploadUC := usecase.NewProxyUploadUseCase(cachingRepo, s3Client, accessAuthService)
	proxyDownloadUC := usecase.NewProxyDownloadUseCase(cachingRepo, s3Client, accessAuthService)
	storageErrorChecker := s3.NewStorageErrorChecker()
	proxyHandler := handler.NewProxyHandler(proxyUploadUC, proxyDownloadUC, storageErrorChecker, cfg.Server.ProxyTimeout)

	postgresHealthChecker := postgres.NewPostgresHealthChecker(pool)
	redisHealthChecker := redis.NewRedisHealthChecker(redisClient)
	s3HealthChecker := s3.NewS3HealthChecker(s3Client)

	readinessUC := usecase.NewReadinessUseCase(
		postgresHealthChecker,
		redisHealthChecker,
		s3HealthChecker,
	)

	e := echo.New()
	e.HideBanner = true
	e.HidePort = true
	e.HTTPErrorHandler = authMiddleware.CustomHTTPErrorHandler

	ipExtractor, err := buildIPExtractor(cfg.Server.TrustedProxyCIDRs)
	if err != nil {
		return err
	}
	e.IPExtractor = ipExtractor

	e.Use(middleware.Recover())
	e.Use(middleware.RequestID())
	e.Use(middleware.RequestLoggerWithConfig(middleware.RequestLoggerConfig{
		LogStatus:   true,
		LogURI:      true,
		LogMethod:   true,
		LogLatency:  true,
		LogError:    true,
		HandleError: true,
		LogValuesFunc: func(c echo.Context, v middleware.RequestLoggerValues) error {
			attrs := []slog.Attr{
				slog.String("method", v.Method),
				slog.String("uri", v.URI),
				slog.Int("status", v.Status),
				slog.Duration("latency", v.Latency),
			}
			if v.Error != nil {
				attrs = append(attrs, slog.String("error", v.Error.Error()))
				slog.LogAttrs(c.Request().Context(), slog.LevelError, "REQUEST", attrs...)
			} else {
				slog.LogAttrs(c.Request().Context(), slog.LevelInfo, "REQUEST", attrs...)
			}
			return nil
		},
	}))

	e.GET("/healthz", handler.HealthHandler)

	readyzHandler := handler.NewReadyzHandler(readinessUC)
	e.GET("/readyz", readyzHandler.Handle)

	batchHandler := handler.NewBatchHandler(batchUC)
	authDispatcher := authMiddleware.AuthDispatcher(authUC)

	lfsGroup := e.Group("/:owner/:repo/info/lfs")
	lfsGroup.Use(authDispatcher)
	lfsGroup.POST("/objects/batch", batchHandler.Handle)
	lfsGroup.POST("/objects/verify", handler.VerifyHandler(verifyUC))
	lfsGroup.PUT("/objects/:oid", proxyHandler.HandleUpload)
	lfsGroup.GET("/objects/:oid", proxyHandler.HandleDownload)

	if githubOAuthUC != nil {
		loginHandlerConfig := auth.GitHubLoginHandlerConfig{
			TrustProxy:   cfg.Server.TrustProxy,
			AllowedHosts: cfg.OAuth.GitHub.AllowedHosts,
		}
		authGroup := e.Group("/auth/github")
		authGroup.GET("/login", auth.GitHubLoginHandler(githubOAuthUC, loginHandlerConfig))
		authGroup.GET("/callback", auth.GitHubCallbackHandler(githubOAuthUC))
		slog.Info("GitHub OAuth routes registered")
	}

	e.GET("/auth/session", auth.SessionDisplayHandler())

	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

	server := &http.Server{
		Addr:         ":" + port,
		ReadTimeout:  readTimeout,
		WriteTimeout: writeTimeout,
		IdleTimeout:  idleTimeout,
	}

	errChan := make(chan error, 1)
	go func() {
		slog.Info("starting server", "port", port)
		if err := e.StartServer(server); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errChan <- err
		}
		close(errChan)
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-quit:
		slog.Info("received shutdown signal")
	case err := <-errChan:
		if err != nil {
			return err
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	slog.Info("shutting down server")
	if err := e.Shutdown(ctx); err != nil {
		return err
	}

	slog.Info("server stopped gracefully")
	return nil
}

// buildIPExtractor は設定に基づいてIPエクストラクタを構築する。
// 信頼するプロキシのCIDRが指定されている場合、そのCIDRからのX-Forwarded-Forヘッダーのみを信頼する。
// 指定されていない場合、IPスプーフィング防止のため接続元IPを直接使用する。
func buildIPExtractor(trustedProxyCIDRs []string) (echo.IPExtractor, error) {
	if len(trustedProxyCIDRs) == 0 {
		slog.Info("trusted proxy CIDRs not configured, using direct IP extraction")
		return echo.ExtractIPDirect(), nil
	}

	trustOptions := make([]echo.TrustOption, 0, len(trustedProxyCIDRs))
	for _, cidr := range trustedProxyCIDRs {
		_, ipNet, err := net.ParseCIDR(cidr)
		if err != nil {
			return nil, fmt.Errorf("invalid trusted proxy CIDR %q: %w", cidr, err)
		}
		trustOptions = append(trustOptions, echo.TrustIPRange(ipNet))
	}

	slog.Info("trusted proxy CIDRs configured", "cidrs", trustedProxyCIDRs)
	return echo.ExtractIPFromXFFHeader(trustOptions...), nil
}
