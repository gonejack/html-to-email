package cmd

import (
	"bytes"
	"crypto/md5"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/gabriel-vasile/mimetype"
	"github.com/gonejack/email"
)

type HTMLToEmail struct {
	From string
	To   string

	Verbose bool
}

func (h *HTMLToEmail) Run(htmlList []string) (err error) {
	if len(htmlList) == 0 {
		return errors.New("no HTML files given")
	}

	for _, html := range htmlList {
		if h.Verbose {
			log.Printf("processing %s", html)
		}
		err = h.process(html)
		if err != nil {
			return fmt.Errorf("parse %s failed: %s", html, err)
		}
	}

	return
}
func (h *HTMLToEmail) process(html string) (err error) {
	saving := strings.TrimSuffix(html, filepath.Ext(html)) + ".eml"

	if s, e := os.Stat(saving); e == nil && s.Size() > 0 {
		log.Printf("%s skipped", html)
		return
	}
	data, err := ioutil.ReadFile(html)
	if err != nil {
		return err
	}
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(data))
	if err != nil {
		return
	}

	mail := email.NewEmail()
	{
		mail.From = h.From
		mail.To = []string{h.To}
		mail.Subject = doc.Find("title").Text()
		h.setDate(html, doc, mail)
		h.setAttachments(doc, mail)
	}

	htm, err := doc.Html()
	if err != nil {
		return
	}
	mail.HTML = []byte(htm)

	content, err := mail.Bytes()
	if err != nil {
		return fmt.Errorf("cannot generate email: %w", err)
	}

	return ioutil.WriteFile(saving, content, 0766)
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
	defer fd.Close()

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
func (h *HTMLToEmail) setDate(file string, doc *goquery.Document, mail *email.Email) {
	date := time.Now().Format(time.RFC1123Z)

	stat, _ := os.Stat(file)
	if stat != nil {
		date = stat.ModTime().Format(time.RFC1123Z)
	}

	meta := doc.Find(`meta[name="inostar:publish"]`).First()
	if meta.Length() > 0 {
		inopub, _ := meta.Attr("content")
		if inopub != "" {
			date = inopub
		}
	}

	mail.Headers.Set("Date", date)
}
func (h *HTMLToEmail) setAttachments(doc *goquery.Document, mail *email.Email) {
	cids := make(map[string]string)
	doc.Find("img, link").Each(func(i int, e *goquery.Selection) {
		var attr string
		switch e.Get(0).Data {
		case "link":
			attr = "href"
		case "img":
			attr = "src"
			e.RemoveAttr("loading")
			e.RemoveAttr("srcset")
		default:
			attr = "src"
		}

		ref, _ := e.Attr(attr)
		switch {
		case ref == "":
			return
		case strings.HasPrefix(ref, "data:"):
			return
		case strings.HasPrefix(ref, "http://"):
			fallthrough
		case strings.HasPrefix(ref, "https://"):
			patched, err := h.patchReference(ref)
			if err != nil {
				log.Printf("cannot process reference %s", ref)
				return
			}
			e.SetAttr(attr, patched)
		default:
			cid, exist := cids[ref]
			if exist {
				e.SetAttr(attr, fmt.Sprintf("cid:%s", cid))
				return
			}

			cid, err := h.attachLocalFile(mail, ref)
			if err != nil {
				log.Printf("cannot attach %s: %s", ref, err)
				return
			}
			cids[ref] = cid

			e.SetAttr(attr, fmt.Sprintf("cid:%s", cid))
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
}

func md5str(s string) string {
	return fmt.Sprintf("%x", md5.Sum([]byte(s)))
}
