//author tyf
//date   2017-02-17 14:30
//desc 

package qqnews

import (
	"gopkg.in/xmlpath.v2"
	"regexp"
	"log"
	"strings"
	"fmt"
	"time"
	"github.com/tanyfx/ent/comm"
	"github.com/tanyfx/ent/comm/consts"
	"github.com/tanyfx/ent/comm/news"
	"github.com/tanyfx/ent/core/page"
)

func extractQQNews(p *page.Page, titleXpath, authorXpath, dateXpath *xmlpath.Path,
contentRegex *regexp.Regexp) *news.NewsItem {

	//log.Printf("qq news content:\n%s\n", p.GetBody())

	n := &news.NewsItem{
		Link: p.GetRequest().URL.String(),
	}

	imgRegexp := regexp.MustCompile(`<img.*?src="[\w\W]*?>`)
	content := p.GetBody()

	root, err := xmlpath.ParseHTML(strings.NewReader(content))
	if err != nil {
		log.Println("error while parse html xpath", n.Link, err.Error())
		return n
	}
	if title, ok := titleXpath.String(root); ok {
		n.Title = comm.EscapeStr(title)
	}
	if dateTime, ok := dateXpath.String(root); ok {
		n.Date = strings.TrimSpace(dateTime)
	}
	if author, ok := authorXpath.String(root); ok {
		n.Author = comm.EscapeStr(author)
	}

	if m := contentRegex.FindStringSubmatch(string(content)); len(m) > 1 {
		tmpContent := strings.Replace(strings.Replace(m[1], "腾讯娱乐讯", "", -1), "腾讯娱乐", "", -1)
		n.Content = comm.EscapeStr(strings.Replace(tmpContent, "腾讯", "", -1))
		//news.Fields[NewsContent] = ReplaceQQVideoIframe(tmpContent)
	}

	//log.Printf("regexp news content\n%s\n", n.Content)

	imgStr := imgRegexp.FindString(n.Content)
	if len(imgStr) == 0 {
		fmt.Println(time.Now().Format(consts.TimeFormat), "news contains no img, passed:", n.Link)
		n.Content = ""
		n.Content = ""
	}

	return n
}

//通过腾讯娱乐首页的新闻列表中的链接得到移动端新闻链接
//input: http://ent.qq.com/a/20160925/015143.htm or  /a/20160925/015143.htm
//output: http://xw.qq.com/c/ent/20160925015143
func genXWQQNewsURL(url string) string {
	regex := regexp.MustCompile(`/(\d+)`)
	m := regex.FindAllStringSubmatch(url, -1)
	if len(m) != 2 {
		return ""
	}
	suffix := m[0][1] + m[1][1]
	return "http://xw.qq.com/c/ent/" + suffix
}