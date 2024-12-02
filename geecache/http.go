package geecache

import (
	"fmt"
	"geecache/consistenthash"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
)

// 提供被其他节点访问的能力(基于http)

//HTTPPool 只有 2 个参数，一个是 self，用来记录自己的地址，包括主机名/IP 和端口。
//另一个是 basePath，作为节点间通讯地址的前缀，默认是 /_geecache/

const (
	defaultBasePath = "/_geecache/"
	defaultReplicas = 50
)

type HTTPPool struct {
	self        string //当前节点的 URL
	basePath    string //是缓存系统的路径（默认是 /_geecache/）
	mu          sync.Mutex
	peers       *consistenthash.Map    //使用一致性哈希（consistenthash.Map）来根据 key 选择节点
	httpGetters map[string]*httpGetter // 存储 HTTP 客户端
	// 是一个映射，将节点的地址与对应的 httpGetter 绑定。
}

// NewHTTPPool initializes an HTTP pool of peers.
func NewHTTPPool(self string) *HTTPPool {
	return &HTTPPool{
		self:     self,
		basePath: defaultBasePath,
	}
}

// Log info with server name
func (p *HTTPPool) Log(format string, v ...interface{}) {
	log.Printf("[Server %s] %s", p.self, fmt.Sprintf(format, v...))
}

// ServeHTTP handle all http requests
func (p *HTTPPool) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// 检查请求的 URL 路径是否以 basePath 为前缀
	if !strings.HasPrefix(r.URL.Path, p.basePath) {
		panic("HTTPPool serving unexpected path: " + r.URL.Path)
	}
	// 记录日志，打印请求的 HTTP 方法和 URL 路径
	p.Log("%s %s", r.Method, r.URL.Path)
	// /<basepath>/<groupname>/<key> required
	// 解析 URL 路径，路径格式为 /<basepath>/<groupname>/<key>
	parts := strings.SplitN(r.URL.Path[len(p.basePath):], "/", 2)
	if len(parts) != 2 {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	//路径的格式约定是 /basePath/groupName/key，
	//所以在这里首先通过 strings.SplitN 方法将 URL 路径按 / 分割，
	//并确保分割后有两个部分：groupName 和 key。
	//r.URL.Path[len(p.basePath):] 这一步是去掉请求路径中的 basePath 部分，
	//剩下的部分才是 groupName 和 key。
	groupName := parts[0]
	key := parts[1]

	//肯定要有个group
	group := GetGroup(groupName)
	if group == nil {
		http.Error(w, "no such group: "+groupName, http.StatusNotFound)
		return
	}

	view, err := group.Get(key)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/octet-stream") //表示返回的是二进制数据流。
	w.Write(view.ByteSlice())
}

//ServeHTTP 的实现逻辑是比较简单的，首先判断访问路径的前缀是否是 basePath，不是返回错误。
//我们约定访问路径格式为 /<basepath>/<groupname>/<key>，通过 groupname 得到 group 实例，
//再使用 group.Get(key) 获取缓存数据。
//最终使用 w.Write() 将缓存值作为 httpResponse 的 body 返回。

type httpGetter struct {
	baseURL string //baseURL 表示将要访问的远程节点的地址，例如 http://example.com/_geecache/
}

//具体的 HTTP 客户端类 httpGetter，实现 PeerGetter 接口

func (h *httpGetter) Get(group string, key string) ([]byte, error) {
	u := fmt.Sprintf(
		"%v%v/%v",
		h.baseURL,
		url.QueryEscape(group),
		url.QueryEscape(key), //"my group" 会被编码为 "my%20group"，
		// "special&key" 会被编码为 "special%26key"，
		// 确保它们在 URL 查询参数中不会引发语法错误。
	) //将 group 和 key 字符串进行 URL 编码，使它们能够安全地嵌入 URL 查询字符串中
	res, err := http.Get(u) //发送 HTTP 请求，获取缓存数据，并将返回的结果转换为字节数组 ([]byte)。
	//如果 HTTP 响应状态不是 200（OK），则返回错误
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned: %v", res.Status)
	}

	bytes, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response body: %v", err)
	}

	return bytes, nil
}

var _ PeerGetter = (*httpGetter)(nil)

// Set updates the pool's list of peers.
func (p *HTTPPool) Set(peers ...string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.peers = consistenthash.New(defaultReplicas, nil)
	p.peers.Add(peers...)
	p.httpGetters = make(map[string]*httpGetter, len(peers))
	for _, peer := range peers {
		p.httpGetters[peer] = &httpGetter{baseURL: peer + p.basePath}
	}
}

// PickPeer picks a peer according to key
func (p *HTTPPool) PickPeer(key string) (PeerGetter, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if peer := p.peers.Get(key); peer != "" && peer != p.self {
		p.Log("Pick peer %s", peer)
		return p.httpGetters[peer], true
	}
	return nil, false
}

var _ PeerPicker = (*HTTPPool)(nil)
