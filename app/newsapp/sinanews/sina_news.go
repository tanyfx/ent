//author tyf
//date   2017-02-14 16:19
//desc

package sinanews

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

type SinaUpdateProducer struct {
}

func (p *SinaUpdateProducer) Produce(ctxChan chan *news.SimpleCTX) {
	req, _ := http.NewRequest("GET", "http://ent.sina.cn/", nil)
	req.Header.Set("User-Agent", consts.MobileUA)
	ctxChan <- news.NewSimpleCTX(req, &sinaNewsExtractor{}, &sinaImgReplacer{}, &sinaIndexProcessor{})
}

type sinaImgReplacer struct {
}

func (p *sinaImgReplacer) ReplaceImgs(n *news.NewsItem, folderPath, urlPrefix string) (string, []news.NewsImg) {
	newsContent := n.Content
	imgList := []news.NewsImg{}
	imgStrRegexp := regexp.MustCompile(`<img.+?>`)
	dataSrcRegexp := regexp.MustCompile(`<img.+?data-src="(.*?)"`)
	srcRegexp := regexp.MustCompile(`\ssrc="(.*?)"`)
	matches := imgStrRegexp.FindAllString(newsContent, -1)
	imgPrefix := `<img src="`
	imgSuffix := `" width="100%" height="auto"`
	for _, imgStr := range matches {
		match := dataSrcRegexp.FindStringSubmatch(imgStr)
		if len(match) != 2 {
			match = srcRegexp.FindStringSubmatch(imgStr)
		}
		if len(match) == 2 {
			//fmt.Println("found img:", match[1])
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
			newsContent = strings.Replace(newsContent, oldStr, imgPrefix+newURL+imgSuffix, 1)
		}
	}
	return newsContent, imgList
}

type sinaNewsExtractor struct {
}

func (p *sinaNewsExtractor) ExtractNews(newsPage *page.Page) *news.NewsItem {
	n := &news.NewsItem{
		Link: newsPage.GetRequest().URL.String(),
	}

	//titleXpath := xmlpath.MustCompile(`//meta[@property="og:title"]/@content`)
	//timeXpath := xmlpath.MustCompile(`//section[@class="art_title"]//time/text()`)
	//authorXpath := xmlpath.MustCompile(`//a[@class="art_brand_info_company"]/text()`)
	//contentRegex := regexp.MustCompile(`data-sudaclick="kandian_a".*?>([\w\W]*?)</section>`)
	//contentXpath := xmlpath.MustCompile(`//section[@data-sudaclick="kandian_a"]`)

	titleXpath := xmlpath.MustCompile(`//meta[@property="og:title"]/@content`)
	timeXpath := xmlpath.MustCompile(`//meta[@property="article:published_time"]/@content`)
	authorXpath := xmlpath.MustCompile(`//meta[@property="article:author"]/@content`)
	//contentRegex := regexp.MustCompile(`data-sudaclick="kandian_a".*?>([\w\W]*?)</section>`)
	//contentXpath := xmlpath.MustCompile(`//section[@data-sudaclick="kandian_a"]`)

	root, err := xmlpath.ParseHTML(strings.NewReader(newsPage.GetBody()))
	if err != nil {
		log.Println(err.Error())
		return n
	}

	if title, ok := titleXpath.String(root); ok {
		n.Title = comm.EscapeStr(title)
	}
	if dateTime, ok := timeXpath.String(root); ok {
		n.Date = strings.TrimSpace(dateTime)
	}
	if author, ok := authorXpath.String(root); ok {
		//resultNews.Fields[NewsAuthor] = util.EscapeStr(author)
		n.Author = comm.EscapeStr(author)
	}
	tmpContent := getSinaNewsContent(newsPage.GetBody())
	if len(tmpContent) > 0 {
		tmpContent = strings.Replace(strings.Replace(tmpContent, "新浪娱乐讯", "", -1), "新浪娱乐", "", -1)
		n.Content = comm.EscapeStr(strings.Replace(tmpContent, "新浪", "", -1))
	}
	//fmt.Println(time.Now().Format(util.TimeFormat), "fetch sina news done:",
	//	resultNews.Fields[NewsTitle], resultNews.Fields[NewsLink])
	return n
}

type sinaIndexProcessor struct {
}

func (p *sinaIndexProcessor) ProcessPage(indexPage *page.Page) []*item.ItemCTX {
	ctxList := []*item.ItemCTX{}

	//newsLinkXpath := xmlpath.MustCompile(`//div[@class="carditems_box"]/div/a/@href`)
	//newsLinkXpath := xmlpath.MustCompile(`//div[@data-sudaclick="newslist"]/div/a/@href`)
	newsLinkXpath := xmlpath.MustCompile(`//section[@data-sudaclick="feedlist_conf_5"]/a/@href`)

	ztRegexp := regexp.MustCompile(`ent\.sina\.cn/zt_`)

	root, err := xmlpath.ParseHTML(strings.NewReader(indexPage.GetBody()))
	if err != nil {
		log.Println("error while parse html content", err.Error())
		return ctxList
	}

	iter := newsLinkXpath.Iter(root)
	for iter.Next() {
		newsURL := iter.Node().String()
		if !strings.HasPrefix(newsURL, "http://ent.sina") {
			fmt.Println(time.Now().Format(consts.TimeFormat), "not ent sina news, passed:", newsURL)
			continue
		}

		if m := ztRegexp.FindString(newsURL); len(m) > 0 {
			fmt.Println(time.Now().Format(consts.TimeFormat), "zt news, passed:", newsURL)
			continue
		}
		req, _ := http.NewRequest("GET", newsURL, nil)
		ctx := item.NewItemCTX(req, nil, nil)
		ctxList = append(ctxList, ctx)
	}
	return ctxList
}

func (p *sinaIndexProcessor) GetIndexName() string {
	return fmt.Sprint("update sina news")
}

//新浪新闻的图片src为一张空白图片，由js后续下载替换，故需要处理img标签，将空白图片src替换为实际图片src
func getSinaNewsContent(input string) string {
	contents := []string{}
	imgRegexp := regexp.MustCompile(`<section\s+class="art_pic_card">[\w\W]+?(<img.+?>)`)
	altRegexp := regexp.MustCompile(`alt="(.*?)"`)

	paraRegexp := regexp.MustCompile(`<p\s+class="art_t">[\w\W]*?</p>`)
	weiboRegexp := regexp.MustCompile(`<a\s+href="http://weibo.com.+?</a>`)
	imgMatches := imgRegexp.FindAllStringSubmatch(input, -1)
	for _, imgMatch := range imgMatches {

		if len(imgMatch) > 1 {
			imgStr := imgMatch[1]
			altMatch := altRegexp.FindStringSubmatch(imgMatch[1])
			if len(altMatch) > 1 && len(altMatch[1]) > 0 {
				imgStr = imgMatch[1] + altMatch[1]
			}
			contents = append(contents, imgStr)
		}
	}
	paraMatches := paraRegexp.FindAllString(input, -1)
	for _, paragraph := range paraMatches {
		contents = append(contents, weiboRegexp.ReplaceAllString(paragraph, ""))
	}
	return strings.Join(contents, "\n")
}
