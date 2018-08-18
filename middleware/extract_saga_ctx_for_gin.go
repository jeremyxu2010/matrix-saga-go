package middleware

import (
	sagactx "github.com/jeremyxu2010/matrix-saga-go/context"
	"github.com/gin-gonic/gin"
)

func ExtractSagaCtxMiddlewareForGin() gin.HandlerFunc {
	return func(c *gin.Context) {
		sagactx.ExtractFromHttpHeaders(c.Request.Header)
		c.Next()
	}
}
