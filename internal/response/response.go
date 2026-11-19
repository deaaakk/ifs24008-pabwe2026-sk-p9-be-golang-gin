package response

import "github.com/gin-gonic/gin"

func Success(c *gin.Context, code int, msg string, data any) {
	c.JSON(code, gin.H{"status": "success", "message": msg, "data": data})
}
func Fail(c *gin.Context, code int, msg string) {
	c.JSON(code, gin.H{"status": "fail", "message": msg})
}
func Error(c *gin.Context, code int, msg string) {
	c.JSON(code, gin.H{"status": "error", "message": msg})
}
