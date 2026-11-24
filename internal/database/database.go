package database

import (
	"fmt"
	"time"

	"app/internal/config"

	"github.com/glebarez/sqlite"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type User struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Username  string    `gorm:"uniqueIndex;size:100" json:"username"`
	Email     string    `gorm:"uniqueIndex;size:255" json:"email"`
	Password  string    `json:"-"`
	Role      string    `gorm:"size:20;default:member" json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

type Post struct {
	ID          uint       `gorm:"primaryKey" json:"id"`
	UserID      uint       `gorm:"not null;index" json:"user_id"`
	User        User       `gorm:"constraint:OnDelete:CASCADE" json:"author"`
	Title       string     `gorm:"size:255;not null" json:"title"`
	Slug        string     `gorm:"size:255;uniqueIndex;not null" json:"slug"`
	Category    string     `gorm:"size:100" json:"category"`
	Excerpt     string     `json:"excerpt"`
	Content     string     `json:"content"`
	CoverImage  string     `gorm:"column:cover_image_path" json:"cover_image_path"`
	IsPublished bool       `json:"is_published"`
	PublishedAt *time.Time `json:"published_at"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	ReadingTime int        `gorm:"-" json:"reading_time"`
	AuthorName  string     `gorm:"-" json:"author_name"`
}

func Connect(c *config.Config) (*gorm.DB, error) {
	var d gorm.Dialector
	if c.DBConn == "sqlite" {
		d = sqlite.Open("file::memory:")
	} else {
		d = postgres.Open(fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=Asia/Jakarta",
			c.DBHost, c.DBUser, c.DBPass, c.DBName, c.DBPort, c.DBSSL))
	}
	db, err := gorm.Open(d, &gorm.Config{})
	if err != nil {
		return nil, err
	}
	if c.DBConn == "sqlite" {
		s, _ := db.DB()
		s.SetMaxOpenConns(1)
	}
	return db, nil
}

func Migrate(db *gorm.DB, c *config.Config) error {
	if err := db.AutoMigrate(&User{}, &Post{}); err != nil {
		return err
	}
	var n int64
	db.Model(&User{}).Where("role = ?", "admin").Count(&n)
	if n == 0 {
		h, _ := bcrypt.GenerateFromPassword([]byte(c.AdminPass), bcrypt.DefaultCost)
		return db.Create(&User{Username: c.AdminUser, Email: c.AdminUser + "@delcom.org", Password: string(h), Role: "admin"}).Error
	}
	return nil
}
