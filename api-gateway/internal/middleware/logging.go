package middleware

import (
	"log"
	"time"

	"github.com/gin-gonic/gin"
)

// LoggingMiddleware логирует каждый HTTP-запрос: метод, путь,
// статус, длительность. Работает аналогично gRPC-интерсептору.
func LoggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		method := c.Request.Method

		// Обработка запроса
		c.Next()

		duration := time.Since(start)
		status := c.Writer.Status()

		log.Printf("[HTTP] %s %s | %d | %v",
			method, path, status, duration)
	}
}
