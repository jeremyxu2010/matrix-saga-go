package transport

import (
	"google.golang.org/grpc"
	"time"
	"os"
	"os/signal"
	"syscall"
	"reflect"
	"github.com/jeremyxu2010/matrix-saga-go/saga_grpc"
	"context"
	"github.com/jeremyxu2010/matrix-saga-go/constants"
	"github.com/jeremyxu2010/matrix-saga-go/processor"
	sagactx "github.com/jeremyxu2010/matrix-saga-go/context"
	"github.com/jeremyxu2010/matrix-saga-go/log"
	"fmt"
	"github.com/jeremyxu2010/matrix-saga-go/config"
	"github.com/jeremyxu2010/matrix-saga-go/serializer"
)

type TransportContractor struct {
	coordinatorAddress    string
	txEventServiceClient  saga_grpc.TxEventServiceClient
	onConnectedClient     saga_grpc.TxEventService_OnConnectedClient
	serviceConfig         *config.ServiceConfig
	compensationProcessor *processor.CompensationProcessor
	logger                log.Logger
	pendingTasks          chan func()
	s                     serializer.Serializer
}

func NewTransportContractor(coordinatorAddress string, serviceConfig *config.ServiceConfig, compensationProcessor *processor.CompensationProcessor, s serializer.Serializer, logger log.Logger) *TransportContractor {
	transportContractor := &TransportContractor{
		coordinatorAddress:    coordinatorAddress,
		serviceConfig:         serviceConfig,
		compensationProcessor: compensationProcessor,
		logger:                logger,
		pendingTasks:          make(chan func(), 0),
		s: s,
	}
	go transportContractor.scheduleProcessReconnectTask()
	return transportContractor
}

func (c *TransportContractor) Connect() error {
	err := c.initClient()
	if err != nil {
		return err
	}
	c.onConnected()
	return nil
}

func (c *TransportContractor) initClient() error {
	ctx, cancel := context.WithTimeout(context.Background(), constants.GRPC_COMMUNICATE_TIMEOUT)
	defer cancel()
	c.logger.LogInfo(fmt.Sprintf("connect coordinator with address[%s]", c.coordinatorAddress))
	conn, err := grpc.DialContext(ctx, c.coordinatorAddress, grpc.WithInsecure())
	if err != nil {
		return err
	}
	c.txEventServiceClient = saga_grpc.NewTxEventServiceClient(conn)
	return nil
}

func (c *TransportContractor) onConnected() error {
	//ctx, cancel := context.WithTimeout(context.Background(), constants.GRPC_COMMUNICATE_TIMEOUT)
	//defer cancel()
	grpcServiceConfig := &saga_grpc.GrpcServiceConfig{
		ServiceName: c.serviceConfig.ServiceName,
		InstanceId:  c.serviceConfig.InstanceId,
	}
	var err error
	c.onConnectedClient, err = c.txEventServiceClient.OnConnected(context.Background(), grpcServiceConfig)
	if err != nil {
		return err
	}
	stopped := false
	go func() {
		for !stopped {
			grpcCompensateCommand, err := c.onConnectedClient.Recv()
			if err != nil {
				if !stopped {
					c.logger.LogError(fmt.Sprintf("failed to process grpc compensate command, err: %v", err))
					// create reconnect task
					c.pendingTasks <- c.reconnectTask
				}
				continue
			}
			c.logger.LogInfo(fmt.Sprintf("Received compensate command, global tx id: %s, local tx id: %s, compensation method: %s",
				grpcCompensateCommand.GlobalTxId, grpcCompensateCommand.LocalTxId, grpcCompensateCommand.CompensationMethod))
			c.compensationProcessor.ExecuteCompensate(grpcCompensateCommand.GlobalTxId, grpcCompensateCommand.LocalTxId, grpcCompensateCommand.CompensationMethod, grpcCompensateCommand.Payloads)
			c.sendTxCompensatedEvent(grpcCompensateCommand.GlobalTxId, grpcCompensateCommand.LocalTxId, grpcCompensateCommand.ParentTxId, grpcCompensateCommand.CompensationMethod)
		}
	}()
	go func() {
		s := make(chan os.Signal)
		signal.Notify(s, syscall.SIGINT, syscall.SIGTERM)
		<-s
		stopped = true
		ctx, cancel := context.WithTimeout(context.Background(), constants.GRPC_COMMUNICATE_TIMEOUT)
		defer cancel()
		c.txEventServiceClient.OnDisconnected(ctx, grpcServiceConfig)

	}()
	return nil
}

