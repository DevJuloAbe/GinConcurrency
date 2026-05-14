package main

import (
	"context"
	"log"
	"os"
	"strconv"
	"strings"

	"gingonic-concurrency/controller"
	"gingonic-concurrency/database"
	"gingonic-concurrency/repository"
	"gingonic-concurrency/router"
	"gingonic-concurrency/service"

	"github.com/goccy/go-yaml"
)

type AppConfig struct {
	Server struct {
		Port string `yaml:"port"`
	} `yaml:"server"`
	Database struct {
		DSN string `yaml:"dsn"`
	} `yaml:"database"`
	Redis struct {
		Addr     string `yaml:"addr"`
		Password string `yaml:"password"`
		DB       int    `yaml:"db"`
	} `yaml:"redis"`
}

func main() {
	cfg, err := loadConfig("config/config.yaml")
	if err != nil {
		log.Fatal("Failed to load config:", err)
	}

	db, err := database.Connect(cfg.Database.DSN)
	if err != nil {
		log.Fatal("Failed to connect database:", err)
	}

	if err := database.Migrate(db); err != nil {
		log.Fatal("Failed to migrate database:", err)
	}

	redisClient, err := database.ConnectRedis(context.Background(), database.RedisConfig{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	if err != nil {
		log.Fatal("Failed to connect redis:", err)
	}
	defer redisClient.Close()

	userRepository := repository.NewUserRepository(db)
	userService := service.NewUserService(userRepository)
	userController := controller.NewUserController(userService)
	appRouter := router.SetupRouter(userController)

	log.Println("Database connected")
	log.Println("Redis connected")
	if err := appRouter.Run(serverPort(cfg.Server.Port)); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}

func serverPort(configPort string) string {
	port := strings.TrimSpace(os.Getenv("PORT"))
	if port == "" {
		port = strings.TrimSpace(configPort)
	}
	if port == "" {
		return ":8080"
	}
	if strings.Contains(port, ":") {
		return port
	}
	return ":" + port
}

func loadConfig(path string) (*AppConfig, error) {
	config := AppConfig{}
	config.Server.Port = ":8080"
	config.Redis.Addr = "127.0.0.1:6379"

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	if err := applyEnvOverrides(&config); err != nil {
		return nil, err
	}

	return &config, nil
}

func applyEnvOverrides(config *AppConfig) error {
	if value, ok := firstEnv("DATABASE_DSN", "DB_DSN"); ok {
		config.Database.DSN = value
	}
	if value, ok := firstEnv("REDIS_ADDR"); ok {
		config.Redis.Addr = value
	}
	if value, ok := os.LookupEnv("REDIS_PASSWORD"); ok {
		config.Redis.Password = value
	}
	if value, ok := firstEnv("REDIS_DB"); ok {
		db, err := strconv.Atoi(value)
		if err != nil {
			return err
		}
		config.Redis.DB = db
	}

	return nil
}

func firstEnv(names ...string) (string, bool) {
	for _, name := range names {
		if value := strings.TrimSpace(os.Getenv(name)); value != "" {
			return value, true
		}
	}

	return "", false
}
