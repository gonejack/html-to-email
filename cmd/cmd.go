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
func (h *HTMLToEmail) processHTML(html string) (err error) {
	data, err := ioutil.ReadFile(html)
	if err != nil {
		return err
	}

	mail := email.NewEmail()
	mail.From = h.From
	mail.To = []string{h.To}

	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(data))
	if err != nil {
		return
	}

	cids := make(map[string]string)
	doc.Find("img, link").Each(func(i int, selection *goquery.Selection) {
		var attr string
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

		ref, _ := selection.Attr(attr)
		switch {
		case ref == "":
			return
		case strings.HasPrefix(ref, "http://"):
			fallthrough
		case strings.HasPrefix(ref, "https://"):
			patched, err := h.patchReference(ref)
			if err != nil {
				log.Printf("cannot process reference %s", ref)
				return
			}
			selection.SetAttr(attr, patched)
		default:
			cid, exist := cids[ref]
			if exist {
				selection.SetAttr(attr, fmt.Sprintf("cid:%s", cid))
				return
			}

			cid, err := h.attachLocalFile(mail, ref)
			if err != nil {
				log.Printf("cannot attach %s: %s", ref, err)
				return
			}
			cids[ref] = cid

			selection.SetAttr(attr, fmt.Sprintf("cid:%s", cid))
		}
	})
	doc.Find("iframe").Each(func(i int, iframe *goquery.Selection) {
		src, _ := iframe.Attr("src")
		if src == "" {
			iframe.Remove()
		} else {
			iframe.ReplaceWithHtml(fmt.Sprintf(`<a href="%s">%s</a>`, src, src))
		}
	})
	doc.Find("script").Each(func(i int, script *goquery.Selection) {
		script.Remove()
	})

	htm, err := doc.Html()
	mail.Subject = doc.Find("title").Text()
	mail.HTML = []byte(htm)

	content, err := mail.Bytes()
	if err != nil {
		return fmt.Errorf("cannot generate email: %w", err)
	}

	target := strings.TrimSuffix(html, filepath.Ext(html)) + ".eml"

	return ioutil.WriteFile(target, content, 0766)
}
func (h *HTMLToEmail) patchReference(ref string) (string, error) {
	u, err := url.Parse(ref)
	if err != nil {
		if h.Verbose {
			log.Printf("cannot parse reference %s", ref)
		}
		return "", err
	}
	if u.Scheme == "" {
		u.Scheme = "http"
	}
	return u.String(), nil
}
func (h *HTMLToEmail) attachLocalFile(mail *email.Email, ref string) (cid string, err error) {
	localRef := ref
	fd, err := os.Open(localRef)
	if err != nil {
		localRef, _ = url.QueryUnescape(localRef)
		fd, err = os.Open(localRef)
	}
	if err != nil {
		return
	}

	fmime, err := mimetype.DetectFile(localRef)
	if err != nil {
		return
	}
	cid = md5str(ref) + fmime.Extension()
	attachment, err := mail.Attach(fd, cid, fmime.String())
	if err != nil {
		return
	}
	attachment.HTMLRelated = true

	return
}

func md5str(s string) string {
	return fmt.Sprintf("%x", md5.Sum([]byte(s)))
}
