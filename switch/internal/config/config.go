// تحميل إعدادات التطبيق من ملف config.yaml — يُرجى الرجوع إلى SPEC §7
package config

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"time"

	"gopkg.in/yaml.v3"
)

// Config — الإعدادات العامة للتطبيق
type Config struct {
	Server        ServerConfig        `yaml:"server"`
	Carrier       CarrierConfig       `yaml:"carrier"`
	Database      DatabaseConfig      `yaml:"database"`
	Security      SecurityConfig      `yaml:"security"`
	KMS           KMSConfig           `yaml:"kms"`
	Notifications NotificationsConfig `yaml:"notifications"`
}

// ServerConfig — إعدادات خادم HTTP العام
type ServerConfig struct {
	Port         int           `yaml:"port"`          // منفذ الاستماع
	ReadTimeout  time.Duration `yaml:"read_timeout"`  // مهلة القراءة
	WriteTimeout time.Duration `yaml:"write_timeout"` // مهلة الكتابة
}

// CarrierConfig — إعدادات نقطة وصول شركة الاتصالات
type CarrierConfig struct {
	Enabled        bool   `yaml:"enabled"`         // تفعيل نقطة وصول الاتصالات
	Port           int    `yaml:"port"`            // منفذ الاستماع لشبكة الاتصالات
	CommissionRate int64  `yaml:"commission_rate"` // نسبة العمولة بالألف (مثلاً 5 = 0.5%)
}

// DatabaseConfig — إعدادات قاعدة البيانات PostgreSQL
type DatabaseConfig struct {
	Host     string `yaml:"host"`      // عنوان الخادم
	Port     int    `yaml:"port"`      // المنفذ
	Name     string `yaml:"name"`      // اسم قاعدة البيانات
	User     string `yaml:"user"`      // المستخدم
	Password string `yaml:"password"`  // كلمة المرور (تدعم ${ENV_VAR})
	MaxConns int    `yaml:"max_conns"` // أقصى عدد اتصالات
}

// DSN — إنشاء سلسلة الاتصال بقاعدة البيانات
func (d *DatabaseConfig) DSN() string {
	return fmt.Sprintf("host=%s port=%d dbname=%s user=%s password=%s sslmode=disable",
		d.Host, d.Port, d.Name, d.User, d.Password)
}

// SecurityConfig — إعدادات الأمان والتحقق
type SecurityConfig struct {
	TimestampTolerance int64  `yaml:"timestamp_tolerance"` // تسامح الطابع الزمني بالثواني
	LookAheadWindow    int64  `yaml:"look_ahead_window"`   // نافذة العداد المسموحة
	DefaultPayerLimit  int64  `yaml:"default_payer_limit"` // حد الدافع الافتراضي بالوحدة الصغرى
	DailyLimit         int64  `yaml:"daily_limit"`         // الحد اليومي بالوحدة الصغرى
	MonthlyLimit       int64  `yaml:"monthly_limit"`       // الحد الشهري بالوحدة الصغرى
	JWTSecret          string `yaml:"jwt_secret"`          // سر JWT (تدعم ${ENV_VAR})
	JWTExpiry          string `yaml:"jwt_expiry"`          // مدة صلاحية JWT مثل 8h
}

// KMSConfig — إعدادات نظام إدارة المفاتيح
type KMSConfig struct {
	Provider  string `yaml:"provider"`   // المزوّد: local أو aws أو gcp
	MasterKey string `yaml:"master_key"` // المفتاح الرئيسي (تدعم ${ENV_VAR})
}

// NotificationsConfig — إعدادات الإشعارات
type NotificationsConfig struct {
	Telegram TelegramConfig `yaml:"telegram"` // إعدادات تيليجرام
}

// TelegramConfig — إعدادات إشعارات تيليجرام
type TelegramConfig struct {
	Enabled  bool   `yaml:"enabled"`   // تفعيل الإشعارات
	BotToken string `yaml:"bot_token"` // رمز البوت (تدعم ${ENV_VAR})
	ChatID   string `yaml:"chat_id"`   // معرّف المحادثة (تدعم ${ENV_VAR})
}

// envVarRegex — نمط استخراج متغيرات البيئة ${VAR}
var envVarRegex = regexp.MustCompile(`\$\{([^}]+)\}`)

// expandEnv — استبدال ${VAR} بقيمة متغير البيئة المقابل
func expandEnv(s string) string {
	return envVarRegex.ReplaceAllStringFunc(s, func(match string) string {
		varName := match[2 : len(match)-1] // إزالة ${ و }
		if val, ok := os.LookupEnv(varName); ok {
			return val
		}
		return match // إبقاء كما هي إذا لم يُوجَد المتغير
	})
}

// Load — تحميل الإعدادات من ملف YAML مع استبدال متغيرات البيئة
func Load(path string) (*Config, error) {
	// قراءة الملف
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("تحميل الإعدادات: قراءة الملف: %w", err)
	}

	// استبدال متغيرات البيئة
	expanded := expandEnv(string(data))

	// تحليل YAML
	var cfg Config
	if err := yaml.Unmarshal([]byte(expanded), &cfg); err != nil {
		return nil, fmt.Errorf("تحميل الإعدادات: تحليل YAML: %w", err)
	}

	// ضبط القيم الافتراضية
	setDefaults(&cfg)

	return &cfg, nil
}

// setDefaults — ضبط القيم الافتراضية للإعدادات غير المُحدَّدة
func setDefaults(cfg *Config) {
	if cfg.Server.Port == 0 {
		cfg.Server.Port = 8080
	}
	if cfg.Server.ReadTimeout == 0 {
		cfg.Server.ReadTimeout = 30 * time.Second
	}
	if cfg.Server.WriteTimeout == 0 {
		cfg.Server.WriteTimeout = 30 * time.Second
	}
	if cfg.Carrier.Port == 0 {
		cfg.Carrier.Port = 8081
	}
	if cfg.Carrier.CommissionRate == 0 {
		cfg.Carrier.CommissionRate = 5 // 0.5% افتراضياً
	}
	if cfg.Database.Port == 0 {
		cfg.Database.Port = 5432
	}
	if cfg.Database.MaxConns == 0 {
		cfg.Database.MaxConns = 20
	}
	if cfg.Security.TimestampTolerance == 0 {
		cfg.Security.TimestampTolerance = 60
	}
	if cfg.Security.LookAheadWindow == 0 {
		cfg.Security.LookAheadWindow = 10
	}
	if cfg.Security.DefaultPayerLimit == 0 {
		cfg.Security.DefaultPayerLimit = 5000
	}
	if cfg.Security.JWTExpiry == "" {
		cfg.Security.JWTExpiry = "8h"
	}
	if cfg.KMS.Provider == "" {
		cfg.KMS.Provider = "local"
	}
}

// ParseDuration — تحليل مدة زمنية من سلسلة مثل "8h" أو "30s"
func ParseDuration(s string) (time.Duration, error) {
	d, err := time.ParseDuration(s)
	if err != nil {
		return 0, fmt.Errorf("تحليل المدة الزمنية %q: %w", s, err)
	}
	return d, nil
}

// ParseInt — تحليل عدد صحيح من سلسلة
func ParseInt(s string) (int, error) {
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("تحليل العدد %q: %w", s, err)
	}
	return n, nil
}
