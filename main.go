package main

import (
	"fmt"
	"github.com/fiorix/go-redis/redis"
	cache "github.com/victorspringer/http-cache"
	"github.com/victorspringer/http-cache/adapter/memory"
	"golang.org/x/time/rate"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

var PORT = os.Getenv("PORT")

var REDIS_IP = os.Getenv("REDIS_IP")
var REDIS_PORT = os.Getenv("REDIS_PORT")

var increment = 1;

func initRedis() {
	client := RedisClient()

	client.Set("keys", "")
}

func checkEnv() {
	if PORT == "" {
		PORT = "7137"
	}

	if REDIS_PORT == "" {
		REDIS_PORT = "6379"
	}

	if REDIS_IP == "" {
		REDIS_IP = "localhost"
	}
}

func RedisClient() *redis.Client {
	return redis.New(REDIS_IP + ":" + REDIS_PORT)
}

func sendRedis(w http.ResponseWriter, r *http.Request) {
	lat := r.Header.Get("lat")
	lon := r.Header.Get("lon")

	client := RedisClient()

	if err := client.Set("key" + string(increment), string(lat) + "#" + string(lon)); err != nil {
		panic(err)
	}

	keys, err := client.Get("keys")
	if err != nil {
		panic(err)
	}

	if err := client.Set("keys", keys +  "," + "key" + string(increment) ); err != nil {
		panic(err)
	}

	increment++

	if _, err := w.Write([]byte("Done")); err != nil {
		panic(err)
	}
}

func getRedis(w http.ResponseWriter, r *http.Request) {
	client := RedisClient()

	keys, err := client.Get("keys")
	if err != nil {
		panic(err)
	}

	finalString := ""

	arr :=  strings.Split(keys, ",")

	for index, element := range arr {
		key, err := client.Get(element)
		if err != nil {
			panic(err)
		}

		if index != len(arr) {
			finalString += key + ","
		} else {
			finalString += key
		}
	}

	if _, err := w.Write([]byte(finalString)); err != nil {
		panic(err)
	}
}

func delRedis(w http.ResponseWriter, r *http.Request) {
	client := RedisClient()

	keys, err := client.Get("keys")
	if err != nil {
		panic(err)
	}

	arr :=  strings.Split(keys, ",")

	for _, element := range arr {
		_, err := client.Del(element)
		if err != nil {
			panic(err)
		}
	}

	_, err = client.Del("keys")
	if err != nil {
		panic(err)
	}

	if _, err := w.Write([]byte("Done")); err != nil {
		panic(err)
	}
}

// Create a custom visitor struct which holds the rate limiter for each
// visitor and the last time that the visitor was seen.
type visitor struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// Change the the map to hold values of the type visitor.
var visitors = make(map[string]*visitor)
var mu sync.Mutex

// Run a background goroutine to remove old entries from the visitors map.
func init() {
	go cleanupVisitors()
}

func getVisitor(ip string) *rate.Limiter {
	mu.Lock()
	defer mu.Unlock()

	v, exists := visitors[ip]
	if !exists {
		limiter := rate.NewLimiter(1, 3)
		// Include the current time when creating a new visitor.
		visitors[ip] = &visitor{limiter, time.Now()}
		return limiter
	}

	// Update the last seen time for the visitor.
	v.lastSeen = time.Now()
	return v.limiter
}

// Every minute check the map for visitors that haven't been seen for
// more than 3 minutes and delete the entries.
func cleanupVisitors() {
	for {
		time.Sleep(time.Minute)

		mu.Lock()
		for ip, v := range visitors {
			if time.Since(v.lastSeen) > 3*time.Minute {
				delete(visitors, ip)
			}
		}
		mu.Unlock()
	}
}

func limit(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			log.Println(err.Error())
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}

		limiter := getVisitor(ip)
		if limiter.Allow() == false {
			http.Error(w, http.StatusText(429), http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func main() {

	checkEnv()

	initRedis()

	memcached, err := memory.NewAdapter(
		memory.AdapterWithAlgorithm(memory.LFU),
		memory.AdapterWithCapacity(10000000),
	)
	if err != nil {
		fmt.Println(err)
	}

	cacheClient, err := cache.NewClient(
		cache.ClientWithAdapter(memcached),
		cache.ClientWithTTL(1 * time.Minute),
		cache.ClientWithRefreshKey("opn"),
	)
	if err != nil {
		fmt.Println(err)
	}

	http.Handle("/outshine/client/send", limit(http.HandlerFunc(sendRedis)))
	http.Handle("/outshine/client/get", http.HandlerFunc(getRedis))
	http.Handle("/outshine/client/delete", http.HandlerFunc(delRedis))
	http.Handle("/outshine/client/cached", cacheClient.Middleware(http.HandlerFunc(getRedis)))

	if err := http.ListenAndServe(":" + PORT, nil); err != nil {
		panic(err)
	}
}
