package main
import (
	"github.com/choleraehyq/reinet"
)

func hello(id string) string {
	return "Hello, " + id
}

func main() {
	reinet.Get("/:id([0-9]+)", hello)
	reinet.Run(":1234")
}