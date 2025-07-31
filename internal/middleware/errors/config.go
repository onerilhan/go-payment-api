package errors

// ErrorConfig error handling middleware ayarları
type ErrorConfig struct {
	ShowStackTrace  bool           // Stack trace'i response'da göster mi (sadece development)
	CustomErrorMap  map[int]string // Status code'a göre custom mesajlar
	LogLevel        string         // Error log level (ERROR, WARN, INFO)
	IncludeHeaders  []string       // Response'da gösterilecek header'lar
	EnablePanicLogs bool           // Panic durumlarını ayrıca logla
	MaxErrorLength  int            // Error mesajının maksimum uzunluğu
}

// DefaultErrorConfig varsayılan error handling ayarları
func DefaultErrorConfig() *ErrorConfig {
	return &ErrorConfig{
		ShowStackTrace: false, // Production'da false
		CustomErrorMap: map[int]string{
			400: "Geçersiz istek. Lütfen parametrelerinizi kontrol edin.",
			401: "Yetkilendirme gerekli. Lütfen giriş yapın.",
			403: "Bu işlem için yetkiniz bulunmuyor.",
			404: "Aradığınız kaynak bulunamadı.",
			409: "Çakışma. Bu işlem şu anda gerçekleştirilemiyor.",
			429: "Çok fazla istek. Lütfen daha sonra tekrar deneyin.",
			500: "Sunucu hatası. Bu durum teknik ekibimize bildirildi.",
			503: "Servis geçici olarak kullanılamıyor. Lütfen daha sonra deneyin.",
		},
		LogLevel:        "ERROR",
		IncludeHeaders:  []string{"X-Request-ID", "X-RateLimit-Remaining"},
		EnablePanicLogs: true,
		MaxErrorLength:  500,
	}
}

// DevelopmentErrorConfig development ortamı için ayarlar
func DevelopmentErrorConfig() *ErrorConfig {
	config := DefaultErrorConfig()
	config.ShowStackTrace = true
	config.LogLevel = "DEBUG"
	config.MaxErrorLength = 2000
	return config
}

// ProductionErrorConfig production ortamı için güvenli ayarlar
func ProductionErrorConfig() *ErrorConfig {
	config := DefaultErrorConfig()
	config.ShowStackTrace = false
	config.CustomErrorMap[500] = "Bir hata oluştu. Teknik ekibimiz bilgilendirildi."
	config.LogLevel = "ERROR"
	config.MaxErrorLength = 200
	return config
}
