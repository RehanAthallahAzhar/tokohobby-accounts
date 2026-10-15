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
	"strings"
	"syscall"
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/RehanAthallahAzhar/tokohobby-accounts/db"
	"github.com/RehanAthallahAzhar/tokohobby-accounts/internal/configs"
	dbGenerated "github.com/RehanAthallahAzhar/tokohobby-accounts/internal/db"
	"github.com/RehanAthallahAzhar/tokohobby-accounts/internal/handlers"
	"github.com/RehanAthallahAzhar/tokohobby-accounts/internal/helpers"
	"github.com/RehanAthallahAzhar/tokohobby-accounts/internal/models"
	"github.com/RehanAthallahAzhar/tokohobby-accounts/internal/pkg/logger"
	"github.com/RehanAthallahAzhar/tokohobby-accounts/internal/pkg/redisclient"
	"github.com/RehanAthallahAzhar/tokohobby-accounts/internal/repositories"
	"github.com/RehanAthallahAzhar/tokohobby-accounts/internal/routes"
	"github.com/RehanAthallahAzhar/tokohobby-accounts/internal/services"
	"github.com/RehanAthallahAzhar/tokohobby-accounts/internal/services/token"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"

	grpcServer "github.com/RehanAthallahAzhar/tokohobby-accounts/internal/grpc"
	middlewareApp "github.com/RehanAthallahAzhar/tokohobby-accounts/internal/middlewares"
	accountpb "github.com/RehanAthallahAzhar/tokohobby-protos/pb/account"
	authpb "github.com/RehanAthallahAzhar/tokohobby-protos/pb/auth"

	"github.com/RehanAthallahAzhar/tokohobby-accounts/internal/messaging/rabbitmq"
	"github.com/RehanAthallahAzhar/tokohobby-accounts/internal/pkg/tracer"
	"github.com/RehanAthallahAzhar/tokohobby-messaging/kafka"
	rabbitmqpkg "github.com/RehanAthallahAzhar/tokohobby-messaging/rabbitmq"

	_ "github.com/RehanAthallahAzhar/tokohobby-accounts/docs"

	echootel "github.com/labstack/echo-otel/v4"
)

