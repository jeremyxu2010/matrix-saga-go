package saga

import (
	"context"
	"errors"
	"fmt"
	"github.com/jeremyxu2010/matrix-saga-go/config"
	"github.com/jeremyxu2010/matrix-saga-go/constants"
	sagactx "github.com/jeremyxu2010/matrix-saga-go/context"
	"github.com/jeremyxu2010/matrix-saga-go/degorator"
	"github.com/jeremyxu2010/matrix-saga-go/log"
	"github.com/jeremyxu2010/matrix-saga-go/metadata"
	"github.com/jeremyxu2010/matrix-saga-go/processor"
	"github.com/jeremyxu2010/matrix-saga-go/serializer"
	"github.com/jeremyxu2010/matrix-saga-go/transport"
	"github.com/jeremyxu2010/matrix-saga-go/utils"
	"reflect"
	"sync"
)

var (
	initOnce sync.Once

	compensationProcessor *processor.CompensationProcessor
	serviceConfig         *config.ServiceConfig
	transportContractor   *transport.TransportContractor
	s                     serializer.Serializer

	logger log.Logger
)

func init() {
	s = serializer.NewGobSerializer()
	compensationProcessor = processor.NewCompensationProcessor(s, logger)
}

func DecorateSagaStartMethod(sagaStartPtr interface{}, target interface{}, timeout int, autoClose bool) error {
	sagaStartInjectBefore := func(ctx context.Context) error {
		sagaAgentCtx := sagactx.NewSagaAgentContext()
		sagaAgentCtx.Initialize()
		_, err := transportContractor.SendSagaStartedEvent(sagaAgentCtx, timeout)
		if err != nil {
			transportContractor.SendTxAbortedEvent(sagaAgentCtx, "", utils.GetFnName(target), err)
			return err
		}
		logger.LogDebug(fmt.Sprintf("Initialized context %v before execution of method %v", sagaAgentCtx, target))
		return nil
	}

	sagaStartInjectAfter := func(ctx context.Context) error {
		defer func() {
			if autoClose {
				sagactx.ClearSagaAgentContext()
			}
		}()
		sagaAgentCtx, err := sagactx.GetSagaAgentContext()
		if err != nil {
			transportContractor.SendTxAbortedEvent(sagaAgentCtx, "", utils.GetFnName(target), err)
			logger.LogError(fmt.Sprintf("Transaction %v failed.", sagaAgentCtx))
			return err
		}
		if autoClose {
			return sendSagaEndEvent(ctx, sagaAgentCtx, target)
		} else {
			logger.LogDebug(fmt.Sprintf("Transaction with context %v is not finished in the SagaStarted annotated method.", sagaAgentCtx))
			return nil
		}
	}

	err := degorator.Decorate(sagaStartPtr, target, sagaStartInjectBefore, sagaStartInjectAfter)
	if err != nil {
		return err
	}
	return nil
}

func DecorateSagaEndMethod(sagaStartPtr interface{}, target interface{}) error {
	sagaEndInjectBefore := func(ctx context.Context) error {
		return nil
	}
	sagaEndInjectAfter := func(ctx context.Context) error {
		defer func() {
			sagactx.ClearSagaAgentContext()
		}()
		sagaAgentCtx, err := sagactx.GetSagaAgentContext()
		if err != nil {
			transportContractor.SendTxAbortedEvent(sagaAgentCtx, "", utils.GetFnName(target), err)
			logger.LogError(fmt.Sprintf("Transaction %v failed.", sagaAgentCtx))
			return err
		}
		return sendSagaEndEvent(ctx, sagaAgentCtx, target)
	}
	err := degorator.Decorate(sagaStartPtr, target, sagaEndInjectBefore, sagaEndInjectAfter)
	if err != nil {
		return err
	}
	return nil
}

func sendSagaEndEvent(ctx context.Context, sagaAgentCtx *sagactx.SagaAgentContext, target interface{}) error {
	if m, ok := metadata.FromContext(ctx); ok {
		if m[constants.KEY_FUNCTION_CALL_ERROR] != nil {
			if v, ok := m[constants.KEY_FUNCTION_CALL_ERROR].(error); ok {
				err := v.(error)
				transportContractor.SendTxAbortedEvent(sagaAgentCtx, "", utils.GetFnName(target), err)
				logger.LogError(fmt.Sprintf("Transaction %v failed.", sagaAgentCtx))
				return err
			}
		}
	}
	aborted, err := transportContractor.SendSagaEndedEvent(sagaAgentCtx)
	if err != nil {
		transportContractor.SendTxAbortedEvent(sagaAgentCtx, "", utils.GetFnName(target), err)
		logger.LogError(fmt.Sprintf("Transaction %v failed.", sagaAgentCtx))
		return err
	}

	if aborted {
		return errors.New(fmt.Sprintf("transaction %s is aborted", sagaAgentCtx.GlobalTxId))
	}
	logger.LogDebug(fmt.Sprintf("Transaction with context %v has finished.", sagaAgentCtx))
	return nil
}

