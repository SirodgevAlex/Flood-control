package main

import (
	"context"
	"task/floodcontrol"
    "fmt"
    "github.com/go-redis/redis/v8"
)

var N int
var K int

func main() {
	redisClient, err := redis.NewClient("localhost:6379")
    if err != nil {
        fmt.Println("Error creating Redis client:", err)
        return
    }
    defer redisClient.Close()

	
}