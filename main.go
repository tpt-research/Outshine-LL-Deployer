package main

import (
	"fmt"
	"github.com/fiorix/go-redis/redis"
	cache "github.com/victorspringer/http-cache"
	"github.com/victorspringer/http-cache/adapter/memory"
	"net/http"
	"os"
	"strings"
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

	if keys == "" {
		if err := client.Set("keys", keys + "key" + string(increment) ); err != nil {
			panic(err)
		}
	} else {
		if err := client.Set("keys", keys +  "," + "key" + string(increment) ); err != nil {
			panic(err)
		}
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

		if index == len(arr) {
			finalString += key + ","
		} else {
			finalString += key
		}
	}

	if _, err := w.Write([]byte(finalString)); err != nil {
		panic(err)
	}
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

	http.Handle("/outshine/client/send", http.HandlerFunc(sendRedis))
	http.Handle("/outshine/client/get", http.HandlerFunc(getRedis))
	http.Handle("/outshine/client/cached", cacheClient.Middleware(http.HandlerFunc(getRedis)))

	if err := http.ListenAndServe(":" + PORT, nil); err != nil {
		panic(err)
	}
}
