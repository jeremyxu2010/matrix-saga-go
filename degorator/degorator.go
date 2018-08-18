package degorator

import (
	"fmt"
	"reflect"
	"errors"
	"context"
	"github.com/jeremyxu2010/matrix-saga-go/constants"
	"github.com/jeremyxu2010/matrix-saga-go/metadata"
)

var CONTEXT_TYPE_NAME = "context.Context"
var ERROR_TYPE_NAME = "error"

// Decorate injects two functions(injectedBefore & injectedAfter) into the target function.
// The argument decorated is the function after decoration.
// The argument target is the function to be decorated.
// The argument before is the function to be injected before the target function.
// The argument after is the function to be injected after the target function.
func Decorate(decorated interface{}, target interface{}, before interface{}, after interface{}) (err error) {
	var targetFunc reflect.Value
	var decoratedFunc reflect.Value
	var beforeFunc reflect.Value
	var afterFunc reflect.Value

	decoratedFunc, err = checkFPTR(decorated)
	if err != nil {
		return
	}

	targetFunc = reflect.ValueOf(target)
	if targetFunc.Kind() != reflect.Func {
		err = fmt.Errorf("Input target para is not a function.")
		return
	}

	beforeFunc, afterFunc, err = checkInjection(targetFunc.Type(), before, after)
	if err != nil {
		return
	}

	decoratedFunc.Set(reflect.MakeFunc(targetFunc.Type(), func(in []reflect.Value) (out []reflect.Value) {
		ctx := context.Background()
		ctx = metadata.NewContext(ctx, make(metadata.Metadata))
		if targetFunc.Type().IsVariadic() {
			if before != nil {
				if m, ok := metadata.FromContext(ctx); ok {
					m[constants.KEY_FUNCTION_CALL_ARGS] = in
				}
				beforeOut := beforeFunc.CallSlice([]reflect.Value{reflect.ValueOf(ctx)})
				if !beforeOut[0].IsNil() {
					return
				}
			}
			out, err = safeCallSlice(targetFunc, in)
			if after != nil {
				if m, ok := metadata.FromContext(ctx); ok {
					m[constants.KEY_FUNCTION_CALL_ERROR] = err
				}
				afterOut := afterFunc.CallSlice([]reflect.Value{reflect.ValueOf(ctx)})
				if !afterOut[0].IsNil() {
					return
				}
			}
		} else {
			if before != nil {
				if m, ok := metadata.FromContext(ctx); ok {
					m[constants.KEY_FUNCTION_CALL_ARGS] = in
				}
				beforeOut := beforeFunc.Call([]reflect.Value{reflect.ValueOf(ctx)})
				if !beforeOut[0].IsNil() {
					return
				}
			}
			out, err = safeCall(targetFunc, in)
			if after != nil {
				if m, ok := metadata.FromContext(ctx); ok {
					m[constants.KEY_FUNCTION_CALL_ERROR] = err
				}
				afterOut := afterFunc.Call([]reflect.Value{reflect.ValueOf(ctx)})
				if !afterOut[0].IsNil() {
					return
				}
			}
		}
		return
	}))
	return
}

func safeCallSlice(targetFunc reflect.Value, in []reflect.Value) (out []reflect.Value, err error) {
	defer func() {
		cause := recover()
		switch cause.(type) {
		case error:
			err = cause.(error)
		case string:
			err = errors.New(cause.(string))
		}
	}()
	out = targetFunc.CallSlice(in)
	if len(out) > 0 && (out[len(out) - 1].Type().String() == ERROR_TYPE_NAME) {
		if out[len(out) - 1].Interface() != nil {
			err = out[len(out)-1].Interface().(error)
		}
	}
	return
}

func safeCall(targetFunc reflect.Value, in []reflect.Value) (out []reflect.Value, err error) {
	defer func() {
		cause := recover()
		switch cause.(type) {
		case error:
			err = cause.(error)
		case string:
			err = errors.New(cause.(string))
		}
	}()
	out = targetFunc.Call(in)
	if len(out) > 0 && (out[len(out) - 1].Type().String() == ERROR_TYPE_NAME) {
		if out[len(out) - 1].Interface() != nil {
			err = out[len(out) - 1].Interface().(error)
		}
	}
	return
}

