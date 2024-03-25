package main

import (
	// "context"
	"task/internal/floodcontrol"
	"fmt"
)

var N int
var K int

func main() {
	fc, err := floodcontrol.NewRedisFloodControl("localhost:6379", 0, 1)
	if err != nil {
		fmt.Println("Error creating FloodControl:", err)
		return
	}
	defer fc.Close()

	//тут должен быть какой-то код, который записывает в скользящее окно новые запросы, предварительно сделав Check
	//для данного userID. Если True, то записываем, если False, то нет. Я намеренно не писал код
	//потому что тут надо прям модуль втыкать, который без рабоатает без остановки и пытается записать в окно.
	//помимо этого он каждую секунду удаляет старые запросы (см. ReadMe)
}
