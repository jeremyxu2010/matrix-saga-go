package log

type Logger interface {
	LogWarn(content string)
	LogError(content string)
	LogInfo(content string)
	LogDebug(content string)
	LogFatal(content string)
}