func checkFPTR(fptr interface{}) (function reflect.Value, err error) {
	if fptr == nil {
		err = fmt.Errorf("Input para is nil.")
		return
	}
	if reflect.TypeOf(fptr).Kind() != reflect.Ptr {
		err = fmt.Errorf("Input para is not a pointer.")
		return
	}
	function = reflect.ValueOf(fptr).Elem()
	if function.Kind() != reflect.Func {
		err = fmt.Errorf("Input para is not a pointer to a function.")
		return
	}
	return
}

func checkInjection(targetType reflect.Type, before interface{}, after interface{}) (beforeFunc reflect.Value, afterFunc reflect.Value, err error) {
	if before != nil {
		beforeFunc = reflect.ValueOf(before)
		if beforeFunc.Kind() != reflect.Func {
			err = fmt.Errorf("Only a function can be injected before.")
			return
		}
		if beforeFunc.Type().NumIn() != 1 {
			err = fmt.Errorf("The input para number of the function injected before must be one.")
			return
		}
		if beforeFunc.Type().In(0).String() != CONTEXT_TYPE_NAME {
			err = fmt.Errorf("The input para type of the function injected before must be context.Context.")
			return
		}
		if beforeFunc.Type().NumOut() != 1 {
			err = fmt.Errorf("The output para number of the function injected before must be one.")
			return
		}
		if beforeFunc.Type().Out(0).String() != ERROR_TYPE_NAME {
			err = fmt.Errorf("The output para type of the function injected before must be error.")
			return
		}
	}
	if after != nil {
		afterFunc = reflect.ValueOf(after)
		if afterFunc.Kind() != reflect.Func {
			err = fmt.Errorf("Only a function can be injected after.")
			return
		}
		if afterFunc.Type().NumIn() != 1 {
			err = fmt.Errorf("The input para number of the function injected after must be one.")
			return
		}
		if afterFunc.Type().In(0).String() != CONTEXT_TYPE_NAME {
			err = fmt.Errorf("The input para types of the function injected after must be context.Context.")
			return
		}
		if afterFunc.Type().NumOut() != 1 {
			err = fmt.Errorf("The output para number of the function injected after must be one.")
			return
		}
		if afterFunc.Type().Out(0).String() != ERROR_TYPE_NAME {
			err = fmt.Errorf("The output para type of the function injected after must be error.")
			return
		}
	}
	return
}

func checkDecorator(decorator interface{}) (decoFunc reflect.Value, err error) {
	decoFunc, err = checkFPTR(decorator)
	if err != nil {
		return
	}
	if decoFunc.Type().NumIn() != 1 || decoFunc.Type().NumOut() != 1 {
		err = fmt.Errorf("Decorator function must have one input para and one output para.")
		return
	}
	if decoFunc.Type().In(0).Kind() != reflect.Func || decoFunc.Type().Out(0).Kind() != reflect.Func {
		err = fmt.Errorf("Decorator function's input para type and output para type must be function type.")
		return
	}
	if decoFunc.Type().In(0).NumIn() != decoFunc.Type().Out(0).NumIn() {
		err = fmt.Errorf("Decoratee function and decorated function must have same input para number.")
		return
	}
	for i := 0; i < decoFunc.Type().In(0).NumIn(); i++ {
		if decoFunc.Type().In(0).In(i) != decoFunc.Type().Out(0).In(i) {
			err = fmt.Errorf("Decoratee function and decorated function must have same input para type.")
			return
		}
	}
	if decoFunc.Type().In(0).NumOut() != decoFunc.Type().Out(0).NumOut() {
		err = fmt.Errorf("Decoratee function  and decorated function must have same output para number.")
		return
	}
	for i := 0; i < decoFunc.Type().In(0).NumOut(); i++ {
		if decoFunc.Type().In(0).Out(i) != decoFunc.Type().Out(0).Out(i) {
			err = fmt.Errorf("Decoratee function  and decorated function must have same output para type.")
			return
		}
	}
	return
}
