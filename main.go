package main

import (
	"log"

	"github.com/gonejack/html-to-email/html2email"
)

func main() {
	cmd := html2email.HTMLToEmail{
		Options: html2email.MustParseOption(),
	}
	err := cmd.Run()
	if err != nil {
		log.Fatal(err)
	}
}
