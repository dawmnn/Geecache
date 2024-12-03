package singleflight

import "sync"

// 防止缓存击穿

// 假设你在一个分布式系统中，多个节点（比如 8003）向一个服务节点（比如 8001）发起了多个相同的请求
// （例如：请求 ?key=Tom）。如果没有任何控制，这样的请求会导致：
//
// 缓存击穿：如果缓存中没有数据，多个请求可能同时去访问数据库，造成数据库的压力急剧增大。
// 缓存穿透：即使缓存没有数据，多个请求也会同时发起，造成多次不必要的数据库查询。
// 资源浪费：由于请求的重复性，导致相同的操作被执行多次，浪费计算和带宽资源。
// 为了避免这些问题，可以使用 singleflight 来确保对同一 key 的并发请求只会真正发起一次请求，
// 其他的请求会等待这次请求完成，并共享结果。
type call struct {
	wg  sync.WaitGroup //用于等待并发协程完成的同步机制。
	val interface{}    //请求成功返回的结果。
	err error          //请求失败时的错误信息。
}

type Group struct {
	mu sync.Mutex       // protects m//用来保护 m 字典，防止并发读写。
	m  map[string]*call //字典，键是请求的 key，值是当前正在处理该 key 的 call 结构体。
}

//确保对同一个 key 的并发请求只会触发一次后台操作

//如果对某个 key 的请求已经存在，等待它完成并返回它的结果；
//如果没有相同的请求正在进行，就启动新的请求并保存其结果

func (g *Group) Do(key string, fn func() (interface{}, error)) (interface{}, error) {
	g.mu.Lock() // 加锁，防止并发修改 m
	if g.m == nil {
		g.m = make(map[string]*call) // 延迟初始化 m
	}
	if c, ok := g.m[key]; ok { //通过检查 g.m[key] 是否存在，
		// 来判断是否已经有其他 goroutine 正在处理这个 key 对应的请求
		g.mu.Unlock() // 已经有请求在进行，释放锁
		c.wg.Wait()   //// 等待正在进行的请求完成
		return c.val, c.err
	}
	//如果 g.m[key] 不存在（即没有请求在进行），
	//创建一个新的 call 对象 c，并将其添加到 g.m 中
	c := new(call)
	c.wg.Add(1) //使用 c.wg.Add(1) 来表明有一个新的请求正在进行
	g.m[key] = c
	g.mu.Unlock() //然后释放锁，允许其他 goroutine 在此时进行操作。

	c.val, c.err = fn()
	c.wg.Done() //调用 c.wg.Done()，表示请求完成，WaitGroup 的计数器减 1

	g.mu.Lock()
	delete(g.m, key)
	g.mu.Unlock()

	return c.val, c.err
}

//Add(int)：设置等待的 goroutine 数量。每启动一个 goroutine，
//就调用 Add(1)，每当一个 goroutine 完成时，调用 Done()，
//最后通过 Wait() 来等待所有 goroutine 完成。
//Done()：表示一个 goroutine 执行完成，减少 WaitGroup 中的计数。
//Wait()：阻塞当前 goroutine，直到 WaitGroup 中的计数归零。
