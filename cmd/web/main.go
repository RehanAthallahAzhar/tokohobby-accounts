package main

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"strings"
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
	customMiddleware "github.com/RehanAthallahAzhar/tokohobby-accounts/internal/middlewares"
	accountpb "github.com/RehanAthallahAzhar/tokohobby-protos/pb/account"
	authpb "github.com/RehanAthallahAzhar/tokohobby-protos/pb/auth"
)

func main() {
	log := logger.NewLogger()

	cfg, err := configs.LoadConfig(log)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	logger.SetLevel(log, cfg.Logrus.Level)

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
		log.Fatalf("DB connection error: %v", err)
	}

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
		log.Fatalf("Failed to create migrate instance: %v", err)
	}

	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	defer conn.Close()

	// Init SQLC
	sqlcQueries := dbGenerated.New(conn)

	//jwt
	_ = helpers.NewJWTHelper(cfg.Server.JWTSecret)

	// Setup Redis
	redisClient, err := redisclient.NewRedisClient(&cfg.Redis, log)
	if err != nil {
		log.Fatalf("Failed to Inilialization redis client : %v", err)
	}
	defer redisClient.Close()

	// Setup Repo
	usersRepo := repositories.NewUserRepository(sqlcQueries, log)
	jwtBlacklistRepo := repositories.NewJWTBlacklistRepository(redisClient)
	refreshTokenRepo := repositories.NewRefreshTokenRepository(redisClient)

	validate := validator.New()

	// Setup Service
	audiences := strings.Split(cfg.Server.JWTAudience, ",")
	tokenService := token.NewJWTTokenService(cfg.Server.JWTSecret, cfg.Server.JWTIssuer, audiences, jwtBlacklistRepo)
	userService := services.NewUserService(usersRepo, validate, tokenService, jwtBlacklistRepo, log)

	// Setup gRPC
	lis, err := net.Listen("tcp", ":"+cfg.Server.GRPCPort)
	if err != nil {
		log.Fatalf("Failed to listen for gRPC server: %s: %v", cfg.Server.GRPCPort, err)
	}

	s := grpc.NewServer()
	authpb.RegisterAuthServiceServer(s, grpcServer.NewAuthServer(tokenService))
	accountpb.RegisterAccountServiceServer(s, grpcServer.NewAccountServer(userService))
	reflection.Register(s)

	go func() {
		if err := s.Serve(lis); err != nil {
			log.Fatalf("Failed to serve gRPC server: %v", err)
		}
	}()

	e := echo.New()

	e.Use(middleware.RequestID())
	e.Use(customMiddleware.LoggingMiddleware(log))
	e.Use(middleware.CORSWithConfig(middleware.CORSConfig{
		AllowOrigins: []string{"*"}, // Nginx will handle stricter CORS
		AllowMethods: []string{http.MethodGet, http.MethodPost, http.MethodPut, http.MethodDelete, http.MethodOptions},
		AllowHeaders: []string{echo.HeaderOrigin, echo.HeaderContentType, echo.HeaderAccept, echo.HeaderAuthorization},
	}))
	// Setup Route
	handler := handlers.NewHandler(usersRepo, userService, tokenService, jwtBlacklistRepo, refreshTokenRepo, log)
	routes.InitRoutes(e, handler, tokenService)

	// Start Echo API REST Server (Block main goroutine)
	e.Logger.Fatal(e.Start(":" + cfg.Server.Port))

	/*
		e.Start(echoPort) -> Ini adalah fungsi pemblokir (blocking function).
			Begitu Anda memanggilnya, fungsi ini akan mengambil alih main() dan akan terus berjalan tanpa henti
			untuk mendengarkan permintaan HTTP.
	*/
}
