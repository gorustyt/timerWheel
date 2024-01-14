package main

import (
	"fmt"
	tm "github.com/gorustyt/timerWheel"
	"time"
)

func main() {
	t := tm.NewAsyncTimeWheel()
	t.Schedule(1*time.Second, 1*time.Second, func(ts time.Time) {
		fmt.Println(time.Now())
	})
	time.Sleep(1000 * time.Second)
}
