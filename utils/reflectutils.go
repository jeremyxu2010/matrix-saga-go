package utils

import (
	"reflect"
	"runtime"
)

func GetFnName(fn interface{}) string {
	compensedFunc := reflect.ValueOf(fn)
	fnName := runtime.FuncForPC(compensedFunc.Pointer()).Name()
	return fnName
}
