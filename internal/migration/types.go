// internal/migration/types.go
package migration

import (
	"io/fs"
	"path/filepath"
	"time"
)

// ChecksumAlgorithm checksum hesaplama algoritması
type ChecksumAlgorithm string

const (
	ChecksumMD5    ChecksumAlgorithm = "md5"    // Hızlı, development için (128-bit)
	ChecksumSHA256 ChecksumAlgorithm = "sha256" // Güvenli, production için (256-bit)
)

// BackupStrategy down migration öncesi backup stratejisi
type BackupStrategy string

const (
	BackupNone   BackupStrategy = "none"   // Backup alma
	BackupSQL    BackupStrategy = "sql"    // SQL dump (pg_dump ile tam database)
	BackupTable  BackupStrategy = "table"  // Sadece etkilenen tabloları dump
	BackupCustom BackupStrategy = "custom" // Özel backup fonksiyonu (hook)
)

// HealthStatus migration sisteminin genel sağlık durumunu belirtir
type HealthStatus string

const (
	StatusHealthy HealthStatus = "healthy" // Tüm migration'lar uygun
	StatusWarning HealthStatus = "warning" // Pending migration'lar var
	StatusError   HealthStatus = "error"   // Checksum hatası veya system error
)

// MigrationDirection migration yönünü belirtir
type MigrationDirection string

const (
	DirectionUp   MigrationDirection = "up"   // İleri migration (CREATE, ALTER)
	DirectionDown MigrationDirection = "down" // Geri migration (DROP, ROLLBACK)
)

// Migration tek bir veritabanı migration'ını temsil eder
type Migration struct {
	Version      int64      `json:"version"`                // Migration version (timestamp: 20250808123045)
	Name         string     `json:"name"`                   // Migration adı ("create_users_table")
	UpSQL        string     `json:"-"`                      // UP SQL içeriği (JSON'da gösterilmez - güvenlik)
	DownSQL      string     `json:"-"`                      // DOWN SQL içeriği (JSON'da gösterilmez - güvenlik)
	Applied      bool       `json:"applied"`                // Uygulandı mı?
	AppliedAt    *time.Time `json:"appliedAt,omitempty"`    // Ne zaman uygulandı? (RFC3339 format)
	UpChecksum   string     `json:"upChecksum"`             // UP dosyası checksum
	DownChecksum string     `json:"downChecksum,omitempty"` // DOWN dosyası checksum (opsiyonel)
	UpFileSize   int64      `json:"upFileSize"`             // UP dosya boyutu (byte) - EKLENDİ
	DownFileSize int64      `json:"downFileSize,omitempty"` // DOWN dosya boyutu (byte) - EKLENDİ
	Description  string     `json:"description,omitempty"`  // Migration açıklaması (dosyadan parse)
	HasDownFile  bool       `json:"hasDownFile"`            // DOWN dosyası mevcut mu? - EKLENDİ
}

// MigrationStatus migration sisteminin genel durumunu gösterir
type MigrationStatus struct {
	CurrentVersion int64        `json:"currentVersion"`          // Şu anki version (timestamp)
	Migrations     []Migration  `json:"migrations"`              // Tüm migration'lar (version sıralı)
	TotalCount     int          `json:"totalCount"`              // Toplam migration sayısı
	AppliedCount   int          `json:"appliedCount"`            // Uygulanan migration sayısı
	PendingCount   int          `json:"pendingCount"`            // Bekleyen migration sayısı
	LastAppliedAt  *time.Time   `json:"lastAppliedAt,omitempty"` // Son migration zamanı
	SystemHealth   HealthStatus `json:"systemHealth"`            // Sağlık durumu
	ChecksumValid  bool         `json:"checksumValid"`           // Tüm checksum'lar geçerli mi?
	ErrorCount     int          `json:"errorCount"`              // Checksum hata sayısı
	WarningCount   int          `json:"warningCount"`            // Uyarı sayısı (pending migration)
}

