// internal/migration/runner_files.go
package migration

import (
	"crypto/md5"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

// Hem eski (000001) hem yeni (20250808123045) formatları destekle
//   - 6 haneli: sıra numarası (000001)
//   - 14 haneli: timestamp (YYYYMMDDHHMMSS)
var migrationFilePattern = regexp.MustCompile(`^(\d{6}|\d{14})_([a-zA-Z0-9_]+)\.(up|down)\.sql$`)

// LoadMigrationsFromDisk ./migrations klasöründeki tüm migration dosyalarını okur
func (r *Runner) LoadMigrationsFromDisk() ([]Migration, error) {
	// Deadlock riskini azaltmak için burada global lock kullanmıyoruz

	if r.config.Verbose {
		log.Info().
			Str("path", r.config.MigrationsPath).
			Msg("Migration dosyaları okunuyor...")
	}

	// .up.sql dosyalarını bul
	upFiles, err := filepath.Glob(filepath.Join(r.config.MigrationsPath, "*.up.sql"))
	if err != nil {
		return nil, fmt.Errorf("migration dosyaları bulunamadı: %w", err)
	}

	if len(upFiles) == 0 {
		log.Warn().
			Str("path", r.config.MigrationsPath).
			Msg("Hiç migration dosyası bulunamadı")
		return []Migration{}, nil
	}

	var migrations []Migration
	for _, upFile := range upFiles {
		migration, err := r.parseMigrationFile(upFile)
		if err != nil {
			if r.config.Verbose {
				log.Warn().
					Err(err).
					Str("file", upFile).
					Msg("Migration dosyası parse edilemedi, atlanıyor")
			}
			continue
		}
		migrations = append(migrations, migration)
	}

	// Version'a göre sırala (eskiden yeniye)
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})

	if r.config.Verbose {
		log.Info().
			Int("count", len(migrations)).
			Msg("Migration dosyaları başarıyla okundu")
	}

	return migrations, nil
}

// parseMigrationFile tek bir migration dosyasını parse eder
func (r *Runner) parseMigrationFile(upFilePath string) (Migration, error) {
	filename := filepath.Base(upFilePath)
	matches := migrationFilePattern.FindStringSubmatch(filename)
	if len(matches) != 4 {
		return Migration{}, fmt.Errorf("geçersiz migration dosya formatı: %s", filename)
	}

	versionStr := matches[1]
	version, err := strconv.ParseInt(versionStr, 10, 64)
	if err != nil {
		return Migration{}, fmt.Errorf("geçersiz version formatı %s: %w", versionStr, err)
	}

	// 6 haneli kısa formatı deterministik 14 haneye yükselt
	// Örn: 000001 -> <bugün YYYYMMDD> + 000001  => YYYYMMDD000001
	if len(versionStr) == 6 {
		today := time.Now().Format("20060102")
		version, _ = strconv.ParseInt(today+fmt.Sprintf("%06d", version), 10, 64)
	}

	name := matches[2]
	name = strings.ReplaceAll(name, "_", " ")
	name = toTitleCase(name)

	// UP içeriği
	upContent, err := os.ReadFile(upFilePath)
	if err != nil {
		return Migration{}, fmt.Errorf("UP dosyası okunamadı %s: %w", upFilePath, err)
	}
	upStat, err := os.Stat(upFilePath)
	if err != nil {
		return Migration{}, fmt.Errorf("UP dosya bilgileri alınamadı: %w", err)
	}
	upFileSize := upStat.Size()

	// DOWN kontrol
	downFilePath := strings.Replace(upFilePath, ".up.sql", ".down.sql", 1)
	var downContent []byte
	var hasDownFile bool
	var downFileSize int64

	if downStat, err := os.Stat(downFilePath); err == nil {
		if b, err := os.ReadFile(downFilePath); err == nil {
			downContent = b
			hasDownFile = true
			downFileSize = downStat.Size()
		} else {
			log.Warn().
				Str("file", downFilePath).
				Msg("DOWN dosyası okunabilir değil")
		}
	}

	if r.config.RequireDownFiles && !hasDownFile {
		return Migration{}, fmt.Errorf("DOWN dosyası zorunlu ama bulunamadı: %s", downFilePath)
	}

	upChecksum := r.calculateChecksum(upContent)
	downChecksum := ""
	if hasDownFile {
		downChecksum = r.calculateChecksum(downContent)
	}

	description := r.extractDescription(string(upContent))

	m := Migration{
		Version:      version,
		Name:         name,
		UpSQL:        string(upContent),
		DownSQL:      string(downContent),
		Applied:      false,
		AppliedAt:    nil,
		UpChecksum:   upChecksum,
		DownChecksum: downChecksum,
		UpFileSize:   upFileSize,
		DownFileSize: downFileSize,
		Description:  description,
		HasDownFile:  hasDownFile,
	}

	if r.config.Debug {
		log.Debug().
			Int64("version", version).
			Str("name", name).
			Int64("up_size", m.UpFileSize).
			Int64("down_size", downFileSize).
			Bool("has_down", hasDownFile).
			Str("up_checksum", upChecksum[:8]+"...").
			Msg("Migration dosyası parse edildi")
	}

	return m, nil
}

func (r *Runner) calculateChecksum(content []byte) string {
	switch r.config.ChecksumAlgorithm {
	case ChecksumMD5:
		sum := md5.Sum(content)
		return fmt.Sprintf("%x", sum)
	case ChecksumSHA256:
		sum := sha256.Sum256(content)
		return fmt.Sprintf("%x", sum)
	default:
		sum := sha256.Sum256(content)
		return fmt.Sprintf("%x", sum)
	}
}

// SQL dosyasının başındaki açıklama yorum satırını çıkarır
func (r *Runner) extractDescription(sqlContent string) string {
	lines := strings.Split(sqlContent, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "--") {
			desc := strings.TrimSpace(strings.TrimPrefix(line, "--"))
			desc = strings.TrimSpace(strings.TrimPrefix(desc, "Migration:"))
			desc = strings.TrimSpace(strings.TrimPrefix(desc, "Description:"))
			if desc != "" && !strings.HasPrefix(desc, "Version:") {
				return desc
			}
		} else {
			break
		}
	}
	return ""
}

// strings.Title deprecated olduğu için basit title-case
func toTitleCase(s string) string {
	parts := strings.Fields(strings.ToLower(s))
	for i, p := range parts {
		if len(p) > 0 {
			parts[i] = strings.ToUpper(p[:1]) + p[1:]
		}
	}
	return strings.Join(parts, " ")
}
