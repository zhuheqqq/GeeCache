package GeeCache

import (
	"fmt"
	"log"
	"sync"
)

// 从本地获取数据，通过从文件、数据库等方式，具体看自己实现
type Getter interface {
	Get(key string) ([]byte, error) //回调函数
}

type GetterFunc func(key string) ([]byte, error) //接口型函数

// 缓存组，负责管理缓存数据的加载、获取和缓存
type Group struct {
	name      string
	getter    Getter //获取数据，这里会传入具体的方法
	mainCache cache
}

var (
	mu     sync.RWMutex
	groups = make(map[string]*Group)
)

func (f GetterFunc) Get(key string) ([]byte, error) {
	return f(key) //调用f函数并传递key参数 返回结果
}

// 创建一个新的缓存组并添加到全局映射中
func NewGroup(name string, cacheBytes int64, getter Getter) *Group {
	if getter == nil {
		panic("nil Getter")
	}
	mu.Lock()
	defer mu.Unlock()
	g := &Group{
		name:      name,
		getter:    getter,
		mainCache: cache{cacheBytes: cacheBytes},
	}
	groups[name] = g
	return g
}

// 获取指定名称的缓存组
func GetGroup(name string) *Group {
	mu.RLock()
	g := groups[name]
	mu.RUnlock()
	return g
}

// 如果成功获取到值则返回，如果没有则从本地获取
func (g *Group) Get(key string) (ByteView, error) {
	if key == "" {
		return ByteView{}, fmt.Errorf("key is required") //如果key为空，返回一个空的byteview和错误信息
	}

	if v, ok := g.mainCache.get(key); ok {
		log.Println("[GeeCache] hit")
		return v, nil
	}
	return g.load(key)
}

// 从本地获取值
func (g *Group) load(key string) (value ByteView, err error) {
	return g.getLocally(key)
}

func (g *Group) getLocally(key string) (ByteView, error) {
	bytes, err := g.getter.Get(key)
	if err != nil {
		return ByteView{}, err
	}
	value := ByteView{b: cloneBytes(bytes)}
	g.populateCache(key, value)
	return value, nil
}

// 加入缓存
func (g *Group) populateCache(key string, value ByteView) {
	g.mainCache.add(key, value)
}
