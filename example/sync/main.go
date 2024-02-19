package main

import (
	"fmt"
	"github.com/gorustyt/timerWheel"
	"time"
)

func main() {
	//创建一个timer,需要手动调用update 驱动
	t := timerWheel.NewSyncTimeWheel()
	//定时任务
	t.Add(3*time.Second, func(ts time.Time) {
		fmt.Println("add timer=======", ts)
	})
	//调度任务
	t.Schedule(1*time.Second, 1*time.Second, func(ts time.Time) {
		fmt.Println("Schedule=======", ts)
	})
	for { //驱动timer
		t.Update(time.Now(), func(ts time.Time, node *timerWheel.TimerWheelNode) {
			fmt.Printf("manulal exec node id :%v\n", node.Id)
			node.Handle(ts)
		})
		time.Sleep(10 * time.Second)
	}
}