func (c *TransportContractor) sendTxCompensatedEvent(globalTxId string, localTxId string, parentTxId string, compensationMethod string) {
	e := &saga_grpc.GrpcTxEvent{
		ServiceName:        c.serviceConfig.ServiceName,
		InstanceId:         c.serviceConfig.InstanceId,
		Timestamp:          time.Now().UnixNano() / int64(time.Millisecond),
		GlobalTxId:         globalTxId,
		LocalTxId:          localTxId,
		ParentTxId:         parentTxId,
		Type:               constants.EVENT_NAME_TXCOMPENSATEDEVENT,
		Timeout:            int32(0),
		CompensationMethod: compensationMethod,
		RetryMethod:        "",
		Retries:            0,
		Payloads:           nil,
	}
	ctx, cancel := context.WithTimeout(context.Background(), constants.GRPC_COMMUNICATE_TIMEOUT)
	defer cancel()
	c.txEventServiceClient.OnTxEvent(ctx, e)
}

func (c *TransportContractor) SendSagaStartedEvent(sagaAgentCtx *sagactx.SagaAgentContext, timeout int) (bool, error) {
	e := &saga_grpc.GrpcTxEvent{
		ServiceName:        c.serviceConfig.ServiceName,
		InstanceId:         c.serviceConfig.InstanceId,
		Timestamp:          time.Now().UnixNano() / int64(time.Millisecond),
		GlobalTxId:         sagaAgentCtx.GlobalTxId,
		LocalTxId:          sagaAgentCtx.LocalTxId,
		ParentTxId:         "",
		Type:               constants.EVENT_NAME_SAGASTARTEDEVENT,
		Timeout:            int32(timeout),
		CompensationMethod: "",
		RetryMethod:        "",
		Retries:            0,
		Payloads:           nil,
	}
	ctx, cancel := context.WithTimeout(context.Background(), constants.GRPC_COMMUNICATE_TIMEOUT)
	defer cancel()
	grpcAck, err := c.txEventServiceClient.OnTxEvent(ctx, e)
	if err != nil {
		return false, err
	}
	return grpcAck.Aborted, nil
}

func (c *TransportContractor) SendSagaEndedEvent(sagaAgentCtx *sagactx.SagaAgentContext) (bool, error) {
	e := &saga_grpc.GrpcTxEvent{
		ServiceName:        c.serviceConfig.ServiceName,
		InstanceId:         c.serviceConfig.InstanceId,
		Timestamp:          time.Now().UnixNano() / int64(time.Millisecond),
		GlobalTxId:         sagaAgentCtx.GlobalTxId,
		LocalTxId:          sagaAgentCtx.LocalTxId,
		ParentTxId:         "",
		Type:               constants.EVENT_NAME_SAGAENDEDEVENT,
		Timeout:            int32(0),
		CompensationMethod: "",
		RetryMethod:        "",
		Retries:            0,
		Payloads:           nil,
	}
	ctx, cancel := context.WithTimeout(context.Background(), constants.GRPC_COMMUNICATE_TIMEOUT)
	defer cancel()
	grpcAck, err := c.txEventServiceClient.OnTxEvent(ctx, e)
	if err != nil {
		return false, err
	}
	return grpcAck.Aborted, nil
}

func (c *TransportContractor) SendTxStartedEvent(sagaAgentCtx *sagactx.SagaAgentContext, parentTxId string, compensationMethod string, timeout int, args []reflect.Value) (bool, error) {
	b, err := c.s.Serialize(args)
	if err != nil {
		c.logger.LogError(fmt.Sprintf("%v", err))
		return false, err
	}
	e := &saga_grpc.GrpcTxEvent{
		ServiceName:        c.serviceConfig.ServiceName,
		InstanceId:         c.serviceConfig.InstanceId,
		Timestamp:          time.Now().UnixNano() / int64(time.Millisecond),
		GlobalTxId:         sagaAgentCtx.GlobalTxId,
		LocalTxId:          sagaAgentCtx.LocalTxId,
		ParentTxId:         parentTxId,
		Type:               constants.EVENT_NAME_TXSTARTEDEVENT,
		Timeout:            int32(timeout),
		CompensationMethod: compensationMethod,
		RetryMethod:        "",
		Retries:            0,
		Payloads:           b,
	}
	ctx, cancel := context.WithTimeout(context.Background(), constants.GRPC_COMMUNICATE_TIMEOUT)
	defer cancel()
	grpcAck, err := c.txEventServiceClient.OnTxEvent(ctx, e)
	if err != nil {
		return false, err
	}
	return grpcAck.Aborted, nil
}

