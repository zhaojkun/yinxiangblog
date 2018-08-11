package utils

import (
	"fmt"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

func Render(title, content string) (string, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(content))
	if err != nil {
		return "", err
	}
	res, err := doc.Find("en-note").Html()
	if err != nil {
		return "", err
	}
	tpl := `<html><head><title>%s</title></head><body>%s</body></html>`
	res = fmt.Sprintf(tpl, title, res)
	return res, err
}
