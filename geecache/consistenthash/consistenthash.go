package consistenthash

//一致性哈希
import (
	"hash/crc32"
	"sort"
	"strconv"
)

// Hash maps bytes to uint32
type Hash func(data []byte) uint32 //用来将任意数据（通常是一个字节数组）
// 映射到一个 uint32 类型的哈希值上
//Hash 可以是用户定义的任何哈希函数，也可以使用默认的 crc32.ChecksumIEEE

// Map constains all hashed keys
type Map struct {
	hash     Hash  //哈希转换函数
	replicas int   //虚拟节点的数量
	keys     []int //哈希环// Sorted//这个切片保存的是所有虚拟节点的哈希值，
	// 并且是升序排列的。哈希环的节点就是这些虚拟节点的哈希值。
	hashMap map[int]string //一个映射关系，保存了虚拟节点的哈希值和对应的真实节点名称。
	// 通过这个映射，哈希环上的哈希值可以对应到一个真实节点
	//键是虚拟节点的哈希值，值是真实节点的名称。
}

// New creates a Map instance
func New(replicas int, fn Hash) *Map {
	m := &Map{
		replicas: replicas,
		hash:     fn,
		hashMap:  make(map[int]string),
	}
	if m.hash == nil {
		m.hash = crc32.ChecksumIEEE
	}
	return m
}

// Add adds some keys to the hash.
func (m *Map) Add(keys ...string) {
	for _, key := range keys {
		for i := 0; i < m.replicas; i++ {
			hash := int(m.hash([]byte(strconv.Itoa(i) + key))) //虚拟节点的名称是：
			// strconv.Itoa(i) + key， 即通过添加编号的方式区分不同虚拟节点
			m.keys = append(m.keys, hash)
			m.hashMap[hash] = key
		}
	}
	sort.Ints(m.keys)
}

// Get gets the closest item in the hash to the provided key.
func (m *Map) Get(key string) string {
	if len(m.keys) == 0 {
		return ""
	}

	hash := int(m.hash([]byte(key)))
	// Binary search for appropriate replica.
	idx := sort.Search(len(m.keys), func(i int) bool {
		return m.keys[i] >= hash
	})
	//sort.Search 是 Go 标准库中的一个函数，用于在一个已排序的切片中执行二分查找
	//虚拟节点的哈希值是排序好的。
	//通过二分查找可以在对数时间内快速找到第一个哈希值大于等于 hash 的位置。
	//这使得一致性哈希的查找过程效率很高，
	//即使虚拟节点数量很多，查找的时间复杂度仍然是 O(log n)。

	return m.hashMap[m.keys[idx%len(m.keys)]]
	//如果 idx == len(m.keys)，说明应选择 m.keys[0]，
	//因为 m.keys 是一个环状结构，所以用取余数的方式来处理这种情况。
}
