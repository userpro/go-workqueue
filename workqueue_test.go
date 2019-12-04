package workqueue

import (
	"errors"
	"math/rand"
	"testing"
	"time"
)

type testT struct {
	id  int
	cnt int
}

func TestWorkQ(t *testing.T) {
	f := func() bool {
		flag := rand.Intn(1)
		t.Log("flag: ", flag)
		if flag == 1 {
			return true
		}
		return false
	}
	randErr := func() error {
		if f() {
			return nil
		}
		return errors.New("random error")
	}

	// 初始化间隔
	SetDuration(time.Second)

	// 开始运行
	Start()

	// 设置该组开始条件 f => func() bool
	// 该函数可以后期设置
	SetGroupStart("testOrder", f)

	Bound := 50
	// 第一次重试
	for i := 1; i <= Bound; i++ {
		Ch <- &Item{
			Type:  Order,
			Group: "testOrder",
			GroupItem: GroupItem{
				Task: &testT{id: i},
				Do: func(args ...interface{}) error {
					if len(args) == 0 {
						return errors.New("no params")
					}
					time.Sleep(100 * time.Millisecond)
					tmp, _ := args[0].(*testT)
					tmp.cnt++
					t.Logf("[Order] id: %d    Retry cnt: %d\n", tmp.id, tmp.cnt)
					if tmp.cnt > 5 {
						return errors.New("timeout")
					}
					return randErr()
				},
				Retry: func(args ...interface{}) bool {
					if len(args) == 0 {
						return false
					}
					tmp, _ := args[0].(*testT)
					tmp.cnt++
					t.Logf("[Order] id: %d    Retry cnt: %d\n", tmp.id, tmp.cnt)
					if tmp.cnt > 5 {
						t.Logf("[Rand] id: %d Retry func repeat overflow", tmp.id)
						return false
					}
					return true
				},
				Callback: func(args ...interface{}) {
					t.Logf("Callback!!!")
				},
			},
		}

		Ch <- &Item{
			Type:  Rand,
			Group: "testRand",
			GroupItem: GroupItem{
				Task: &testT{id: i + Bound},
				Do: func(args ...interface{}) error {
					if len(args) == 0 {
						return errors.New("no params")
					}
					time.Sleep(100 * time.Millisecond)
					tmp, _ := args[0].(*testT)
					tmp.cnt++
					t.Logf("[Rand] id: %d    Retry cnt: %d\n", tmp.id, tmp.cnt)
					if tmp.cnt > 5 {
						return errors.New("[Rand] timeout")
					}
					return randErr()
				},
				Retry: func(args ...interface{}) bool {
					if len(args) == 0 {
						return false
					}
					tmp, _ := args[0].(*testT)
					tmp.cnt++
					t.Logf("[Rand] id: %d    Retry cnt: %d\n", tmp.id, tmp.cnt)
					if tmp.cnt > 5 {
						t.Logf("[Rand] id: %d Retry func repeat overflow", tmp.id)
						return false
					}
					return true
				},
				Callback: func(args ...interface{}) {
					t.Logf("Callback!!!")
				},
			},
		}
	}
	time.Sleep(time.Second * 5)
	// 删除指定分组
	DelGroup("testOrder")

	// 第二次重试
	for i := Bound*2 + 1; i <= Bound*3; i++ {
		Ch <- &Item{
			Type:  Order,
			Group: "testOrder",
			GroupItem: GroupItem{
				Task: &testT{id: i},
				Do: func(args ...interface{}) error {
					time.Sleep(10 * time.Millisecond)
					tmp, _ := args[0].(*testT)
					tmp.cnt++
					t.Logf("[Order] id: %d    Retry cnt: %d\n", tmp.id, tmp.cnt)
					if tmp.cnt > 5 {
						return errors.New("timeout")
					}
					return randErr()
				},
				Retry: func(args ...interface{}) bool {
					tmp, _ := args[0].(*testT)
					tmp.cnt++
					t.Logf("[Order] id: %d    Retry cnt: %d\n", tmp.id, tmp.cnt)
					if tmp.cnt > 5 {
						t.Logf("[Rand] id: %d Retry func repeat overflow", tmp.id)
						return false
					}
					return true
				},
				Callback: func(args ...interface{}) {
					t.Logf("Callback!!!")
				},
			},
		}

		Ch <- &Item{
			Type:  Rand,
			Group: "testRand",
			GroupItem: GroupItem{
				Task: &testT{id: i + Bound},
				Do: func(args ...interface{}) error {
					time.Sleep(200 * time.Millisecond)
					tmp, _ := args[0].(*testT)
					tmp.cnt++
					t.Logf("[Rand] id: %d    Retry cnt: %d\n", tmp.id, tmp.cnt)
					if tmp.cnt > 5 {
						return errors.New("[Rand] timeout")
					}
					return randErr()
				},
				Retry: func(args ...interface{}) bool {
					tmp, _ := args[0].(*testT)
					tmp.cnt++
					t.Logf("[Rand] id: %d    Retry cnt: %d\n", tmp.id, tmp.cnt)
					if tmp.cnt > 5 {
						t.Log("[Rand] Retry func repeat overflow")
						return false
					}
					return true
				},
				Callback: func(args ...interface{}) {
					t.Logf("Callback!!!")
				},
			},
		}
	}

	time.Sleep(time.Second * 10)
}

func sendEmptyTask(f func() bool) {
	Ch <- &Item{
		Type:  Order,
		Group: "groupName",
		GroupItem: GroupItem{
			Do: func(args ...interface{}) error {
				if f() {
					return nil
				}
				return errors.New("[sendEmptyTask] retry task")
			},
			Retry: func(args ...interface{}) bool {
				return true
			},
		},
	}
}
