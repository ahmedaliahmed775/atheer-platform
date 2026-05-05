// مستودع المستخدمين الإداريين — عمليات CRUD على جدول admin_users
// يُرجى الرجوع إلى SPEC §4
package db

import (
	"context"
	"fmt"
	"time"

	"github.com/atheer/switch/internal/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// AdminRepo — واجهة عمليات المستخدمين الإداريين
type AdminRepo interface {
	// FindByEmail — يبحث عن مستخدم إداري ببريده الإلكتروني
	FindByEmail(ctx context.Context, email string) (*model.AdminUser, error)

	// FindByID — يبحث عن مستخدم إداري بمعرّفه
	FindByID(ctx context.Context, id int64) (*model.AdminUser, error)

	// Create — ينشئ مستخدم إداري جديد
	Create(ctx context.Context, user *model.AdminUser) error

	// Update — يحدّث بيانات مستخدم إداري
	Update(ctx context.Context, user *model.AdminUser) error

	// UpdateStatus — يحدّث حالة التفعيل لمستخدم إداري
	UpdateStatus(ctx context.Context, id int64, isActive bool) error

	// List — يعرض قائمة المستخدمين الإداريين
	List(ctx context.Context) ([]model.AdminUser, error)
}

// adminRepo — تنفيذ مستودع المستخدمين الإداريين
type adminRepo struct {
	pool *pgxpool.Pool
}

// NewAdminRepo — ينشئ نسخة مستودع المستخدمين الإداريين
func NewAdminRepo(pool *pgxpool.Pool) AdminRepo {
	return &adminRepo{pool: pool}
}

// FindByEmail — يبحث عن مستخدم إداري ببريده الإلكتروني
func (r *adminRepo) FindByEmail(ctx context.Context, email string) (*model.AdminUser, error) {
	var user model.AdminUser
	var totpSecret *string
	var lastLoginAt *time.Time

	err := r.pool.QueryRow(ctx, `
		SELECT id, email, password_hash, totp_secret,
		       role, scope, is_active, last_login_at,
		       created_at, updated_at
		FROM admin_users
		WHERE email = $1
	`, email).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &totpSecret,
		&user.Role, &user.Scope, &user.IsActive, &lastLoginAt,
		&user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil // المستخدم غير موجود
		}
		return nil, fmt.Errorf("مستودع الإداريين: بحث ببريد %s: %w", email, err)
	}

	if totpSecret != nil {
		user.TOTPSecret = *totpSecret
	}
	user.LastLoginAt = lastLoginAt

	return &user, nil
}

// FindByID — يبحث عن مستخدم إداري بمعرّفه
func (r *adminRepo) FindByID(ctx context.Context, id int64) (*model.AdminUser, error) {
	var user model.AdminUser
	var totpSecret *string
	var lastLoginAt *time.Time

	err := r.pool.QueryRow(ctx, `
		SELECT id, email, password_hash, totp_secret,
		       role, scope, is_active, last_login_at,
		       created_at, updated_at
		FROM admin_users
		WHERE id = $1
	`, id).Scan(
		&user.ID, &user.Email, &user.PasswordHash, &totpSecret,
		&user.Role, &user.Scope, &user.IsActive, &lastLoginAt,
		&user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil // المستخدم غير موجود
		}
		return nil, fmt.Errorf("مستودع الإداريين: بحث بمعرّف %d: %w", id, err)
	}

	if totpSecret != nil {
		user.TOTPSecret = *totpSecret
	}
	user.LastLoginAt = lastLoginAt

	return &user, nil
}

// Create — ينشئ مستخدم إداري جديد
func (r *adminRepo) Create(ctx context.Context, user *model.AdminUser) error {
	_, err := r.pool.Exec(ctx, `
		INSERT INTO admin_users (email, password_hash, totp_secret, role, scope, is_active)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, user.Email, user.PasswordHash,
		nullIfEmpty(user.TOTPSecret),
		user.Role, user.Scope, user.IsActive,
	)
	if err != nil {
		return fmt.Errorf("مستودع الإداريين: إنشاء مستخدم %s: %w", user.Email, err)
	}
	return nil
}

// Update — يحدّث بيانات مستخدم إداري
func (r *adminRepo) Update(ctx context.Context, user *model.AdminUser) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE admin_users
		SET password_hash = $1, totp_secret = $2, role = $3,
		    scope = $4, is_active = $5, updated_at = NOW()
		WHERE email = $6
	`, user.PasswordHash,
		nullIfEmpty(user.TOTPSecret),
		user.Role, user.Scope, user.IsActive, user.Email,
	)
	if err != nil {
		return fmt.Errorf("مستودع الإداريين: تحديث مستخدم %s: %w", user.Email, err)
	}
	return nil
}

// UpdateStatus — يحدّث حالة التفعيل لمستخدم إداري
func (r *adminRepo) UpdateStatus(ctx context.Context, id int64, isActive bool) error {
	_, err := r.pool.Exec(ctx, `
		UPDATE admin_users
		SET is_active = $1, updated_at = NOW()
		WHERE id = $2
	`, isActive, id)
	if err != nil {
		return fmt.Errorf("مستودع الإداريين: تحديث حالة المستخدم %d: %w", id, err)
	}
	return nil
}

// List — يعرض قائمة المستخدمين الإداريين
func (r *adminRepo) List(ctx context.Context) ([]model.AdminUser, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, email, password_hash, totp_secret,
		       role, scope, is_active, last_login_at,
		       created_at, updated_at
		FROM admin_users
		ORDER BY email
	`)
	if err != nil {
		return nil, fmt.Errorf("مستودع الإداريين: قائمة المستخدمين: %w", err)
	}
	defer rows.Close()

	var users []model.AdminUser
	for rows.Next() {
		var user model.AdminUser
		var totpSecret *string
		var lastLoginAt *time.Time

		if err := rows.Scan(
			&user.ID, &user.Email, &user.PasswordHash, &totpSecret,
			&user.Role, &user.Scope, &user.IsActive, &lastLoginAt,
			&user.CreatedAt, &user.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("مستودع الإداريين: قراءة صف: %w", err)
		}

		if totpSecret != nil {
			user.TOTPSecret = *totpSecret
		}
		user.LastLoginAt = lastLoginAt

		users = append(users, user)
	}

	return users, nil
}
