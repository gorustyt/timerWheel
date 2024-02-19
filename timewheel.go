package timerWheel

import (
	"container/list"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

const (
	wheelModeAsync = 1 //异步
	wheelModeSync  = 2 //同步
)
const (
	TIME_NEAR_SHIFT  = 8
	TIME_NEAR        = 1 << TIME_NEAR_SHIFT
	TIME_LEVEL_SHIFT = 6
	TIME_LEVEL       = 1 << TIME_LEVEL_SHIFT
	TIME_NEAR_MASK   = TIME_NEAR - 1
	TIME_LEVEL_MASK  = TIME_LEVEL - 1
)

type TimeWheelHandle func(ts time.Time)

// 时间轮节点
type TimerWheelNode struct {
	Id           int64
	Expire       int64
	Handle       TimeWheelHandle
	Duration     time.Duration
	ExpireAt     time.Time
	FireDuration time.Duration
}

type TimeWheel struct {
	mode         int32
	autoIdInc    int64
	cancels      map[int64]func()
	tick         int64 //不用考虑溢出的问题
	near         [TIME_NEAR]*list.List
	t            [4][TIME_LEVEL]*list.List
	current      int64     //当前跑了多少tick
	currentPoint time.Time //上一次计算时间
	startTime    time.Time //创建时间
	quit         bool      //是否退出
	l            sync.Mutex
}

func NewAsyncTimeWheel() *TimeWheel {
	t := NewSyncTimeWheel()
	t.mode = wheelModeAsync
	go t.run()
	return t
}

func NewSyncTimeWheel() *TimeWheel {
	t := &TimeWheel{
		currentPoint: time.Now(),
		startTime:    time.Now(),
		current:      time.Now().UnixNano() / (10 * 1e6),
		cancels:      map[int64]func(){},
		mode:         wheelModeSync,
	}
	for i := range t.near {
		t.near[i] = list.New()
	}
	for i, v := range t.t {
		for j := range v {
			t.t[i][j] = list.New()
		}
	}
	return t
}
func (t *TimeWheel) lock() {
	if t.mode == wheelModeSync {
		return
	}
	t.l.Lock()
}
func (t *TimeWheel) unlock() {
	if t.mode == wheelModeSync {
		return
	}
	t.l.Unlock()
}
func (t *TimeWheel) execute(ts time.Time, process func(ts time.Time, node *TimerWheelNode)) {
	idx := t.tick & TIME_NEAR_MASK
	l := t.near[idx]
	e := l.Front()

	for e != nil {
		node := e.Value.(*TimerWheelNode)
		e = e.Next()
		t.remove(node.Id)
		if node.FireDuration > 0 { //循环任务
			node.ExpireAt = node.ExpireAt.Add(node.Duration)
			node.Expire = t.calTick(node.ExpireAt)
			t.addNode(node)
		}
		t.unlock()
		if process != nil {
			process(ts, node)
		}
		t.lock()

	}

}
func (t *TimeWheel) Remove(id int64) {
	t.lock()
	t.remove(id)
	t.unlock()
}

func (t *TimeWheel) remove(id int64) {
	if f, ok := t.cancels[id]; ok {
		f()
	}
}

func (t *TimeWheel) Schedule(fireDuration, duration time.Duration, handle TimeWheelHandle) int64 {
	node := &TimerWheelNode{
		Handle:       handle,
		Duration:     duration,
		FireDuration: fireDuration,
		ExpireAt:     time.Now().Add(fireDuration)}
	node.Expire = t.calTick(node.ExpireAt)
	t.lock()
	t.addNode(node)
	t.unlock()
	return node.Id
}

func (t *TimeWheel) calTick(expireAt time.Time) int64 {
	return int64(durationToTick(expireAt.Sub(t.startTime)))
}

func (t *TimeWheel) Add(duration time.Duration, handle TimeWheelHandle) int64 {
	node := &TimerWheelNode{
		Handle:   handle,
		Duration: duration,
		ExpireAt: time.Now().Add(duration)}
	node.Expire = t.calTick(node.ExpireAt)
	t.lock()
	t.addNode(node)
	t.unlock()
	return node.Id
}

func (t *TimeWheel) addNode(node *TimerWheelNode) {
	if node.Id == 0 {
		node.Id = atomic.AddInt64(&t.autoIdInc, 1)
	}
	if node.Expire|TIME_NEAR_MASK == t.tick|TIME_NEAR_MASK {
		l := t.near[node.Expire&TIME_NEAR_MASK]
		e := l.PushBack(node)
		t.cancels[node.Id] = func() {
			delete(t.cancels, node.Id)
			l.Remove(e)
		}
	} else {
		mask := int64(TIME_NEAR << TIME_LEVEL_SHIFT)
		var i int
		for i = 0; i < 3; i++ {
			if node.Expire|(mask-1) == t.tick|(mask-1) {
				break
			}
			mask <<= TIME_LEVEL_SHIFT
		}
		l := t.t[i][(node.Expire>>(TIME_NEAR_SHIFT+i*TIME_LEVEL_SHIFT))&TIME_LEVEL_MASK]
		e := l.PushBack(node)
		t.cancels[node.Id] = func() {
			delete(t.cancels, node.Id)
			l.Remove(e)
		}
	}
}

func (t *TimeWheel) timerShift() {
	mask := TIME_NEAR
	t.tick++
	ct := t.tick
	if ct == 0 {
		t.move(3, 0)
	} else {
		var i int
		it := ct >> TIME_NEAR_SHIFT
		for int(ct)&(mask-1) == 0 {
			idx := int(it) & TIME_LEVEL_MASK
			if idx != 0 {
				t.move(i, idx)
				break
			}
			it >>= TIME_LEVEL_SHIFT
			mask <<= TIME_LEVEL_SHIFT
			i++
		}
	}
}

func (t *TimeWheel) move(level, idx int) {
	l := t.t[level][idx]
	e := l.Front()
	for e != nil {
		node := e.Value.(*TimerWheelNode)
		e = e.Next()
		t.remove(node.Id)
		t.addNode(node)

	}
}

func (t *TimeWheel) Update(now time.Time, process func(ts time.Time, node *TimerWheelNode)) {
	if now.Before(t.currentPoint) {
		fmt.Println("find error currentPoint")
		t.currentPoint = now
	} else {
		diff := int(durationToTick(now.Sub(t.currentPoint)))
		if diff == 0 {
			return
		}
		t.currentPoint = t.currentPoint.Add(time.Duration(diff) * 10 * time.Millisecond)
		for i := 0; i < diff; i++ {
			t.update(t.currentPoint, process)
		}
		t.current += int64(diff)
	}
}

func (t *TimeWheel) update(ts time.Time, process func(ts time.Time, node *TimerWheelNode)) {
	t.lock()
	t.execute(ts, process)
	t.timerShift()
	t.execute(ts, process)
	t.unlock()
}

// 这里改为手动运行，方便自己使用驱动
func (t *TimeWheel) run() {
	fmt.Println("timeWheel run start")
	ticker := time.NewTicker(1 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			t.Update(time.Now(), func(ts time.Time, node *TimerWheelNode) {
				go node.Handle(ts)
			})
		}
		if t.quit {
			break
		}
	}
}

func (t *TimeWheel) Close() {
	fmt.Println("timeWheel close")
	t.quit = true
}

func durationToTick(d time.Duration) float64 {
	sec := d / time.Second
	nsec := d % time.Second
	return float64(sec)*100 + float64(nsec)/1e7
}
