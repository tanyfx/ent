//author tyf
//date   2017-02-17 14:42
//desc

package qqnews

import (
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/tanyfx/ent/comm"
	"github.com/tanyfx/ent/comm/consts"
	"github.com/tanyfx/ent/comm/news"
	"github.com/tanyfx/ent/core/item"
	"github.com/tanyfx/ent/core/page"
	"gopkg.in/xmlpath.v2"
)

type XWIndexProducer struct {
}

func (p *XWIndexProducer) Produce(ctxChan chan *news.SimpleCTX) {
	req, _ := http.NewRequest("GET", "http://xw.qq.com/m/ent/", nil)
	req.Header.Set("User-Agent", consts.MobileUA)
	ctxChan <- news.NewSimpleCTX(req, &xwQQNewsExtractor{}, &xwImgReplacer{}, &xwIndexPageProcessor{})
}

type xwImgReplacer struct {
}

func (p *xwImgReplacer) ReplaceImgs(n *news.NewsItem, folderPath, urlPrefix string) (string, []news.NewsImg) {
	imgList := []news.NewsImg{}
	newsContent := n.Content
	imgRegexp := regexp.MustCompile(`<img.+?src="(.*?)"`)
	matches := imgRegexp.FindAllStringSubmatch(newsContent, -1)
	for _, match := range matches {
		if len(match) > 1 {
			//fmt.Println("found news img:", match[1])
			imgURL := match[1]
			imgName, err := comm.DownloadImage(imgURL, folderPath)
			if err != nil {
				log.Println("error while download img", imgURL, err.Error())
				continue
			}
			tmpImg := news.GenNewsImg(n.GetNewsID(), n.Date, n.Title, folderPath, imgName, imgURL)
			imgList = append(imgList, tmpImg)
			newURL := urlPrefix + imgName
			newsContent = strings.Replace(newsContent, imgURL, newURL, 1)
		}
	}
	isHidden := true
	if len(matches) == 0 {
		isHidden = false
	}
	n.Content = newsContent
	newsContent, tmpImgList := news.ReplaceQQVideoIframe(n, folderPath, urlPrefix, isHidden)
	if len(tmpImgList) > 0 {
		imgList = append(imgList, tmpImgList...)
	}
	n.Content = newsContent
	return newsContent, imgList
}

type xwQQNewsExtractor struct {
}

func (p *xwQQNewsExtractor) ExtractNews(newsPage *page.Page) *news.NewsItem {
	titlePath := xmlpath.MustCompile(`//*[@class="title"]`)
	datePath := xmlpath.MustCompile(`//*[@class="time"]`)
	authorPath := xmlpath.MustCompile(`//*[@class="author"]`)
	//contentRegex := regexp.MustCompile("<div class=\"content fontsmall\">([\\W\\w]*?)<script")
	contentRegex := regexp.MustCompile(`class="content fontsmall">([\w\W]*?)\n.*?content/E`)

	return extractQQNews(newsPage, titlePath, authorPath, datePath, contentRegex)
}

// http://xw.qq.com/m/ent/
// http://xw.qq.com/service/interface.php?m=star&a=getdata
type xwIndexPageProcessor struct {
}

func (p *xwIndexPageProcessor) ProcessPage(indexPage *page.Page) []*item.ItemCTX {
	ctxList := []*item.ItemCTX{}

	newsLinkPath := xmlpath.MustCompile(`//*[@class="list"]//li/a/@href`)
	root, err := xmlpath.ParseHTML(strings.NewReader(indexPage.GetBody()))
	if err != nil {
		log.Println("error while parse xw qq index html page", err.Error())
		return ctxList
	}

	iter := newsLinkPath.Iter(root)
	for iter.Next() {
		newsURL := iter.Node().String()
		req, err := http.NewRequest("GET", newsURL, nil)
		if err != nil {
			log.Println("error while parse http request:", newsURL, err.Error())
			continue
		}
		ctx := item.NewItemCTX(req, nil, nil)
		ctxList = append(ctxList, ctx)
	}
	fmt.Println(time.Now().Format(consts.TimeFormat), "get xw qq news context length:", len(ctxList))

	return ctxList
}
