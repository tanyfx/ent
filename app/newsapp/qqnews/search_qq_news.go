//author tyf
//date   2017-02-17 14:31
//desc 

package qqnews

import (
	"fmt"
	"log"
	"time"
	"net/http"
	"net/url"
	"strings"

	"gopkg.in/xmlpath.v2"
	"golang.org/x/text/transform"
	"golang.org/x/text/encoding/simplifiedchinese"

	"github.com/tanyfx/ent/comm"
	"github.com/tanyfx/ent/comm/consts"
	"github.com/tanyfx/ent/comm/news"
	"github.com/tanyfx/ent/core/page"
	"github.com/tanyfx/ent/core/item"
)

type QQSearchIndexProducer struct {
}

func (p *QQSearchIndexProducer) Produce(ctxChan chan *news.SimpleCTX, starPairMap map[string]comm.StarIDPair) {
	baseURL := "http://news.sogou.com/news?sort=1&mode=2&"

	for searchName, pair := range starPairMap {
		tmpQuery, err := url.ParseQuery("query=site:ent.qq.com+" + searchName)
		if err != nil {
			log.Println("error while parse query:", err.Error())
			continue
		}
		searchURL := baseURL + tmpQuery.Encode()
		req, err := http.NewRequest("GET", searchURL, nil)
		if err != nil {
			log.Println("error while gen new request:", err.Error())
			continue
		}
		searchIndexProcessor := &qqSearchIndexProcessor{pair}
		ctx := news.NewSimpleCTX(req, &xwQQNewsExtractor{}, &xwImgReplacer{}, searchIndexProcessor)
		ctxChan <- ctx
	}
}

type qqSearchIndexProcessor struct {
	starPair comm.StarIDPair
}

func (p *qqSearchIndexProcessor) ProcessPage(indexPage *page.Page) []*item.ItemCTX {
	resultList := []*item.ItemCTX{}
	newsLinkPath := xmlpath.MustCompile("//*[@class=\"vrTitle\"]/a/@href")
	contentReader := transform.NewReader(strings.NewReader(indexPage.GetBody()), simplifiedchinese.GBK.NewDecoder())
	root, err := xmlpath.ParseHTML(contentReader)
	if err != nil {
		log.Println("error while parse search qq news page:", err.Error())
		return resultList
	}
	iter := newsLinkPath.Iter(root)
	for iter.Next() {
		newsURL := iter.Node().String()
		if len(newsURL) == 0 {
			continue
		}
		mobileNewsURL := genXWQQNewsURL(newsURL)
		req, err := http.NewRequest("GET", mobileNewsURL, nil)
		if err != nil {
			log.Println("error while gen new http request:", mobileNewsURL, err.Error())
		}
		ctx := item.NewItemCTX(req, nil, nil)
		ctx.AddMeta(consts.SearchID, p.starPair.StarID)
		ctx.AddMeta(consts.SearchStar, p.starPair.NameCN)
		resultList = append(resultList, ctx)
	}
	fmt.Printf("%s get search %s\t%s\tqq news context length: %d\n",
		time.Now().Format(consts.TimeFormat), p.starPair.StarID, p.starPair.NameCN, len(resultList))
	return resultList
}