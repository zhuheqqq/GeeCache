/*
lru算法是最近最少使用，如果数据最近被访问过，那么将来被访问的概率会更高
可以维护一个队列，如果某条记录被访问了，则移动到队尾，队首是最近最少访问的数据，淘汰该记录即可
*/

package lru

import (
	"container/list"
)

// it is not safe for concurrent access
type Cache struct {
	maxBytes int64 //允许使用的最大内存
	nbytes   int64 //当前已使用的内存
	ll       *list.List
	cache    map[string]*list.Element //键是字符串，值是双向链表对应节点的指针

	onEvicted func(key string, value Value) //某条记录被移除时的回调函数
}

type entry struct {
	key   string
	value Value
}

// 用于返回值所占的内存大小
type Value interface {
	Len() int
}

func New(maxBytes int64, onEvicted func(string, Value)) *Cache {
	return &Cache{
		maxBytes:  maxBytes,
		ll:        list.New(),
		cache:     make(map[string]*list.Element),
		onEvicted: onEvicted,
	}
}

//查找：从字典中找到对应的双向链表的节点，然后移至队尾
func (c *Cache) Get(key string)(value Value,ok bool){
	if ele,ok : =c.cache[key];ok{
		c.ll.MoveToFront(ele)		//将链表中的节点ele移动到队尾
		//（双向链表作为队列，队首队尾是相对的，在这里约定 front 为队尾）
		kv := ele.Value.(*entry)
		return kv.value,true
	}
	return
}

//删除：缓存淘汰
func (c *Cache) RemoveOldest() {
	ele :=c.ll.Back()		//取到队首节点，从链表删除
	if ele !=nil {
		c.ll.Remove(ele)
		kv := ele.Value.(*entry)
		delete(c.cache,kv.key)		//从字典中删除映射
		c.nbytes -= int64(len(kv.key)) + int64(kv.value.Len())		//更新当前所用内存
		if c.onEvicted != nil {
			c.onEvicted(kv.key,kv.value)
		}
	}
}
