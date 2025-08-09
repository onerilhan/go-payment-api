// cmd/migrate/main.go
package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"

	"github.com/onerilhan/go-payment-api/internal/config"
	"github.com/onerilhan/go-payment-api/internal/db"
	"github.com/onerilhan/go-payment-api/internal/migration"
)

func main() {
	// .env dosyasını yükle
	if err := godotenv.Load(); err != nil {
		fmt.Println("Warning: .env file not found, using environment variables")
	}

	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	// Config yükle
	cfg := config.LoadConfig()

	// Database bağlantısı
	database, err := db.Connect(cfg.GetDSN())
	if err != nil {
		fmt.Printf("Database connection failed: %v\n", err)
		os.Exit(1)
	}
	defer database.Close()

	// Migration runner oluştur (CLI config ile)
	runner := migration.NewRunner(database, migration.CLIConfig())

	// Komut çalıştır
	switch command {
	case "status":
		handleStatus(runner)
	case "up":
		handleUp(runner, os.Args[2:])
	case "down":
		handleDown(runner, os.Args[2:])
	case "create":
		handleCreate(runner, os.Args[2:])
	case "init":
		handleInit(runner)
	default:
		fmt.Printf("Unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Print(`
Migration CLI Tool

USAGE:
    go run cmd/migrate/main.go <command> [arguments]

COMMANDS:
    status              Show migration status
    up [version]        Apply pending migrations (up to optional version)
    down <version>      Rollback migrations to specified version
    create <name>       Create new migration files
    init                Initialize migration system

EXAMPLES:
    go run cmd/migrate/main.go status
    go run cmd/migrate/main.go up
    go run cmd/migrate/main.go up 20250808123045
    go run cmd/migrate/main.go down 20250808120000
    go run cmd/migrate/main.go create "add_user_avatar"
    go run cmd/migrate/main.go init
`)
}

func handleStatus(runner *migration.Runner) {
	fmt.Println("Checking migration status...")

	status, err := runner.GetStatus()
	if err != nil {
		fmt.Printf("Failed to get migration status: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("\nMigration Status:\n")
	fmt.Printf("  Current Version: %d\n", status.CurrentVersion)
	fmt.Printf("  Total Migrations: %d\n", status.TotalCount)
	fmt.Printf("  Applied: %d\n", status.AppliedCount)
	fmt.Printf("  Pending: %d\n", status.PendingCount)
	fmt.Printf("  System Health: %s\n", status.SystemHealth)
	fmt.Printf("  Checksum Valid: %t\n", status.ChecksumValid)

	if len(status.Migrations) > 0 {
		fmt.Printf("\nMigrations:\n")
		fmt.Println("  VERSION          | STATUS   | NAME")
		fmt.Println("  -----------------|----------|--------------------")

		for _, m := range status.Migrations {
			statusIcon := "PENDING"
			appliedAt := ""

			if m.Applied {
				statusIcon = "APPLIED"
				if m.AppliedAt != nil {
					appliedAt = fmt.Sprintf(" (%s)", m.AppliedAt.Format("2006-01-02 15:04"))
				}
			}

			fmt.Printf("  %14d | %-8s | %s%s\n",
				m.Version, statusIcon, m.Name, appliedAt)
		}
	}

	if status.PendingCount > 0 {
		fmt.Printf("\nYou have %d pending migration(s). Run 'up' to apply them.\n", status.PendingCount)
	} else {
		fmt.Printf("\nAll migrations are up to date!\n")
	}
}

func handleUp(runner *migration.Runner, args []string) {
	targetVersion := int64(0)
	if len(args) > 0 {
		var err error
		targetVersion, err = strconv.ParseInt(args[0], 10, 64)
		if err != nil {
			fmt.Printf("Invalid version number: %s\n", args[0])
			os.Exit(1)
		}
	}

	if targetVersion > 0 {
		fmt.Printf("Applying migrations up to version %d...\n", targetVersion)
	} else {
		fmt.Println("Applying all pending migrations...")
	}

	results, err := runner.RunUp(targetVersion)
	if err != nil {
		fmt.Printf("Migration failed: %v\n", err)
		os.Exit(1)
	}

	if len(results) == 0 {
		fmt.Println("No pending migrations to apply")
		return
	}

	fmt.Printf("\nMigration Results:\n")
	successCount := 0

	for _, result := range results {
		status := "FAILED"
		if result.Success {
			status = "SUCCESS"
			successCount++
		}

		fmt.Printf("  %s | Version %d | %s | %v\n",
			status, result.Version, result.Name, result.ExecutionTime)

		if !result.Success {
			fmt.Printf("    Error: %s\n", result.Error)
			break
		}

		if result.AffectedRows > 0 {
			fmt.Printf("    Affected rows: %d\n", result.AffectedRows)
		}
	}

	fmt.Printf("\nSummary: %d/%d migrations applied successfully\n", successCount, len(results))

	if successCount == len(results) {
		fmt.Println("All migrations completed successfully!")
	} else {
		fmt.Println("Some migrations failed. Check errors above.")
		os.Exit(1)
	}
}

func handleDown(runner *migration.Runner, args []string) {
	if len(args) == 0 {
		fmt.Println("Target version required for rollback")
		fmt.Println("Usage: down <version>")
		os.Exit(1)
	}

	targetVersion, err := strconv.ParseInt(args[0], 10, 64)
	if err != nil {
		fmt.Printf("Invalid version number: %s\n", args[0])
		os.Exit(1)
	}

	fmt.Printf("Rolling back to version %d...\n", targetVersion)

	// Confirmation
	fmt.Printf("WARNING: This will rollback your database!\n")
	fmt.Printf("Are you sure you want to continue? (y/N): ")

	var response string
	fmt.Scanln(&response)

	if strings.ToLower(response) != "y" && strings.ToLower(response) != "yes" {
		fmt.Println("Rollback cancelled")
		return
	}

	results, err := runner.RunDown(targetVersion)
	if err != nil {
		fmt.Printf("Rollback failed: %v\n", err)
		os.Exit(1)
	}

	if len(results) == 0 {
		fmt.Println("No migrations to rollback")
		return
	}

	fmt.Printf("\nRollback Results:\n")
	successCount := 0

	for _, result := range results {
		status := "FAILED"
		if result.Success {
			status = "SUCCESS"
			successCount++
		}

		fmt.Printf("  %s | Version %d | %s | %v\n",
			status, result.Version, result.Name, result.ExecutionTime)

		if !result.Success {
			fmt.Printf("    Error: %s\n", result.Error)
			break
		}

		if result.AffectedRows > 0 {
			fmt.Printf("    Affected rows: %d\n", result.AffectedRows)
		}
	}

	fmt.Printf("\nSummary: %d/%d rollbacks completed successfully\n", successCount, len(results))

	if successCount == len(results) {
		fmt.Println("Rollback completed successfully!")
	} else {
		fmt.Println("Some rollbacks failed. Check errors above.")
		os.Exit(1)
	}
}

func handleCreate(runner *migration.Runner, args []string) {
	if len(args) == 0 {
		fmt.Println("Migration name required")
		fmt.Println("Usage: create <name>")
		fmt.Println("Example: create \"add_user_avatar\"")
		os.Exit(1)
	}

	name := strings.Join(args, " ")

	fmt.Printf("Creating migration: %s\n", name)

	err := createMigrationFiles(name)
	if err != nil {
		fmt.Printf("Failed to create migration: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Migration files created successfully!")
	fmt.Println("  Check ./migrations/ directory")
	fmt.Println("  Edit the SQL files and run 'up' to apply")
}

func handleInit(runner *migration.Runner) {
	fmt.Println("Initializing migration system...")

	err := runner.Initialize()
	if err != nil {
		fmt.Printf("Initialization failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Migration system initialized successfully!")
	fmt.Println("  Migration tracking table created")
	fmt.Println("  Run 'status' to check current state")
}

// createMigrationFiles yeni migration dosyaları oluşturur
func createMigrationFiles(name string) error {
	// Timestamp version oluştur (YYYYMMDDHHMMSS format)
	version := generateTimestampVersion()

	// Migration adını temizle
	cleanName := cleanMigrationName(name)

	// Dosya adları
	upFileName := fmt.Sprintf("%d_%s.up.sql", version, cleanName)
	downFileName := fmt.Sprintf("%d_%s.down.sql", version, cleanName)

	// Migration klasörü yolu
	migrationsPath := "./migrations"

	// Klasör yoksa oluştur
	if err := os.MkdirAll(migrationsPath, 0755); err != nil {
		return fmt.Errorf("migration klasörü oluşturulamadı: %w", err)
	}

	// UP dosyası içeriği
	upContent := fmt.Sprintf(`-- Migration: %s
-- Version: %d
-- Created: %s

-- TODO: Add your UP migration SQL here

`, name, version, formatTimestamp(version))

	// DOWN dosyası içeriği
	downContent := fmt.Sprintf(`-- Rollback: %s
-- Version: %d
-- Created: %s

-- TODO: Add your DOWN migration SQL here
-- This should reverse the changes made in the UP migration

`, name, version, formatTimestamp(version))

	// UP dosyasını oluştur
	upPath := fmt.Sprintf("%s/%s", migrationsPath, upFileName)
	if err := os.WriteFile(upPath, []byte(upContent), 0644); err != nil {
		return fmt.Errorf("UP dosyası oluşturulamadı: %w", err)
	}

	// DOWN dosyasını oluştur
	downPath := fmt.Sprintf("%s/%s", migrationsPath, downFileName)
	if err := os.WriteFile(downPath, []byte(downContent), 0644); err != nil {
		return fmt.Errorf("DOWN dosyası oluşturulamadı: %w", err)
	}

	fmt.Printf("  Created: %s\n", upPath)
	fmt.Printf("  Created: %s\n", downPath)

	return nil
}

// Helper functions

// generateTimestampVersion timestamp formatında version oluşturur
func generateTimestampVersion() int64 {
	now := time.Now()
	// YYYYMMDDHHMMSS format: 20250808143022
	version := now.Format("20060102150405")

	// String'i int64'e çevir
	versionInt, _ := strconv.ParseInt(version, 10, 64)
	return versionInt
}

// cleanMigrationName migration adını dosya adı için temizler
func cleanMigrationName(name string) string {
	// Küçük harfe çevir
	clean := strings.ToLower(name)

	// Boşlukları underscore yap
	clean = strings.ReplaceAll(clean, " ", "_")

	// Özel karakterleri kaldır
	clean = strings.ReplaceAll(clean, "-", "_")
	clean = strings.ReplaceAll(clean, ".", "_")
	clean = strings.ReplaceAll(clean, ",", "_")
	clean = strings.ReplaceAll(clean, "!", "_")
	clean = strings.ReplaceAll(clean, "?", "_")
	clean = strings.ReplaceAll(clean, "/", "_")
	clean = strings.ReplaceAll(clean, "\\", "_")

	// Çoklu underscore'ları tek yap
	for strings.Contains(clean, "__") {
		clean = strings.ReplaceAll(clean, "__", "_")
	}

	// Başında ve sonunda underscore varsa kaldır
	clean = strings.Trim(clean, "_")

	return clean
}

// formatTimestamp timestamp'i human readable formata çevirir
func formatTimestamp(version int64) string {
	versionStr := fmt.Sprintf("%d", version)
	if len(versionStr) != 14 {
		return "Invalid timestamp"
	}

	// 20250808143022 -> 2025-08-08 14:30:22
	year := versionStr[0:4]
	month := versionStr[4:6]
	day := versionStr[6:8]
	hour := versionStr[8:10]
	minute := versionStr[10:12]
	second := versionStr[12:14]

	return fmt.Sprintf("%s-%s-%s %s:%s:%s", year, month, day, hour, minute, second)
}
