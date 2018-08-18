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
