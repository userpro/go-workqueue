package workqueue

import (
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/userpro/go-workqueue/common"
)

// Ch workq任务入口 传入&workq.Item{}
var Ch chan interface{}
var once sync.Once

// workq 任务类型
const (
	Order int = 0 // 顺序
	Rand  int = 1 // 乱序
)

// Item work任务
type Item struct {
	Type  int
	Group string
	GroupItem
}

// GroupItem group对应的队列的元素
type GroupItem struct {
	Task     interface{}
	Do       func(...interface{}) error // func(task)
	Retry    func(...interface{}) bool  // func(task, err)  第二个参数为Do函数返回到error 返回true表示继续重试
	Callback func(...interface{})       // func(task, bool) 第二个参数表示该任务是否成功执行
}

type group struct {
	start func() bool
	q     *common.SyncList // list<GroupItem> => queue
	l     common.Mutex
}

type workQ struct {
	Duration time.Duration
	order    map[string]*group
	rand     map[string]*group
	ol, rl   common.Mutex
}

func (g *group) init() {
	if g.q == nil {
		g.q = &common.SyncList{}
		g.q.New()
	}
}

func (g *group) setStart(s func() bool) {
	g.l.Lock()
	g.start = s
	g.l.Unlock()
}

func (g *group) push(data interface{}) {
	g.q.PushBack(data)
}

// 顺序执行
func (g *group) orderDo() {
	if g.q == nil {
		return
	}

	if !g.l.TryLock() {
		return
	}
	defer g.l.Unlock()

	for g.q.Size() > 0 {
		t := g.q.PopFront()
		if t == nil {
			return
		}
		e, _ := t.(*GroupItem)
		if e.Do == nil {
			continue
		}
		// 执行失败
		if err := e.Do(e.Task); err != nil {
			// log.Error(err)
			if e.Retry != nil && e.Retry(e.Task, err) {
				g.q.PushFront(e)
				continue
			}

			if e.Callback != nil {
				e.Callback(e.Task, false)
			}
			return
		}

		if e.Callback != nil {
			e.Callback(e.Task, true)
		}
	}
}

// 乱序执行
func (g *group) randDo() {
	// 失败需要继续重试队列
	failq := &common.SyncList{}
	failq.New()

	g.l.Lock()
	// swap list
	runq := g.q
	g.q = &common.SyncList{}
	g.q.New()
	g.l.Unlock()

	var wg sync.WaitGroup
	wg.Add(runq.Size())

	for runq.Size() > 0 {
		t := runq.PopFront()
		if t == nil {
			return
		}
		e, _ := t.(*GroupItem)
		go func(failq *common.SyncList, e *GroupItem) {
			defer wg.Done()
			if e.Do == nil {
				return
			}

			// 执行失败
			if err := e.Do(e.Task); err != nil {
				// log.Error(err)
				if e.Retry != nil && e.Retry(e.Task, err) {
					failq.PushBack(e)
					return
				}
				if e.Callback != nil {
					e.Callback(e.Task, false)
				}
				return
			}

			if e.Callback != nil {
				e.Callback(e.Task, true)
			}
		}(failq, e)
	}

	wg.Wait()
	// 将失败队列合并回主队列
	g.q.PushBackList(failq)
}

func (r *workQ) setGroupStart(g string, f func() bool) {
	r.ol.Lock()
	v, ok := r.order[g]
	if !ok {
		r.order[g] = &group{}
		v, _ = r.order[g]
		v.init()
	}
	v.setStart(f)
	r.ol.Unlock()

	r.rl.Lock()
	v, ok = r.rand[g]
	if !ok {
		r.rand[g] = &group{}
		v, _ = r.rand[g]
		v.init()
	}
	v.setStart(f)
	r.rl.Unlock()
}

func (r *workQ) delGroup(group string) {
	r.ol.Lock()
	if _, ok := r.order[group]; ok {
		delete(r.order, group)
	}
	r.ol.Unlock()

	r.rl.Lock()
	if _, ok := r.rand[group]; ok {
		delete(r.rand, group)
	}
	r.rl.Unlock()
}

func (r *workQ) init() {
	r.order = make(map[string]*group)
	r.rand = make(map[string]*group)
}

func (r *workQ) push(t *Item) {
	var g *group
	switch t.Type {
	case Order:
		tg, ok := r.order[t.Group]
		if !ok {
			r.order[t.Group] = &group{}
			tg = r.order[t.Group]
			tg.init()
		}
		g = tg

	case Rand:
		tg, ok := r.rand[t.Group]
		if !ok {
			r.rand[t.Group] = &group{}
			tg = r.rand[t.Group]
			tg.init()
		}
		g = tg

	default:
		log.Errorf("Unknown item type: %v.", t)
		return
	}

	g.push(&GroupItem{Task: t.Task, Do: t.Do, Retry: t.Retry, Callback: t.Callback})
}

func (r *workQ) orderDo() {
	r.ol.Lock()
	for _, v := range r.order {
		if v.q == nil || v.q.Size() <= 0 {
			continue
		}
		if v.start == nil || v.start() {
			go v.orderDo()
		}
	}
	r.ol.Unlock()
}

func (r *workQ) randDo() {
	r.rl.Lock()
	for _, v := range r.rand {
		if v.q == nil || v.q.Size() <= 0 {
			continue
		}
		if v.start == nil || v.start() {
			go v.randDo()
		}
	}
	r.rl.Unlock()
}

/* --- 正文开始 --- */

// 任务队列
var workq workQ

func init() {
	Ch = make(chan interface{}, 10)
	workq = workQ{}
	workq.init()
	workq.Duration = time.Second // 默认间隔
}

// SetGroupStart 设置每次开始重试的前置条件
func SetGroupStart(g string, f func() bool) {
	workq.setGroupStart(g, f)
}

// DelGroup 删除指定分组
func DelGroup(g string) {
	workq.delGroup(g)
}

// SetDuration 设置定时重试间隔
func SetDuration(d time.Duration) {
	workq.Duration = d
}

// Start 开启运行
func Start() {
	once.Do(
		func() {
			orderT := time.NewTimer(workq.Duration)
			randT := time.NewTimer(workq.Duration)

			go func() {
				for {
					t := <-Ch
					it, ok := t.(*Item)
					if ok {
						workq.push(it)
					}
				}
			}()

			go func(t *time.Timer) {
				for {
					t.Reset(workq.Duration)
					<-t.C
					workq.orderDo()
				}
			}(orderT)

			go func(t *time.Timer) {
				for {
					t.Reset(workq.Duration)
					<-t.C
					go workq.randDo()
				}
			}(randT)
		})
}
