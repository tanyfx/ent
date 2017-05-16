//author tyf
//date   2017-02-15 18:43
//desc 

package toutiaonews

import (
	"regexp"
	"strings"
	"fmt"
	"log"
	"time"
	"github.com/PuerkitoBio/goquery"
	"github.com/tanyfx/ent/comm"
	"github.com/tanyfx/ent/comm/consts"
	"github.com/tanyfx/ent/core/page"
	"github.com/tanyfx/ent/comm/news"
)

type ToutiaoImgReplacer struct {
}

func (p *ToutiaoImgReplacer) ReplaceImgs(n *news.NewsItem, folderPath, urlPrefix string) (string, []news.NewsImg) {
	newsContent := n.Content
	imgList := []news.NewsImg{}

	imgRegexp := regexp.MustCompile(`img src=".*?"`)

	matches := imgRegexp.FindAllString(newsContent, -1)
	for _, match := range matches {
		imgURL := string(match)
		imgURL = strings.TrimPrefix(imgURL, `img src="`)
		imgURL = strings.TrimSuffix(imgURL, `"`)
		if !strings.HasPrefix(imgURL, "http") {
			fmt.Println(time.Now().Format(consts.TimeFormat), "invalid img url, passed:", imgURL)
			continue
		}

		imgName, err := comm.DownloadImage(imgURL, folderPath)
		if err != nil {
			log.Println("error while download img", imgURL, err.Error())
			continue
		}

		tmpImg := news.GenNewsImg(n.GetNewsID(), n.Date, n.Title, folderPath, imgName, imgURL)
		imgList = append(imgList, tmpImg)
		newURL := strings.TrimSuffix(urlPrefix, "/") + "/" + imgName
		newsContent = strings.Replace(newsContent, imgURL, newURL, -1)
	}
	return newsContent, imgList
}

type ToutiaoNewsExtractor struct {
}

func (p *ToutiaoNewsExtractor) ExtractNews(newsPage *page.Page) *news.NewsItem {
	metaMap := newsPage.GetMeta()
	n := &news.NewsItem{
		Title: metaMap[news.NewsTitle],
		Link: newsPage.GetRequest().URL.String(),
	}

	pRegexp := regexp.MustCompile(`<p>.*</p>`)
	imgRegexp := regexp.MustCompile(`img\s.*src.".*?"`)

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(newsPage.GetBody()))
	if err != nil {
		fmt.Println("error while extract toutiao news:", newsPage.GetRequest().URL, err.Error())
		return n
	}
	tmpAuthor := doc.Find(".src").Text()
	tmpDate := doc.Find(".time").Text()
	if len(tmpDate) == 0 {
		tmpDate = time.Now().Format("2006-01-02 15:04:05")
	}
	if len(tmpAuthor) == 0 {
		tmpAuthor = "今日头条"
	}
	n.Date = tmpDate
	n.Author = strings.TrimSpace(tmpAuthor)

	contentSel := doc.Find(`.article-content`)
	contentSel.Find(`script`).Remove()
	contentSel.Find(`.mp-vote-box`).Remove()
	contentSel.Find(`.footnote`).Remove()
	if contentSel.Length() == 0 {
		log.Println("news content is nil, passed", n.Title, n.Link)
		return n
	}
	n.Content, _ = contentSel.Html()
	matches := pRegexp.FindAllString(n.Content, -1)
	upper := 0
	if len(matches) > 4 {
		upper = len(matches) - 4
	}
	for i, para := range matches {
		if i < 4 || i > upper {
			if strings.Contains(p, `微信`) || strings.Contains(p, `公众号`) || strings.Contains(p, `订阅号`) {
				n.Content = strings.Replace(n.Content, para, "", 1)
			}
		}
	}

	if imgStr := imgRegexp.FindString(n.Content); len(imgStr) == 0 {
		fmt.Println(time.Now().Format("2006/01/02 15:04:05"), "news content no img", n.Title)
		n.Content = ""
		n.Title = ""
	}
	return n
}

