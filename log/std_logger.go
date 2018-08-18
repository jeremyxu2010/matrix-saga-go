package log

import (
	"log"
	"os"
)

type stdLogger struct {
	logger *log.Logger
}

func NewStdLogger() Logger {
	logger := log.New(os.Stdout, "logger: ", log.Llongfile|log.LstdFlags)
	return &stdLogger{
		logger: logger,
	}
}

func (l *stdLogger)LogWarn(content string) {
	l.logger.Println(content)
}

func (l *stdLogger)LogError(content string){
	l.logger.Println(content)
}

func (l *stdLogger)LogInfo(content string){
	l.logger.Println(content)
}

func (l *stdLogger)LogDebug(content string){
	l.logger.Println(content)
}

func (l *stdLogger)LogFatal(content string){
	l.logger.Fatalln(content)
}


