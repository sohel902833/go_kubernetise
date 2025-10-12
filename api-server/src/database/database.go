package database

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/redis/go-redis/v9"
	"github.com/sohel902833/go-kubernetise-api-server/ent"

	_ "github.com/lib/pq"
)

var (
	Client      *ent.Client
	RedisClient *redis.Client
)

func Connect() {
	// -------------------------------
	// PostgreSQL connection
	// -------------------------------
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		os.Getenv("POSTGRES_HOST"),
		os.Getenv("POSTGRES_PORT"),
		os.Getenv("POSTGRES_USER"),
		os.Getenv("POSTGRES_PASSWORD"),
		os.Getenv("POSTGRES_DB"),
	)

	drv, err := entsql.Open(dialect.Postgres, dsn)
	if err != nil {
		log.Fatalf("failed opening Postgres connection: %v", err)
	}

	Client = ent.NewClient(ent.Driver(drv))

	// Run auto-migration
	if err := Client.Schema.Create(context.Background()); err != nil {
		log.Fatalf("failed creating schema resources: %v", err)
	}

	fmt.Println("✅ Connected to PostgreSQL and migrated schema")

	// -------------------------------
	// Redis connection
	// -------------------------------
	RedisClient = redis.NewClient(&redis.Options{
		Addr:         fmt.Sprintf("%s:%s", os.Getenv("REDIS_HOST"), os.Getenv("REDIS_PORT")),
		Password:     os.Getenv("REDIS_PASSWORD"), // leave empty if no password
		DB:           0,                            // default DB
		DialTimeout:  5 * time.Second,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
		PoolSize:     10,
	})

	// Ping Redis to check connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := RedisClient.Ping(ctx).Err(); err != nil {
		log.Fatalf("failed connecting to Redis: %v", err)
	}

	fmt.Println("✅ Connected to Redis")
}