func DecorateCompensableMethod(compensablePtr interface{}, target interface{}, compensed interface{}, timeout int) error {

	compensedFuncName := utils.GetFnName(compensed)

	compensationProcessor.RegisterCompensationFunc(compensedFuncName, compensed)

	compensableInjectBefore := func(ctx context.Context) error {
		sagaAgentCtx, err := sagactx.GetSagaAgentContext()
		if err != nil {
			return err
		}
		parentLocalTxId := sagaAgentCtx.LocalTxId
		if m, ok := metadata.FromContext(ctx); ok {
			m[constants.KEY_PARENT_LOCAL_TX_ID] = parentLocalTxId
		}
		sagaAgentCtx.NewLocalTxId()
		logger.LogDebug(fmt.Sprintf("Updated context %v for compensable method %v", sagaAgentCtx, target))
		logger.LogDebug(fmt.Sprintf("Intercepting compensable method %v with context %v", target, sagaAgentCtx))
		var args []reflect.Value
		if m, ok := metadata.FromContext(ctx); ok {
			if v, ok := m[constants.KEY_FUNCTION_CALL_ARGS].([]reflect.Value); ok {
				args = v
			}
		}
		aborted, err := transportContractor.SendTxStartedEvent(sagaAgentCtx, parentLocalTxId, compensedFuncName, timeout, args)
		if err != nil {
			sagaAgentCtx.LocalTxId = parentLocalTxId
			logger.LogDebug(fmt.Sprintf("Restored context back to %v", sagaAgentCtx))
			return err
		}
		if aborted {
			abortedLocalTxId := sagaAgentCtx.LocalTxId
			sagaAgentCtx.LocalTxId = parentLocalTxId
			logger.LogDebug(fmt.Sprintf("Restored context back to %v", sagaAgentCtx))
			return errors.New(fmt.Sprintf("Abort sub transaction %s because global transaction %s has already aborted.", abortedLocalTxId, sagaAgentCtx.GlobalTxId))
		}
		return nil
	}

	compensableInjectAfter := func(ctx context.Context) error {
		var parentLocalTxId string
		if m, ok := metadata.FromContext(ctx); ok {
			if v, ok := m[constants.KEY_PARENT_LOCAL_TX_ID].(string); ok {
				parentLocalTxId = v
			}
		}
		sagaAgentCtx, err := sagactx.GetSagaAgentContext()
		if err != nil {
			return err
		}
		if m, ok := metadata.FromContext(ctx); ok {
			if m[constants.KEY_FUNCTION_CALL_ERROR] != nil {
				if v, ok := m[constants.KEY_FUNCTION_CALL_ERROR].(error); ok {
					err = v.(error)
					transportContractor.SendTxAbortedEvent(sagaAgentCtx, "", utils.GetFnName(target), err)
					logger.LogError(fmt.Sprintf("Transaction %v failed.", sagaAgentCtx))
					return err
				}
			}
		}
		aborted, err := transportContractor.SendTxEndedEvent(sagaAgentCtx, parentLocalTxId, compensedFuncName)
		if err != nil {
			transportContractor.SendTxAbortedEvent(sagaAgentCtx, "", utils.GetFnName(target), err)
			logger.LogError(fmt.Sprintf("Transaction %v failed.", sagaAgentCtx))
			sagaAgentCtx.LocalTxId = parentLocalTxId
			logger.LogDebug(fmt.Sprintf("Restored context back to %v", sagaAgentCtx))
			return err
		}

		if aborted {
			sagaAgentCtx.LocalTxId = parentLocalTxId
			logger.LogDebug(fmt.Sprintf("Restored context back to %v", sagaAgentCtx))
			return errors.New(fmt.Sprintf("transaction %s is aborted", parentLocalTxId))
		}
		sagaAgentCtx.LocalTxId = parentLocalTxId
		logger.LogDebug(fmt.Sprintf("Restored context back to %v", sagaAgentCtx))
		return nil
	}

	err := degorator.Decorate(compensablePtr, target, compensableInjectBefore, compensableInjectAfter)
	if err != nil {
		return err
	}
	return nil
}

func InitSagaAgent(serviceName string, coordinatorAddress string, l log.Logger) error {
	initOnce.Do(func() {
		logger = l
		if logger == nil {
			logger = log.NewNoopLogger()
		}
		serviceConfig = config.NewServiceConfig(serviceName)
		transportContractor = transport.NewTransportContractor(coordinatorAddress, serviceConfig, compensationProcessor, s, logger)
	})
	err := transportContractor.Connect()
	if err != nil {
		return err
	}
	return nil
}
