package log

type noopLogger struct {
}

func NewNoopLogger() Logger {
	return &noopLogger{}
}

func (l *noopLogger)LogWarn(content string) {
}

func (l *noopLogger)LogError(content string){
}

func (l *noopLogger)LogInfo(content string){
}

func (l *noopLogger)LogDebug(content string){
}

func (l *noopLogger)LogFatal(content string){
}


