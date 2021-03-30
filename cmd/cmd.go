package cmd

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/jordan-wright/email"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"path/filepath"

	"strings"

	"github.com/PuerkitoBio/goquery"
)

type HTMLToEmail struct {
	client http.Client

	From string
	To   string

	Verbose bool
}

func (h *HTMLToEmail) Run(htmlList []string) (err error) {
	if len(htmlList) == 0 {
		htmlList, _ = filepath.Glob("*.html")
	}
	if len(htmlList) == 0 {
		return errors.New("no HTML files given")
	}

	for _, html := range htmlList {
		if h.Verbose {
			log.Printf("processing %s", html)
		}
		err = h.processHTML(html)
		if err != nil {
			return fmt.Errorf("parse %s failed: %s", html, err)
		}
	}

	return
}
func (h *HTMLToEmail) processHTML(path string) (err error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}

	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(data))
	if err != nil {
		return
	}

	doc = h.cleanDoc(doc)

	doc.Find("img, script, link").Each(func(i int, selection *goquery.Selection) {
		var link string

		var attr string
		switch selection.Get(0).Data {
		case "link":
			attr = "href"
		default:
			attr = "src"
		}

		link, _ = selection.Attr(attr)
		if link == "" {
			return
		}

		fixed, err := h.fixLink(link)
		if err != nil {
			return
		}
		selection.SetAttr(attr, fixed)
	})
	doc.Find("link").Each(func(i int, selection *goquery.Selection) {
		ref, _ := selection.Attr("href")
		if ref == "" {
			return
		}
		fixed, err := h.fixLink(ref)
		if err != nil {
			return
		}
		selection.SetAttr("href", fixed)
	})

	html, err := doc.Html()
	mail := email.NewEmail()
	{
		mail.From = h.From
		mail.To = []string{h.To}
		mail.Subject = doc.Find("title").Text()
		mail.HTML = []byte(html)
	}

	output := strings.TrimSuffix(path, filepath.Ext(path)) + ".eml"
	content, err := mail.Bytes()
	if err != nil {
		return fmt.Errorf("cannot generate email: %w", err)
	}

	return ioutil.WriteFile(output, content, 0766)
}
func (_ *HTMLToEmail) cleanDoc(doc *goquery.Document) *goquery.Document {
	// remove inoreader ads
	doc.Find("body").Find(`div:contains("ads from inoreader")`).Closest("center").Remove()

	return doc
}
func (h *HTMLToEmail) fixLink(link string) (string, error) {
	u, err := url.Parse(link)
	if err != nil {
		if h.Verbose {
			log.Printf("cannot parse link %s", link)
		}
		return "", err
	}
	if u.Scheme == "" {
		u.Scheme = "https"
	}
	return u.String(), nil
}
