package errors

type SagaAgentError struct {
	msg string
}

func NewSagaAgentError(msg string)*SagaAgentError {
	return &SagaAgentError{
		msg: msg,
	}
}

func (e *SagaAgentError) Error()string{
	return e.msg
}
