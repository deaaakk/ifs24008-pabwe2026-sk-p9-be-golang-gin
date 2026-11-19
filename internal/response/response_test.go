package response

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestResponses(t *testing.T) {
	for i, f := range []func(*gin.Context){
		func(c *gin.Context) { Success(c, 200, "ok", 1) },
		func(c *gin.Context) { Fail(c, 400, "x") },
		func(c *gin.Context) { Error(c, 500, "x") },
	} {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		f(c)
		if w.Code != []int{200, 400, 500}[i] {
			t.Fatal(i, w.Code)
		}
	}
}
