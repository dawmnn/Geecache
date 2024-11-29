package lru

//缓存淘汰策略
import "container/list"

// Value use Len to count how many bytes it takes
type Value interface {
	Len() int
}

type Cache struct {
	maxBytes  int64                         // 缓存的最大字节数
	nbytes    int64                         // 当前已使用的字节数
	ll        *list.List                    // 双向链表，维护缓存项的使用顺序
	cache     map[string]*list.Element      // 字典，键值对映射
	OnEvicted func(key string, value Value) // 删除元素时的回调函数
}

type entry struct {
	key   string // 缓存项的键
	value Value  // 缓存项的值
}

func New(maxBytes int64, onEvicted func(string, Value)) *Cache {
	return &Cache{
		maxBytes:  maxBytes,
		ll:        list.New(),
		cache:     make(map[string]*list.Element),
		OnEvicted: onEvicted,
	}
}

func (c *Cache) Get(key string) (value Value, ok bool) {
	if ele, ok := c.cache[key]; ok {
		// 将访问到的节点移到队尾，表示这个节点最近被使用
		c.ll.MoveToFront(ele) //把ele放到c.ll后面//即将链表中的节点 ele 移动到队首

		kv := ele.Value.(*entry) // 获取缓存值
		return kv.value, true
	}
	return
}

//移除最近最少访问的节点（队首）

// RemoveOldest removes the oldest item
func (c *Cache) RemoveOldest() {
	ele := c.ll.Back()
	if ele != nil {
		c.ll.Remove(ele)
		kv := ele.Value.(*entry)
		delete(c.cache, kv.key)                                //删除这个map[key}映射
		c.nbytes -= int64(len(kv.key)) + int64(kv.value.Len()) //key删了，内容没删
		if c.OnEvicted != nil {
			c.OnEvicted(kv.key, kv.value)
		}
	}
}

// Add adds a value to the cache.
func (c *Cache) Add(key string, value Value) {
	if ele, ok := c.cache[key]; ok {
		c.ll.MoveToFront(ele) //往队首放
		kv := ele.Value.(*entry)
		c.nbytes += int64(value.Len()) - int64(kv.value.Len())
		kv.value = value
	} else {
		ele := c.ll.PushFront(&entry{key, value}) //放到队首,并且e.list = l
		c.cache[key] = ele
		c.nbytes += int64(len(key)) + int64(value.Len())
	}
	for c.maxBytes != 0 && c.maxBytes < c.nbytes {
		c.RemoveOldest()
	}
}

// Len the number of cache entries
func (c *Cache) Len() int {
	return c.ll.Len()
}
