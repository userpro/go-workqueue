# go-workqueue

### 原理

支持go module

任务分为顺序执行和乱序并发执行两种

任务通过channel入队

每隔一定时间开始执行任务队列

>  顺序执行队列, 按入队顺序执行任务, 只有前一个执行完成才会执行下一个.
>
>  乱序执行队列, 并发执行所有任务.

### Example

~~~go
import (
  "time"
  
  gow "github.com/userpro/go-workqueue"
)

gow.SetDuration(time.Second) // [该函数调用非必须] workqueue的扫描间隔(默认1s)
gow.Start() // 开始扫描
gow.SetGroupStart("testOrder", func() bool {}) // [该函数调用非必须] 第一个参数是分组名, 每个分组名对应一个任务队列; 第二个参数是一个无入参返回bool的函数, 每次先执行该函数, 由其返回值决定是否执行对应的队列

gow.DelGroup(g string) // 删除指定分组

gow.Ch <- &gow.Item{
  Type:  gow.Order, // 任务类型 分 Order(顺序执行), Rand(乱序执行)
  Group: "testOrder", // 分组名
  GroupItem: gow.GroupItem{
    Task: ...,
    Do: func(args ...interface{}) error {
      task, _ := args[0].(...)
      ... args[0] => Task, 返回error会执行retry函数 ...
      if ... {
        return errors.New("...")
      }
      return nil
    },
    Retry: func(args ...interface{}) bool {
      task, _ := args[0].(...)
      ... 重试策略函数 ...
      ... args[0] => Task, 返回true会继续尝试执行Do函数 ...
      if ... {
        return false
      }
      return true
    },
    Callback: func(args ...interface{}) {
      task, _ := args[0].(...)
      done, _ := args[1].(bool) // done true函数最终执行成功 false失败
      ...
    },
  },
}

// 建议封装成类似如下函数的形式使用
func sendEmptyTask(f func() bool) {
	gow.Ch <- &gow.Item{
		Type:  Order,
		Group: "groupName",
		GroupItem: gow.GroupItem{
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
~~~

更详细见 `workqueue_test.go`