// MigrationResult bir migration işleminin sonucunu tutar
type MigrationResult struct {
	Success       bool               `json:"success"`                // Başarılı oldu mu?
	Version       int64              `json:"version"`                // Hangi version?
	Name          string             `json:"name"`                   // Migration adı
	Direction     MigrationDirection `json:"direction"`              // "up" veya "down"
	ExecutionTime time.Duration      `json:"executionTime"`          // Execution süresi (nanosecond)
	Error         string             `json:"error,omitempty"`        // Hata mesajı (varsa)
	AffectedRows  int64              `json:"affectedRows,omitempty"` // Etkilenen satır sayısı
	ChecksumValid bool               `json:"checksumValid"`          // Checksum kontrolü geçti mi?
	BackupTaken   bool               `json:"backupTaken,omitempty"`  // Backup alındı mı?
	BackupPath    string             `json:"backupPath,omitempty"`   // Backup dosya yolu (absolute)
	SqlStatements int                `json:"sqlStatements"`          // Kaç SQL statement çalıştırıldı
	StartedAt     time.Time          `json:"startedAt"`              // Başlangıç zamanı
	CompletedAt   *time.Time         `json:"completedAt,omitempty"`  // Tamamlanma zamanı
}

// MigrationConfig migration ayarlarını tutar
type MigrationConfig struct {
	// Path ve dosya ayarları
	MigrationsPath     string      `json:"migrationsPath"`  // Migration dosyalarının yolu (absolute)
	TableName          string      `json:"tableName"`       // Takip tablosu adı
	AutoCreatePath     bool        `json:"autoCreatePath"`  // Migration klasörü yoksa oluştur
	FilePermissions    fs.FileMode `json:"-"`               // Yeni dosya izinleri (JSON'da gösterilmez)
	FilePermissionsStr string      `json:"filePermissions"` // Dosya izinleri string formatında ("0644")

	// Güvenlik ayarları
	ChecksumAlgorithm ChecksumAlgorithm `json:"checksumAlgorithm"` // Checksum algoritması
	ValidateChecksums bool              `json:"validateChecksums"` // Checksum kontrolü aktif mi?
	AllowDirtyMigrate bool              `json:"allowDirtyMigrate"` // Değişmiş dosyaları zorla çalıştır
	RequireDownFiles  bool              `json:"requireDownFiles"`  // DOWN dosyası zorunlu mu? - EKLENDİ

	// Performans ayarları
	LockTimeout        int `json:"lockTimeout"`        // Kilit timeout (saniye)
	MaxOpenConns       int `json:"maxOpenConns"`       // Maksimum DB bağlantısı
	TransactionTimeout int `json:"transactionTimeout"` // Transaction timeout (saniye)
	BatchSize          int `json:"batchSize"`          // Toplu işlem boyutu

	// Backup ayarları
	BackupStrategy BackupStrategy `json:"backupStrategy"` // Backup stratejisi
	BackupPath     string         `json:"backupPath"`     // Backup dosyalarının yolu (absolute)
	KeepBackups    int            `json:"keepBackups"`    // Kaç backup dosyası sakla
	BackupTimeout  int            `json:"backupTimeout"`  // Backup timeout (saniye)

	// Çalışma modu
	IsCLI   bool `json:"isCli"`   // CLI modunda mı?
	DryRun  bool `json:"dryRun"`  // Sadece test et, uygulamadan
	Verbose bool `json:"verbose"` // Detaylı log çıktısı
	Debug   bool `json:"debug"`   // Debug mod (SQL query'leri logla)
}

// DefaultConfig varsayılan ayarları döner
func DefaultConfig() *MigrationConfig {
	// Migration path'ini absolute yap - HATA YÖNETİMİ İLE
	absPath, err := filepath.Abs("./migrations")
	if err != nil {
		absPath = "./migrations" // Fallback
	}

	// Backup path'ini absolute yap - HATA YÖNETİMİ İLE
	backupPath, err := filepath.Abs("./backups/migrations")
	if err != nil {
		backupPath = "./backups/migrations" // Fallback
	}

	return &MigrationConfig{
		// Path ayarları
		MigrationsPath:     absPath,
		TableName:          "schema_migrations",
		AutoCreatePath:     true,
		FilePermissions:    0644,
		FilePermissionsStr: "0644",

		// Güvenlik ayarları
		ChecksumAlgorithm: ChecksumSHA256, // Production-ready default
		ValidateChecksums: true,
		AllowDirtyMigrate: false,
		RequireDownFiles:  false, // DOWN dosyası opsiyonel

		// Performans ayarları
		LockTimeout:        300, // 5 dakika
		MaxOpenConns:       1,   // Güvenli başlangıç
		TransactionTimeout: 900, // 15 dakika
		BatchSize:          100, // Toplu işlem boyutu

		// Backup ayarları
		BackupStrategy: BackupNone,
		BackupPath:     backupPath,
		KeepBackups:    5,
		BackupTimeout:  600, // 10 dakika

		// Çalışma modu
		IsCLI:   false,
		DryRun:  false,
		Verbose: false,
		Debug:   false,
	}
}