func (c *TransportContractor) SendTxEndedEvent(sagaAgentCtx *sagactx.SagaAgentContext, parentTxId string, compensationMethod string) (bool, error) {
	e := &saga_grpc.GrpcTxEvent{
		ServiceName:        c.serviceConfig.ServiceName,
		InstanceId:         c.serviceConfig.InstanceId,
		Timestamp:          time.Now().UnixNano() / int64(time.Millisecond),
		GlobalTxId:         sagaAgentCtx.GlobalTxId,
		LocalTxId:          sagaAgentCtx.LocalTxId,
		ParentTxId:         parentTxId,
		Type:               constants.EVENT_NAME_TXENDEDEVENT,
		Timeout:            int32(0),
		CompensationMethod: compensationMethod,
		RetryMethod:        "",
		Retries:            0,
		Payloads:           nil,
	}
	ctx, cancel := context.WithTimeout(context.Background(), constants.GRPC_COMMUNICATE_TIMEOUT)
	defer cancel()
	grpcAck, err := c.txEventServiceClient.OnTxEvent(ctx, e)
	if err != nil {
		return false, err
	}
	return grpcAck.Aborted, nil
}

func (c *TransportContractor) SendTxAbortedEvent(sagaAgentCtx *sagactx.SagaAgentContext, parentTxId string, compensationMethod string, err error) (bool, error) {
	errStr := err.Error()
	if len(errStr) > constants.PAYLOADS_MAX_LENGTH {
		errStr = errStr[0:constants.PAYLOADS_MAX_LENGTH]
	}
	e := &saga_grpc.GrpcTxEvent{
		ServiceName:        c.serviceConfig.ServiceName,
		InstanceId:         c.serviceConfig.InstanceId,
		Timestamp:          time.Now().UnixNano() / int64(time.Millisecond),
		GlobalTxId:         sagaAgentCtx.GlobalTxId,
		LocalTxId:          sagaAgentCtx.LocalTxId,
		ParentTxId:         parentTxId,
		Type:               constants.EVENT_NAME_TXABORTEDEVENT,
		Timeout:            int32(0),
		CompensationMethod: compensationMethod,
		RetryMethod:        "",
		Retries:            0,
		Payloads:           []byte(errStr),
	}
	ctx, cancel := context.WithTimeout(context.Background(), constants.GRPC_COMMUNICATE_TIMEOUT)
	defer cancel()
	grpcAck, err := c.txEventServiceClient.OnTxEvent(ctx, e)
	if err != nil {
		return false, err
	}
	return grpcAck.Aborted, nil
}

func (c *TransportContractor) scheduleProcessReconnectTask() {
	for {
		select {
		case taskFn := <- c.pendingTasks:
			taskFn()
		case <- time.Tick(constants.GRPC_RECONNECT_DELAY):
			continue
		}
	}
}

func (c *TransportContractor) reconnectTask(){
	grpcServiceConfig := &saga_grpc.GrpcServiceConfig{
		ServiceName: c.serviceConfig.ServiceName,
		InstanceId:  c.serviceConfig.InstanceId,
	}
	c.logger.LogInfo(fmt.Sprintf("Retry connecting to coordinator at %s", c.coordinatorAddress))
	ctx, cancel := context.WithTimeout(context.Background(), constants.GRPC_COMMUNICATE_TIMEOUT)
	defer cancel()
	c.txEventServiceClient.OnDisconnected(ctx, grpcServiceConfig)
	var err error
	err = c.initClient()
	if err != nil {
		c.logger.LogError(fmt.Sprintf("Failed to reconnect to coordinator at %s, error: %v", c.coordinatorAddress, err))
		// create reconnect task
		c.pendingTasks <- c.reconnectTask
		return
	} else {
		c.logger.LogInfo(fmt.Sprintf("Retry connecting to coordinator at %s is successful", c.coordinatorAddress))
		return
	}
	c.onConnectedClient, err = c.txEventServiceClient.OnConnected(context.Background(), grpcServiceConfig)
	if err != nil {
		c.logger.LogError(fmt.Sprintf("Failed to reconnect to coordinator at %s, error: %v", c.coordinatorAddress, err))
		// create reconnect task
		c.pendingTasks <- c.reconnectTask
		return
	} else {
		c.logger.LogInfo(fmt.Sprintf("Retry connecting to coordinator at %s is successful", c.coordinatorAddress))
		return
	}
}
