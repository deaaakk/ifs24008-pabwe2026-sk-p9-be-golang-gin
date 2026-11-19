package docs

import (
	"strings"
	"testing"

	"github.com/swaggo/swag"
)

func TestDocs(t *testing.T) {
	d, err := swag.ReadDoc()
	if err != nil || !strings.Contains(d, "Delcom Blog API") {
		t.Fatal("doc tidak valid", err)
	}
}
