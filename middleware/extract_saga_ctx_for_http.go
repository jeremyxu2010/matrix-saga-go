package middleware

import (
	"net/http"
	sagactx "github.com/jeremyxu2010/matrix-saga-go/context"
)

func ExtractSagaCtxMiddlewareForHttp(handler http.HandlerFunc)http.HandlerFunc{
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sagactx.ExtractFromHttpHeaders(r.Header)
		handler(w, r)
	})
}