// CLIConfig CLI kullanımı için optimize edilmiş ayarlar
func CLIConfig() *MigrationConfig {
	c := DefaultConfig()
	c.IsCLI = true
	c.MaxOpenConns = 1           // CLI için tek bağlantı güvenli
	c.LockTimeout = 1800         // CLI: 30 dakika (manuel işlem)
	c.BackupStrategy = BackupSQL // CLI: SQL backup default
	c.Verbose = true             // CLI: detaylı çıktı
	c.Debug = false              // CLI: debug opsiyonel
	c.AllowDirtyMigrate = false  // CLI: güvenlik öncelik
	c.RequireDownFiles = true    // CLI: DOWN dosyası zorunlu
	return c
}

// AppStartupConfig uygulama başlangıcı için optimize
func AppStartupConfig() *MigrationConfig {
	c := DefaultConfig()
	c.IsCLI = false
	c.MaxOpenConns = 3            // Startup: hız için daha fazla bağlantı
	c.LockTimeout = 180           // Startup: 3 dakika
	c.BackupStrategy = BackupNone // Startup: backup yok (hız)
	c.Verbose = false             // Startup: sessiz
	c.Debug = false               // Startup: debug kapalı
	c.ValidateChecksums = true    // Startup: mutlaka kontrol
	c.AllowDirtyMigrate = false   // Startup: güvenlik
	c.RequireDownFiles = false    // Startup: DOWN opsiyonel
	return c
}

// ProductionConfig production ortamı için maksimum güvenlik
func ProductionConfig() *MigrationConfig {
	c := DefaultConfig()
	c.IsCLI = false
	c.MaxOpenConns = 1           // Production: tek bağlantı
	c.LockTimeout = 1800         // Production: 30 dakika
	c.TransactionTimeout = 1800  // Production: uzun timeout
	c.BackupStrategy = BackupSQL // Production: mutlaka backup
	c.ValidateChecksums = true   // Production: checksum zorunlu
	c.AllowDirtyMigrate = false  // Production: değişmiş dosya yasak
	c.AutoCreatePath = false     // Production: manuel path oluştur
	c.KeepBackups = 10           // Production: daha fazla backup
	c.Verbose = true             // Production: detaylı log
	c.Debug = false              // Production: debug kapalı
	c.RequireDownFiles = true    // Production: DOWN zorunlu
	c.BackupTimeout = 1200       // Production: 20 dakika backup timeout
	return c
}

// DevelopmentConfig development ortamı için esnek ayarlar
func DevelopmentConfig() *MigrationConfig {
	c := DefaultConfig()
	c.ChecksumAlgorithm = ChecksumMD5 // Development: hızlı MD5
	c.ValidateChecksums = false       // Development: esnek
	c.AllowDirtyMigrate = true        // Development: değişiklik OK
	c.BackupStrategy = BackupNone     // Development: backup gereksiz
	c.LockTimeout = 60                // Development: kısa timeout
	c.Verbose = true                  // Development: detaylı log
	c.Debug = true                    // Development: debug aktif
	c.RequireDownFiles = false        // Development: DOWN opsiyonel
	c.AutoCreatePath = true           // Development: otomatik klasör oluştur
	return c
}

// TestConfig test ortamı için optimize ayarlar
func TestConfig() *MigrationConfig {
	c := DefaultConfig()
	c.ChecksumAlgorithm = ChecksumMD5 // Test: hızlı MD5
	c.ValidateChecksums = false       // Test: esnek
	c.AllowDirtyMigrate = true        // Test: her şeye izin
	c.BackupStrategy = BackupNone     // Test: backup gereksiz
	c.LockTimeout = 30                // Test: çok kısa timeout
	c.TransactionTimeout = 60         // Test: kısa transaction
	c.Verbose = false                 // Test: sessiz
	c.Debug = false                   // Test: debug kapalı
	c.DryRun = false                  // Test: gerçek çalıştır
	c.RequireDownFiles = false        // Test: DOWN opsiyonel
	c.AutoCreatePath = true           // Test: otomatik klasör
	return c
}
