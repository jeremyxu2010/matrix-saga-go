package constants

import "time"

const (
	GRPC_COMMUNICATE_TIMEOUT = time.Second * 5
	GRPC_RECONNECT_DELAY = time.Second * 10

	PAYLOADS_MAX_LENGTH = 10240
)
