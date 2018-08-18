package config

import (
	"strings"
	"github.com/jeremyxu2010/matrix-saga-go/utils"
)

func NewServiceConfig(serviceName string) *ServiceConfig{
	hostAddress, err := utils.GetFirstNotLoopbackIPv4Address()
	if err != nil {
		panic(err)
	}
	instanceId := strings.Join([]string{serviceName, hostAddress}, "-")
	return &ServiceConfig{
		ServiceName: serviceName,
		InstanceId:  instanceId,
	}
}

type ServiceConfig struct{
	ServiceName string
	InstanceId  string
}
