package context

import (
	"errors"
	"github.com/cosmos72/gls"
	"github.com/jeremyxu2010/matrix-saga-go/constants"
	"net/http"
)

type SagaAgentContext struct {
	GlobalTxId string
	LocalTxId string
}

func NewSagaAgentContext()*SagaAgentContext{
	return &SagaAgentContext{}
}

func (c *SagaAgentContext) Initialize() {
	c.GlobalTxId = idGenerator.NextId()
	c.LocalTxId = c.GlobalTxId
	SetSagaAgentContext(c)
}

func SetSagaAgentContext(c *SagaAgentContext) {
	gls.Set(constants.KEY_SAGA_AGENT_CONTEXT, c)
}

func ClearSagaAgentContext(){
	gls.Del(constants.KEY_SAGA_AGENT_CONTEXT)
}

func (c *SagaAgentContext) NewLocalTxId() {
	c.LocalTxId = idGenerator.NextId()
}

func GetSagaAgentContext()(*SagaAgentContext, error){
	if v, ok := gls.Get(constants.KEY_SAGA_AGENT_CONTEXT); ok {
		if sagaAgentContext, ok := v.(*SagaAgentContext); ok {
			return sagaAgentContext, nil
		} else {
			return nil, errors.New("Can not found SagaAgentContext")
		}
	} else {
		return nil, errors.New("Can not found SagaAgentContext")
	}
}

func ExtractFromHttpHeaders(headers http.Header){
	if _, ok := gls.Get(constants.KEY_SAGA_AGENT_CONTEXT); !ok {
		if len(headers.Get(constants.KEY_GLOBAL_TX_ID_KEY)) > 0 {
			c := NewSagaAgentContext()
			c.GlobalTxId = headers.Get(constants.KEY_GLOBAL_TX_ID_KEY)
			c.LocalTxId = headers.Get(constants.KEY_LOCAL_TX_ID_KEY)
			SetSagaAgentContext(c)
		}
	}
}

func InjectIntoHttpHeaders(headers http.Header){
	c, _ := GetSagaAgentContext()
	if c != nil {
		headers.Set(constants.KEY_GLOBAL_TX_ID_KEY, c.GlobalTxId)
		headers.Set(constants.KEY_GLOBAL_TX_ID_KEY, c.LocalTxId)
	}
}

func MustGetSagaAgentContext()*SagaAgentContext{
	sagaAgentContext, err := GetSagaAgentContext()
	if err != nil {
		panic(err)
	}
	return sagaAgentContext
}
