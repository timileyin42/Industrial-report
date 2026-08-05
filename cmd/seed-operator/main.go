// Command seed-operator creates the first operator account. There is no
// bootstrap HTTP endpoint for this deliberately — an unauthenticated
// "create the first admin" route is itself a security hole (see plan
// notes / CLAUDE.md security section). Run this once per environment:
//
//	DATABASE_URL=... go run ./cmd/seed-operator -email admin@zgnis.test -password 'change-me-now'
package main

import (
	"context"
	"flag"
	"log"
	"os"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/timileyin42/zgnis-solar/internal/auth"
	"github.com/timileyin42/zgnis-solar/internal/db"
)

func main() {
	email := flag.String("email", "", "operator email")
	password := flag.String("password", "", "operator password (min 8 chars)")
	flag.Parse()

	if *email == "" || len(*password) < 8 {
		log.Fatal("usage: seed-operator -email <email> -password <min 8 chars>")
	}

	dbURL := mustEnv("DATABASE_URL")
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Fatalf("db connect: %v", err)
	}
	defer pool.Close()

	hash, err := auth.HashSecret(*password)
	if err != nil {
		log.Fatalf("hash password: %v", err)
	}

	queries := db.New(pool)
	user, err := queries.CreateUser(ctx, db.CreateUserParams{
		Email:        *email,
		PasswordHash: hash,
		Role:         db.UserRoleOperator,
		SiteID:       pgtype.Text{},
	})
	if err != nil {
		log.Fatalf("create operator: %v", err)
	}

	log.Printf("created operator user id=%d email=%s", user.ID, user.Email)
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		log.Fatalf("required env var %s not set", key)
	}
	return v
}
