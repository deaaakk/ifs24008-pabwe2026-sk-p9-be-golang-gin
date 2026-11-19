package config

import "testing"

func TestLoad(t *testing.T) {
	t.Setenv("APP_PORT", "9999")
	t.Setenv("DB_HOST", "")
	c := Load()
	if c.Port != "9999" || c.DBHost != "127.0.0.1" {
		t.Fatal("config salah", c)
	}
}
