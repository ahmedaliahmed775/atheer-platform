// أداة مؤقتة لإنشاء مستخدم إداري افتراضي
package main

import (
	"context"
	"fmt"
	"log"
	"os/signal"
	"syscall"

	"github.com/atheer/switch/internal/config"
	"github.com/atheer/switch/internal/db"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	cfg, err := config.Load("config.yaml")
	if err != nil {
		log.Fatal(err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := db.NewPool(ctx, cfg.Database)
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	// إنشاء كلمة مرور مشفّرة
	password := "admin123"
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatal(err)
	}

	// إدراج المستخدم الإداري
	tag, err := pool.Exec(ctx,
		`INSERT INTO admin_users (email, password_hash, role, scope, is_active)
		 VALUES ($1, $2, $3, $4, $5)
		 ON CONFLICT (email) DO NOTHING`,
		"admin@atheer.ye", string(hash), "SUPER_ADMIN", "*", true)
	if err != nil {
		log.Fatal(err)
	}

	if tag.RowsAffected() == 0 {
		fmt.Println("المستخدم الإداري موجود مسبقاً")
	} else {
		fmt.Println("تم إنشاء المستخدم الإداري بنجاح!")
		fmt.Println("البريد: admin@atheer.ye")
		fmt.Println("كلمة المرور: admin123")
	}
}
