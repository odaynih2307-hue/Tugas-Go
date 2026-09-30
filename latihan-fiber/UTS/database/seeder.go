package database

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/jackc/pgx/v5/pgxpool"
)

// RunMigrationsAndSeeders mengeksekusi file SQL migrasi dan seeder.
func RunMigrationsAndSeeders(ctx context.Context, pool *pgxpool.Pool, baseDir string) error {
	migrationFiles := []string{
		"migrations/001_create_schema.sql",
		"migrations/002_seeder.sql",
	}

	for _, relPath := range migrationFiles {
		fullPath := filepath.Join(baseDir, relPath)
		content, err := os.ReadFile(fullPath)
		if err != nil {
			return fmt.Errorf("membaca file %s: %w", fullPath, err)
		}

		log.Printf("Mengeksekusi SQL: %s ...", relPath)
		if _, err := pool.Exec(ctx, string(content)); err != nil {
			return fmt.Errorf("eksekusi SQL %s gagal: %w", relPath, err)
		}
	}

	log.Println("Migrasi dan seeder berhasil dieksekusi sepenuhnya.")
	return nil
}
