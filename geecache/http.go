package geecache

import (
	"fmt"
	"log"
	"net/http"
	"strings"
)

// 提供被其他节点访问的能力(基于http)

//HTTPPool 只有 2 个参数，一个是 self，用来记录自己的地址，包括主机名/IP 和端口。
//另一个是 basePath，作为节点间通讯地址的前缀，默认是 /_geecache/

const defaultBasePath = "/_geecache/"

// HTTPPool implements PeerPicker for a pool of HTTP peers.
type HTTPPool struct {
	// this peer's base URL, e.g. "https://example.net:8000"
	self     string
	basePath string
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
	//路径的格式约定是 /basePath/groupName/key，所以在这里首先通过 strings.SplitN 方法将 URL 路径按 / 分割，
	//并确保分割后有两个部分：groupName 和 key。
	//r.URL.Path[len(p.basePath):] 这一步是去掉请求路径中的 basePath 部分，剩下的部分才是 groupName 和 key。
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
