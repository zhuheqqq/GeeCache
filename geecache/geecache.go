package geecache

import (
	"GeeCache/singleflight"
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
	getter    Getter     //获取数据，这里会传入具体的方法
	mainCache cache      //指定缓存大小
	peers     PeerPicker //用于选择远程节点
	loader    *singleflight.Group
}

var (
	mu     sync.RWMutex
	groups = make(map[string]*Group)
)

func (g *Group) RegisterPeers(peers PeerPicker) {
	if g.peers != nil {
		panic("RegisterPeerPicker called more than once")
	}
	g.peers = peers
}

// 如果有远程节点则从远程节点获取值，否则从本地获取值
func (g *Group) load(key string) (value ByteView, err error) {
	viewi, err := g.loader.Do(key, func() (interface{}, error) {
		if g.peers != nil {
			if peer, ok := g.peers.PickPeer(key); ok {
				if value, err = g.getFromPeer(peer, key); err == nil {
					return value, nil
				}
				log.Println("[GeeCache] Failed to get from peer", err)
			}
		}
		return g.getLocally(key)
	})

	if err == nil {
		return viewi.(ByteView), nil
	}
	return
}

func (g *Group) getFromPeer(peer PeerGetter, key string) (ByteView, error) {
	bytes, err := peer.Get(g.name, key)
	if err != nil {
		return ByteView{}, err
	}
	return ByteView{b: bytes}, nil
}

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
		loader:    &singleflight.Group{},
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
