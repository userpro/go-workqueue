package common

import (
	"container/list"
	"sync"
)

// SyncList 双向链表
type SyncList struct {
	l    *list.List
	lock sync.Mutex
}

// New 新建一个双向链表
func (q *SyncList) New() {
	q.l = list.New()
}

// PushBack 从back入
func (q *SyncList) PushBack(data interface{}) {
	q.lock.Lock()
	q.l.PushBack(data)
	q.lock.Unlock()
}

// PushFront 从back入
func (q *SyncList) PushFront(data interface{}) {
	q.lock.Lock()
	q.l.PushFront(data)
	q.lock.Unlock()
}

// PopFront 从front出
func (q *SyncList) PopFront() interface{} {
	q.lock.Lock()
	defer q.lock.Unlock()
	e := q.l.Front()
	if e != nil {
		r := q.l.Remove(e)
		return r
	}
	return nil
}

// PopBack 从front出
func (q *SyncList) PopBack() interface{} {
	q.lock.Lock()
	defer q.lock.Unlock()
	e := q.l.Back()
	if e != nil {
		r := q.l.Remove(e)
		return r
	}
	return nil
}

// PushBackList 合并两个list
func (q *SyncList) PushBackList(ls *SyncList) {
	q.lock.Lock()
	ls.lock.Lock()
	q.l.PushBackList(ls.l)
	ls.lock.Unlock()
	q.lock.Unlock()
}

// PushFrontList 合并两个list
func (q *SyncList) PushFrontList(ls *SyncList) {
	q.lock.Lock()
	ls.lock.Lock()
	q.l.PushFrontList(ls.l)
	ls.lock.Unlock()
	q.lock.Unlock()
}

// Size 双向链表长度
func (q *SyncList) Size() int {
	q.lock.Lock()
	length := q.l.Len()
	q.lock.Unlock()
	return length
}

// Clear 清空双向链表
func (q *SyncList) Clear() {
	q.lock.Lock()
	q.l.Init()
	q.lock.Unlock()
}
