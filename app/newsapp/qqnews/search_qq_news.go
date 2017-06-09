//author tyf
//date   2017-02-17 14:31
//desc

package qqnews

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"

	"golang.org/x/text/encoding/simplifiedchinese"
	"golang.org/x/text/transform"
	"gopkg.in/xmlpath.v2"

	"github.com/tanyfx/ent/comm"
	"github.com/tanyfx/ent/comm/consts"
	"github.com/tanyfx/ent/comm/news"
	"github.com/tanyfx/ent/core/item"
	"github.com/tanyfx/ent/core/page"
)

type QQSearchIndexProducer struct {
}

func (p *QQSearchIndexProducer) Produce(ctxChan chan *news.SimpleCTX, starPairMap map[string]comm.StarIDPair) {
	baseURL := "http://news.sogou.com/news?sort=1&mode=2&"

	count := 0
	length := len(starPairMap)
	for searchName, pair := range starPairMap {
		count++
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
		searchIndexProcessor := &qqSearchIndexProcessor{
			searchIndex: count,
			total:       length,
			starPair:    pair,
		}
		ctx := news.NewSimpleCTX(req, &xwQQNewsExtractor{}, &xwImgReplacer{}, searchIndexProcessor)
		ctxChan <- ctx
	}
}

type qqSearchIndexProcessor struct {
	searchIndex int
	total       int
	starPair    comm.StarIDPair
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
	return resultList
}

func (p *qqSearchIndexProcessor) GetIndexName() string {
	prefix := fmt.Sprintf("(%d/%d)", p.searchIndex, p.total)
	name := fmt.Sprintf("search qq %s\t%s\t%s\tnews", prefix, p.starPair.NameCN, p.starPair.StarID)
	return name
}