// @title TokoHobby Accounts API
// @version 1.0
// @description Microservice untuk manajemen akun dan autentikasi TokoHobby.
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.email support@tokohobby.com

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host localhost:8080
// @BasePath /api/v1

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Masukkan token JWT dengan format: Bearer {token}
func main() {
	log := logger.NewLogger()

	cfg, err := configs.LoadConfig(log)
	if err != nil {
		log.Error("Failed to load configuration", slog.Any("error", err))
		os.Exit(1)
	}

	dbCredential := models.Credential{
		Host:         cfg.Database.Host,
		Username:     cfg.Database.User,
		Password:     cfg.Database.Password,
		DatabaseName: cfg.Database.Name,
		Port:         cfg.Database.Port,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Setup DB
	conn, err := db.Connect(ctx, &dbCredential)
	if err != nil {
		log.Error("DB connection error", slog.Any("error", err))
		os.Exit(1)
	}

	defer func() {
		if err := conn.Close(); err != nil {
			log.Error("Failed to close DB connection cleanly", slog.Any("error", err))
		}
	}()

	connectionString := fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=disable",
		dbCredential.Username,
		dbCredential.Password,
		dbCredential.Host,
		dbCredential.Port,
		dbCredential.DatabaseName,
	)

	m, err := migrate.New(
		cfg.Migration.Path,
		connectionString,
	)
	if err != nil {
		log.Error("Failed to create migrate instance", slog.Any("error", err))
		os.Exit(1)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		log.Error("Failed to run migrations", slog.Any("error", err))
		os.Exit(1)
	}

	// Init SQLC
	sqlcQueries := dbGenerated.New(conn)

	//jwt
	_ = helpers.NewJWTHelper(cfg.Server.JWTSecret)

	// Setup Redis
	redisClient, err := redisclient.NewRedisClient(&cfg.Redis, log)
	if err != nil {
		log.Error("Failed to initialize redis client", slog.Any("error", err))
		os.Exit(1)
	}
	defer redisClient.Close()

	// Initialize RabbitMQ
	rmqConfig := &rabbitmqpkg.RabbitMQConfig{
		URL:            cfg.RabbitMQ.URL,
		MaxRetries:     cfg.RabbitMQ.MaxRetries,
		RetryDelay:     cfg.RabbitMQ.RetryDelay,
		PrefetchCount:  cfg.RabbitMQ.PrefetchCount,
		ReconnectDelay: cfg.RabbitMQ.ReconnectDelay,
	}
	rmq, err := rabbitmqpkg.NewRabbitMQ(rmqConfig)
	if err != nil {
		log.Error("Failed to connect to RabbitMQ", slog.Any("error", err))
		os.Exit(1)
	}

	defer func() {
		if err := rmq.Close(); err != nil {
			log.Error("Failed to close RabbitMQ connection cleanly", slog.Any("error", err))
		}
	}()

	// Setup exchange
	if err := rabbitmqpkg.SetupUserExchange(rmq); err != nil {
		log.Error("Failed to setup user exchange", slog.Any("error", err))
		os.Exit(1)
	}

	// Create event publisher (RabbitMQ)
	eventPublisher := rabbitmq.NewEventPublisher(rmq, log)

	// kafka producer for user activity tracking
	var kafkaProducer *kafka.ActivityProducer
	if cfg.Kafka.Brokers != "" {
		kafkaBrokers := strings.Split(cfg.Kafka.Brokers, ",")
		kafkaProducer = kafka.NewActivityProducer(kafkaBrokers)

		defer func() {
			if err := kafkaProducer.Close(); err != nil {
				log.Error("Failed to close kafka producer cleanly", slog.Any("error", err))
			}
		}()

		log.Info("Kafka producer initialized", slog.Any("brokers", kafkaBrokers))
	} else {
		log.Warn("Kafka brokers not configured, user activity tracking disabled")
	}

	// Initialize Tracer
	samplerRatio := cfg.Tracer.SamplerRatio
	shutdownTracer, err := tracer.InitTracer(ctx, tracer.Config{
		ServiceName:    cfg.Tracer.ServiceName,
		ServiceVersion: cfg.Tracer.ServiceVersion,
		Environment:    cfg.Tracer.Environment,
		Endpoint:       cfg.Tracer.Endpoint,
		Insecure:       cfg.Tracer.Insecure,
		SamplerRatio:   samplerRatio,
	})
	if err != nil {
		log.ErrorContext(ctx, "Failed to initialize tracer", slog.Any("error", err))
	}
	defer func() {
		if err := shutdownTracer(context.Background()); err != nil {
			log.ErrorContext(ctx, "Error shutting down tracer", slog.Any("error", err))
		}
	}()

	// Setup Repo
	usersRepo := repositories.NewUserRepository(sqlcQueries)
	jwtBlacklistRepo := repositories.NewJWTBlacklistRepository(redisClient)
	refreshTokenRepo := repositories.NewRefreshTokenRepository(redisClient)

	validate := validator.New()

	// Setup Service
	audiences := strings.Split(cfg.Server.JWTAudience, ",")
	tokenService := token.NewJWTTokenService(cfg.Server.JWTSecret, cfg.Server.JWTIssuer, audiences, jwtBlacklistRepo)
	baseUserService := services.NewUserService(usersRepo, validate, tokenService, jwtBlacklistRepo, refreshTokenRepo, eventPublisher, kafkaProducer, log)
	baseAuthService := services.NewAuthService(usersRepo, validate, tokenService, jwtBlacklistRepo, refreshTokenRepo, eventPublisher, kafkaProducer, log)
	userService := services.NewUserTracingDecorator(baseUserService)
	authService := services.NewAuthTracingDecorator(baseAuthService)

	// Setup Handler
	userHandler := handlers.NewUserHandler(userService, jwtBlacklistRepo, log)
	authHandler := handlers.NewAuthHandler(authService, userService, jwtBlacklistRepo, log)

	// Setup gRPC
	lis, err := net.Listen("tcp", net.JoinHostPort("", cfg.Server.GRPCPort))
	if err != nil {
		log.Error("Failed to listen for gRPC server", slog.String("port", cfg.Server.GRPCPort), slog.Any("error", err))
		os.Exit(1)
	}

	s := grpc.NewServer()
	authpb.RegisterAuthServiceServer(s, grpcServer.NewAuthServer(tokenService))
	accountpb.RegisterAccountServiceServer(s, grpcServer.NewAccountServer(userService))
	reflection.Register(s)

	go func() {
		if err := s.Serve(lis); err != nil {
			log.Error("Failed to serve gRPC server", slog.Any("error", err))
			os.Exit(1)
		}
	}()

	e := echo.New()

	e.Use(middleware.Recover())
	e.Use(middleware.RequestID())
	e.Use(middlewareApp.LoggingMiddleware(log))
	e.Use(echootel.NewMiddleware("tokohobby-accounts-service"))
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"}, // Nginx will handle stricter CORS (NOT FOR PRODUCTION)
		AllowMethods: []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodOptions},
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, echo.HeaderAuthorization},
	}))

	// Setup Route
	routes.InitRoutes(e, userHandler, authHandler, tokenService)

	// Graceful shutdown pattern
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		if err := e.Start(":" + cfg.Server.Port); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Error("Failed to start server", slog.Any("error", err))
			os.Exit(1)
		}
	}()

	<-quit // tunggu sinyal shutdown

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := e.Shutdown(shutdownCtx); err != nil {
		log.Error("Server forced to shutdown", slog.Any("error", err))
	}
	s.GracefulStop() // gRPC graceful stop
	log.Info("Server exited gracefully")
}
