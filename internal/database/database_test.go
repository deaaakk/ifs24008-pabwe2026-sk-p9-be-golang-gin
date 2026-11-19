package database

import (
	"testing"

	"app/internal/config"
)

func TestDatabase(t *testing.T) {
	c := &config.Config{DBConn: "sqlite", AdminUser: "admin", AdminPass: "admin"}
	db, err := Connect(c)
	if err != nil {
		t.Fatal(err)
	}
	if Migrate(db, c) != nil || Migrate(db, c) != nil {
		t.Fatal("migrate gagal")
	}
	s, _ := db.DB()
	s.Close()
	if Migrate(db, c) == nil {
		t.Fatal("harusnya error")
	}
	bad := &config.Config{DBConn: "pgsql", DBHost: "127.0.0.1", DBPort: "1", DBName: "x", DBUser: "x", DBSSL: "disable"}
	if _, err := Connect(bad); err == nil {
		t.Fatal("harusnya gagal konek")
	}
}
