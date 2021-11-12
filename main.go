package main

import (
	"log"

	"github.com/gonejack/html-to-email/cmd"
)

func main() {
	var c cmd.HTMLToEmail
	if e := c.Run(); e != nil {
		log.Fatal(e)
	}
}
