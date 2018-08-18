## 简述

Saga事务方案目前已有较成熟的开源实现[servicecomb-saga](https://github.com/apache/incubator-servicecomb-saga)，但servicecomb-saga是采用java语言实现的，对于golang语言的业务项目来说接入servicecomb-saga方案有一些困难，本项目尝试为servicecomb-saga实现一个golang语言的omega，以帮助golang语言的业务项目接入。

本程序库参考[servicecomb-saga的omega模块](https://github.com/apache/incubator-servicecomb-saga/tree/master/omega)，完全兼容servicecomb-saga java版实现，可与java版互操作。

## 使用方法

可参考[test示例](./test)，下面说明主要步骤

假设原有的业务代码如下：

`test/old/demo.go`

```go
package main

import (
	"fmt"
	"time"
	"os"
	"os/signal"
	"syscall"
)

var (
	BALANCES map[string]int
)

func init()  {
	initDatas()
}

func initDatas(){
	BALANCES = make(map[string]int, 0)
	BALANCES["foo"] = 500
	BALANCES["bar"] = 500
}

func TransferMoney() error {
	err := TransferOut("foo", 100)
	if err != nil {
		return err
	}
	err = TransferIn("bar", 100)
	if err != nil {
		return err
	}
	return nil
}

func TransferOut(from string, amount int) error {
	oldAmount, _ := BALANCES[from]
	BALANCES[from] = oldAmount - amount
	return nil
}

func TransferIn(to string, amount int) error {
	oldAmount, _ := BALANCES[to]
	BALANCES[to] = oldAmount + amount
	return nil
}

func main() {
	TransferMoney()
	stopped := false
	go func() {
		s := make(chan os.Signal)
		signal.Notify(s, syscall.SIGINT, syscall.SIGTERM)
		<-s
		stopped = true
	}()
	for !stopped {
		fmt.Println(BALANCES["foo"], BALANCES["bar"])
		time.Sleep(time.Second * 3)
	}
}
```

上面的代码比较简单，就是完成一个基本的转帐，将钱从一个帐户转到另一个帐户。

只需要根据以下的步骤，进行简单的改造即可接入分布式事务支持，最终改造后的代码见[test/sagatx_demo.go](./test/sagatx_demo.go)。

### 初始化SagaAgent

在程序入口处初始化SagaAgent，代码如下：

```go
func main(){
    ......
    saga.InitSagaAgent("saga-go-demo", "10.12.142.216:30571", nil)
    ......
}
```

### 构造SagaStart、Compensable方法

由于go语言特性，无法无侵入地进行AOP编程，只能采用Decorator模式代替，因此用Decorator对原来的分布事务入口函数、本地事务函数进行包装，代码如下：

```go
var (
    TransferMoneySagaStartDecorated func() error
	TransferOutCompensableDecorated func(from string, amount int) error
	TransferInCompensableDecorated func(to string, amount int) error
)

func init()  {
	err := saga.DecorateSagaStartMethod(&TransferMoneySagaStartDecorated, TransferMoney, 20)
	if err != nil {
		panic(err)
	}
	err = saga.DecorateCompensableMethod(&TransferOutCompensableDecorated, TransferOut, CancelTransferOut, 5)
	if err != nil {
		panic(err)
	}
	err = saga.DecorateCompensableMethod(&TransferInCompensableDecorated, TransferIn, CancelTransferIn, 5)
	if err != nil {
		panic(err)
	}
    
    ......
}

.......

func CancelTransferOut(from string, amount int) error {
	oldAmount, _ := BALANCES[from]
	BALANCES[from] = oldAmount + amount
	return nil
}

......

func CancelTransferIn(to string, amount int) error {
	oldAmount, _ := BALANCES[to]
	BALANCES[to] = oldAmount - amount
	return nil
}
```

**注意，每个本地事务函数要提供对应的幂等补偿函数**

### 修改对应的包装函数

由于go语言特性，无法无侵入地进行AOP编程，需要手动将原来的分布事务入口函数、本地事务函数修改为对应的包装函数，代码如下：

```go
func TransferMoney() error {
    //err := TransferOut("foo", 100)
	err := TransferOutCompensableDecorated("foo", 100)
	if err != nil {
		return err
	}
    //err = TransferIn("bar", 100)
	err = TransferInCompensableDecorated("bar", 100)
	if err != nil {
		return err
	}
	return nil
}

func main() {
	......
    //TransferMoney()
	TransferMoneySagaStartDecorated()
    ......
}
```

### 传递saga上下文信息

如果一个分布式事务涉及到多个业务服务，则需要在业务服务间传递saga上下文信息，这里涉及两个步骤。

1. 在接收到HTTP请求处使用middleware接收saga上下文信息，请使用合适的[middleware](./middleware)进行处理。

2. 发送HTTP请求到其它业务服务时，使用[sagactx.InjectIntoHttpHeaders](./context/saga_agent_context.go)将当前的saga上下文信息织入HTTP请求头中。


## License
Licensed under an [Apache 2.0 license](LICENSE).