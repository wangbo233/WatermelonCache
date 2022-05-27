package main

import (
	"GoCache"
	"flag"
	"fmt"
	"log"
	"net/http"
)

var db = map[string]string{
	"Tom":  "630",
	"Jack": "589",
	"Sam":  "567",
}

func createGroup() *GoCache.Group {
	return GoCache.NewGroup("scores", 2<<10, GoCache.GetterFunc(
		func(key string) ([]byte, error) {
			log.Println("[SlowDB] search key", key)
			if v, ok := db[key]; ok {
				return []byte(v), nil
			}
			return nil, fmt.Errorf("%s not exist", key)
		}))
}

// 启动缓存服务器：创建 HTTPPool，添加节点信息，注册到 g 中，
// 启动 HTTP 服务（共3个端口，8001/8002/8003），用户不感知
func startCacheServer(addr string, addrs []string, g *GoCache.Group) {
	peers := GoCache.NewHTTPPool(addr)
	peers.Set(addrs...)
	g.RegisterPeers(peers)
	log.Println("gocache is running at", addr)
	log.Fatal(http.ListenAndServe(addr[7:], peers))
}

// 用来启动一个 API 服务（端口 9999），与用户进行交互，用户感知
func startAPIServer(apiAddr string, g *GoCache.Group) {
	http.Handle("/api", http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			key := r.URL.Query().Get("key")
			view, err := g.Get(key)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Write(view.ByteSlice())

		}))
	log.Println("frontend server is running at", apiAddr)
	log.Fatal(http.ListenAndServe(apiAddr[7:], nil))
}

// main() 函数需要命令行传入 port 和 api 2 个参数，用来在指定端口启动 HTTP 服务
func main() {
	var port int
	var api bool
	flag.IntVar(&port, "port", 8001, "GoCache server port")
	flag.BoolVar(&api, "api", false, "Start a api server?")
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
	g := createGroup()
	if api {
		go startAPIServer(apiAddr, g)
	}
	startCacheServer(addrMap[port], []string(addrs), g)
}
