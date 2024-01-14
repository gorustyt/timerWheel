# timerWheel

example:

```go
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
//result
timeWheel run start
2024-01-14 23:14:23.6401488 +0800 CST m=+1.005890901
2024-01-14 23:14:24.6424991 +0800 CST m=+2.008241201
2024-01-14 23:14:25.6376216 +0800 CST m=+3.003363701
2024-01-14 23:14:26.6455683 +0800 CST m=+4.011310401
2024-01-14 23:14:27.6367333 +0800 CST m=+5.002475401
```

go 语言实现 c++游戏框架skynet 的时间轮

游戏中的定时器一般是两种

+ 一种是玩家所在线程调用update ，去处理过期的定时任务，通常由小顶堆实现
+ 另一种是全局定时器，该定时器执行过期任务的时候会另起一个线程，该线会和玩家的线程产生资源竞争，所以执行时候需要加锁。

skynet的实现原理：

+ near 存储最近触发的定时器，有1<<8个链表维护
+ t 存储后续触发的定时器，共4层，每层由64个链表维护
+ 每次tick 会将near 的定时器进行触发，然后将后面4层的定时器往前移动
