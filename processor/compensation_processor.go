package processor

import (
	"reflect"
	"github.com/jeremyxu2010/matrix-saga-go/log"
	"fmt"
	"github.com/jeremyxu2010/matrix-saga-go/context"
	"github.com/jeremyxu2010/matrix-saga-go/serializer"
)

type CompensationProcessor struct {
	funcs  map[string]interface{}
	logger log.Logger
	s      serializer.Serializer
}

func NewCompensationProcessor(s serializer.Serializer, logger log.Logger)*CompensationProcessor {
	return &CompensationProcessor{
		funcs: make(map[string]interface{}, 0),
		logger: logger,
		s: s,
	}
}

func (p *CompensationProcessor)RegisterCompensationFunc(fnName string, fn interface{}){
	p.funcs[fnName] = reflect.ValueOf(fn)
}

func (p *CompensationProcessor) ExecuteCompensate(globalTxId string, localTxId string, compensationMethod string, payloads []byte) {
	args, err := p.s.Unserialize(payloads)
	if err != nil {
		p.logger.LogError(fmt.Sprintf("%v", err))
		return
	}
	if v, ok := p.funcs[compensationMethod]; ok {
		if targetFunc, ok := v.(reflect.Value); ok {
			sagaAgentCtx, _ := context.GetSagaAgentContext()
			if sagaAgentCtx == nil {
				sagaAgentCtx = context.NewSagaAgentContext()
				sagaAgentCtx.GlobalTxId = globalTxId
				sagaAgentCtx.LocalTxId = localTxId
				context.SetSagaAgentContext(sagaAgentCtx)
				defer func() {
					context.ClearSagaAgentContext()
				}()
			} else {
				oldGlobalTxId := sagaAgentCtx.GlobalTxId
				oldLocalTxId := sagaAgentCtx.LocalTxId
				defer func() {
					sagaAgentCtx.GlobalTxId = oldGlobalTxId
					sagaAgentCtx.LocalTxId = oldLocalTxId
				}()
				sagaAgentCtx.GlobalTxId = globalTxId
				sagaAgentCtx.LocalTxId = localTxId
			}
			if targetFunc.Type().IsVariadic() {
				targetFunc.CallSlice(args)
			} else {
				targetFunc.Call(args)
			}
		}
	}
}
