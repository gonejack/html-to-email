package cmd

import (
	"bytes"
	"crypto/md5"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/gabriel-vasile/mimetype"
	"github.com/gonejack/email"
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

	mail := email.NewEmail()
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(data))
	if err != nil {
		return
	}
	doc = h.cleanDoc(doc)

	cids := make(map[string]string)
	doc.Find("img, script, link").Each(func(i int, selection *goquery.Selection) {
		var attr string
		{
			switch selection.Get(0).Data {
			case "link":
				attr = "href"
			case "img":
				attr = "src"
				selection.RemoveAttr("loading")
				selection.RemoveAttr("srcset")
			default:
				attr = "src"
			}
		}
		reference, _ := selection.Attr(attr)

		if reference == "" {
			return
		}

		if !strings.HasPrefix(reference, "http") {
			_, exist := cids[reference]
			if !exist {
				localRef := reference
				fd, err := os.Open(localRef)
				if err != nil {
					localRef, _ = url.QueryUnescape(localRef)
					fd, err = os.Open(localRef)
				}
				if err == nil {
					fmime, err := mimetype.DetectFile(localRef)
					if err != nil {
						log.Printf("cannot detect mime of %s: %s", path, err)
						return
					}
					cid := md5str(reference) + fmime.Extension()
					attachment, err := mail.Attach(fd, cid, fmime.String())
					if err != nil {
						log.Printf("cannot attach %s: %s", fd.Name(), err)
						return
					}
					attachment.HTMLRelated = true
					cids[reference] = cid
				}
			}
		}

		cid, exist := cids[reference]
		if exist {
			selection.SetAttr(attr, fmt.Sprintf("cid:%s", cid))
			return
		}

		fixed, err := h.fixLink(reference)
		if err == nil {
			selection.SetAttr(attr, fixed)
			return
		}

		log.Printf("cannot process reference %s", reference)
	})
	doc.Find("iframe").Each(func(i int, iframe *goquery.Selection) {
		src, _ := iframe.Attr("src")
		iframe.ReplaceWithHtml(fmt.Sprintf(`<a href="%s">%s</a>`, src, src))
	})

	html, err := doc.Html()
	mail.From = h.From
	mail.To = []string{h.To}
	mail.Subject = doc.Find("title").Text()
	mail.HTML = []byte(html)

	content, err := mail.Bytes()
	if err != nil {
		return fmt.Errorf("cannot generate email: %w", err)
	}

	target := strings.TrimSuffix(path, filepath.Ext(path)) + ".eml"

	return ioutil.WriteFile(target, content, 0766)
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
		u.Scheme = "http"
	}
	return u.String(), nil
}

func md5str(s string) string {
	return fmt.Sprintf("%x", md5.Sum([]byte(s)))
}
