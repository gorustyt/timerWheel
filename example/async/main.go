package main

import (
	"fmt"
	tm "github.com/gorustyt/timerWheel"
	"time"
)

func main() {
	//创建一个异步timer，单独的协程驱动，调用update
	t := tm.NewAsyncTimeWheel()
	//定时任务
	t.Add(3*time.Second, func(ts time.Time) {
		fmt.Println("add timer=======", ts)
	})
	//调度任务
	t.Schedule(1*time.Second, 1*time.Second, func(ts time.Time) {
		fmt.Println("Schedule=======", ts)
	})
	time.Sleep(1000 * time.Second)
}
