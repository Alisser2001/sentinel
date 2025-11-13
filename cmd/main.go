package main

import (
	"sentinel/monitor"
)

func main() {
	engine := monitor.NewEngine()
	engine.Run()
}