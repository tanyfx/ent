//author tyf
//date   2017-02-17 14:52
//desc

package qqnews

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/tanyfx/ent/comm"
	"github.com/tanyfx/ent/comm/consts"
	"github.com/tanyfx/ent/comm/news"
	"github.com/tanyfx/ent/core/item"
	"github.com/tanyfx/ent/core/page"
	"gopkg.in/xmlpath.v2"
)

//TODO
//存在问题：浏览器中返回页面与程序中返回页面不一致
//浏览器返回html， 程序返回xml，且内容为分页展示，无法处理。
//http://info.3g.qq.com/g/s?aid=qqom_ss&id=qqom_20170217A05MZT
type qq3gIndexProducer struct {
}

func (p *qq3gIndexProducer) Produce(ctxChan chan *news.SimpleCTX) {
	baseURL := `http://info.3g.qq.com/g/chtab.htm`
	postData := make(url.Values)
	postData.Add(`tabId`, `ent_yaowen_tab`)
	postData.Add(`num`, `30`)
	postData.Add(`tabReq`, `{"alreadyLoad":{"shopping":[],"baseList":[],"card":[]},
	"alreadyLoadNum":0,"recentId":[""]}`)

	req, _ := http.NewRequest("POST", baseURL, strings.NewReader(postData.Encode()))
	req.Header.Set("User-Agent", consts.UserAgent)
	req.Header.Set("Content-Type", consts.ContentType)

	ctxChan <- news.NewSimpleCTX(req, &qq3gNewsExtractor{}, &qq3gImgReplacer{}, &qq3gIndexPageProcessor{})
	//ctxChan <- news.NewSimpleCTX(req, &SinaNewsExtractor{}, &SinaImgReplacer{}, &SinaIndexProcessor{})
}

type qq3gImgReplacer struct {
}

func (p *qq3gImgReplacer) ReplaceImgs(n *news.NewsItem, folderPath, urlPrefix string) (string, []news.NewsImg) {
	newsContent := n.Content
	imgList := []news.NewsImg{}
	log.Println(folderPath, urlPrefix)
	imgRegexp := regexp.MustCompile(`<img\s+src=".+?"\s+data-src="(.+?)"`)
	newStr := `<img src="`
	matches := imgRegexp.FindAllStringSubmatch(newsContent, -1)
	for _, match := range matches {
		if len(match) > 1 {
			imgURL := match[1]
			oldStr := match[0]
			imgName, err := comm.DownloadImage(imgURL, folderPath)
			if err != nil {
				log.Println(err.Error())
				continue
			}
			tmpImg := news.GenNewsImg(n.GetNewsID(), n.Date, n.Title, folderPath, imgName, imgURL)
			imgList = append(imgList, tmpImg)
			newURL := urlPrefix + imgName
			newsContent = strings.Replace(newsContent, oldStr, newStr+newURL+`"`, 1)
		}
	}
	n.Content = newsContent
	newsContent, tmpImgList := news.ReplaceQQVideoIframe(n, folderPath, urlPrefix, true)
	if len(tmpImgList) > 0 {
		imgList = append(imgList, tmpImgList...)
	}
	n.Content = newsContent
	return newsContent, imgList
}

type qq3gNewsExtractor struct {
}

func (p *qq3gNewsExtractor) ExtractNews(newsPage *page.Page) *news.NewsItem {
	titleXpath := xmlpath.MustCompile(`//*[@class="lincoapp-article"]/h1/text()`)
	dateXpath := xmlpath.MustCompile(`//*[@class="time"]/text()`)
	authorXpath := xmlpath.MustCompile(`//*[@class="resource"]/text()`)
	//contentXpath := xmlpath.MustCompile(`//*/article`)
	contentRegex := regexp.MustCompile(`<article>([\w\W]*?)</article>`)

	return extractQQNews(newsPage, titleXpath, authorXpath, dateXpath, contentRegex)
}

type qq3gIndexPageProcessor struct {
}

func (p *qq3gIndexPageProcessor) ProcessPage(indexPage *page.Page) []*item.ItemCTX {

	//log.Printf("3g qq index page content:\n%s\n", indexPage.GetBody())

	ctxList := []*item.ItemCTX{}

	var jsonData map[string]interface{}
	if err := json.Unmarshal([]byte(indexPage.GetBody()), &jsonData); err != nil {
		log.Println("error while unmarshal 3g qq json data:", indexPage.GetRequest().URL.String(), err.Error())
		return ctxList
	}

	newsList, found := jsonData[`entryList`]
	if !found {
		log.Println("get 0 3g qq news in json page")
		return ctxList
	}

	for _, tmpNews := range newsList.([]interface{}) {
		newsMap := tmpNews.(map[string]interface{})
		if comm.InterfaceToString(newsMap[`from`]) != `rec` {
			continue
		}
		newsURL := comm.InterfaceToString(newsMap[`url`])
		req, err := http.NewRequest("GET", newsURL, nil)
		if err != nil {
			log.Println("error while parse 3g qq news url:", newsURL, err.Error())
			continue
		}
		req.Header.Add("User-Agent", consts.MobileUA)

		//log.Println("3g qq index page cookies length:", len(indexPage.GetCookies()))

		//for _, ck := range indexPage.GetCookies() {
		//	req.AddCookie(ck)
		//}
		ctx := item.NewItemCTX(req, nil, nil)
		ctxList = append(ctxList, ctx)
	}
	//fmt.Println(time.Now().Format(consts.TimeFormat), "get 3g qq news context length:", len(ctxList))
	return ctxList
}

func (p *qq3gIndexPageProcessor) GetIndexName() string {
	return fmt.Sprint("update 3g qq news")
}
