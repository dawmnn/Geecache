package main

import (
	"flag"
	"fmt"
	"geecache"
	"log"
	"net/http"
)

//
//var db = map[string]string{
//	"Tom":  "630",
//	"Jack": "589",
//	"Sam":  "567",
//}
//
//func main() {
//	geecache.NewGroup("scores", 2<<10, geecache.GetterFunc(
//		//二进制值实际上等于 2 * 2^10，即 2 * 1024 = 2048
//		func(key string) ([]byte, error) {
//			log.Println("[SlowDB] search key", key)
//			if v, ok := db[key]; ok { //db 是在包级别定义的全局变量，
//				// 所以它对整个 main 包中的所有函数都是可见的。
//				//匿名函数可以访问它所定义作用域中的所有变量，
//				//而它的定义作用域是整个 main 包，所以它可以访问到 db
//				return []byte(v), nil
//			}
//			return nil, fmt.Errorf("%s not exist", key)
//		}))
//
//	addr := "localhost:9999"
//	peers := geecache.NewHTTPPool(addr)
//	log.Println("geecache is running at", addr)
//	log.Fatal(http.ListenAndServe(addr, peers))
//	// 通常是实现了 http.Handler 接口的对象，处理 HTTP 请求的逻辑
//}

var db = map[string]string{
	"Tom":  "630",
	"Jack": "589",
	"Sam":  "567",
}

func createGroup() *geecache.Group {
	return geecache.NewGroup("scores", 2<<10, geecache.GetterFunc(
		func(key string) ([]byte, error) {
			log.Println("[SlowDB] search key", key)
			if v, ok := db[key]; ok {
				return []byte(v), nil
			}
			return nil, fmt.Errorf("%s not exist", key)
		}))
}

func startCacheServer(addr string, addrs []string, gee *geecache.Group) {
	peers := geecache.NewHTTPPool(addr)
	peers.Set(addrs...)
	gee.RegisterPeers(peers)
	log.Println("geecache is running at", addr)
	log.Fatal(http.ListenAndServe(addr[7:], peers))
}

func startAPIServer(apiAddr string, gee *geecache.Group) {
	http.Handle("/api", http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			key := r.URL.Query().Get("key")
			//r.URL.Query() 获取请求 URL 中的查询参数部分。
			//查询参数通常是 URL 中 ? 后面的键值对，例如 http://example.com/?key=value，
			//其中 key 是查询参数的键，value 是值。
			//.Get("key") 是从查询参数中获取名为 "key" 的参数的值。
			//如果 URL 中包含 key=value 这样的查询参数，r.URL.Query().Get("key") 就会返回 value。
			//如果 URL 中没有 key 这个查询参数，Get("key") 会返回一个空字符串 ""。
			view, err := gee.Get(key)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Write(view.ByteSlice())

		}))
	log.Println("fontend server is running at", apiAddr)
	log.Fatal(http.ListenAndServe(apiAddr[7:], nil))

}

func main() {
	var port int
	var api bool
	flag.IntVar(&port, "port", 8001, "Geecache server port")
	//定义了一个整型变量 port，用于存储服务器的端口号。
	flag.BoolVar(&api, "api", false, "Start a api server?")
	//定义了一个布尔型变量 api，用于决定是否启动 API 服务器
	flag.Parse()

	apiAddr := "http://localhost:9999"
	addrMap := map[int]string{
		8001: "http://localhost:8001",
		8002: "http://localhost:8002",
		8003: "http://localhost:8003",
	}

	var addrs []string
	for _, v := range addrMap {
		addrs = append(addrs, v)
	}

	gee := createGroup()
	if api {
		go startAPIServer(apiAddr, gee)
	}
	startCacheServer(addrMap[port], addrs, gee)
}

//startCacheServer() 用来启动缓存服务器：创建 HTTPPool，添加节点信息，
//注册到 gee 中，启动 HTTP 服务（共3个端口，8001/8002/8003），用户不感知。
//startAPIServer() 用来启动一个 API 服务（端口 9999），与用户进行交互，用户感知。
