// Command admin runs small ops commands against the configured database:
//
//	admin set-role <email> <role>   change an account's role
//	admin prune-tokens              delete expired refresh tokens
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/arifinrafi89/waraqah-backend/internal/platform/auth"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/config"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/db"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/db/sqlc"
	"github.com/arifinrafi89/waraqah-backend/internal/platform/logx"
)

func main() {
	if len(os.Args) < 2 {
		usage()
	}
	cfg, err := config.Load()
	if err != nil {
		fail(err)
	}
	log := logx.New(os.Stderr, cfg.LogLevel, cfg.LogFormat)
	ctx := context.Background()
	database, err := db.Open(ctx, cfg.DatabaseURL, 2, false, log)
	if err != nil {
		fail(err)
	}
	defer database.Close()

	switch os.Args[1] {
	case "set-role":
		if len(os.Args) != 4 {
			usage()
		}
		role, ok := auth.ParseRole(os.Args[3])
		if !ok {
			fail(fmt.Errorf("unknown role %q (reader, moderator, catalogManager, support, superAdmin)", os.Args[3]))
		}
		n, err := database.Q().SetUserRoleByEmail(ctx, sqlc.SetUserRoleByEmailParams{Email: pgText(os.Args[2]), Role: string(role)})
		if err != nil {
			fail(err)
		}
		if n == 0 {
			fail(fmt.Errorf("no account with email %q", os.Args[2]))
		}
		fmt.Printf("%s is now %s\n", os.Args[2], role)
	case "prune-tokens":
		n, err := database.Q().PruneRefreshTokens(ctx, time.Now())
		if err != nil {
			fail(err)
		}
		fmt.Printf("deleted %d expired refresh tokens\n", n)
	default:
		usage()
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, "usage: admin set-role <email> <role> | admin prune-tokens")
	os.Exit(2)
}

func fail(err error) {
	fmt.Fprintln(os.Stderr, "error:", err)
	os.Exit(1)
}
