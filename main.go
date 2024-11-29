package main

import (
	"fmt"
	"geecache"
	"log"
	"net/http"
)

var db = map[string]string{
	"Tom":  "630",
	"Jack": "589",
	"Sam":  "567",
}

func main() {
	geecache.NewGroup("scores", 2<<10, geecache.GetterFunc(
		//二进制值实际上等于 2 * 2^10，即 2 * 1024 = 2048
		func(key string) ([]byte, error) {
			log.Println("[SlowDB] search key", key)
			if v, ok := db[key]; ok { //db 是在包级别定义的全局变量，
				// 所以它对整个 main 包中的所有函数都是可见的。
				//匿名函数可以访问它所定义作用域中的所有变量，
				//而它的定义作用域是整个 main 包，所以它可以访问到 db
				return []byte(v), nil
			}
			return nil, fmt.Errorf("%s not exist", key)
		}))

	addr := "localhost:9999"
	peers := geecache.NewHTTPPool(addr)
	log.Println("geecache is running at", addr)
	log.Fatal(http.ListenAndServe(addr, peers))
	// 通常是实现了 http.Handler 接口的对象，处理 HTTP 请求的逻辑
}
