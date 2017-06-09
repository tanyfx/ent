//author tyf
//date   2017-02-21 11:37
//desc

package qqvideo

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/tanyfx/ent/comm/consts"
	"github.com/tanyfx/ent/comm/video"
	"github.com/tanyfx/ent/core/item"
	"github.com/tanyfx/ent/core/page"
	"gopkg.in/redis.v5"
)

func newQQVideoUpdateIndexCTX(vRedisCli *redis.Client) *video.SimpleVideoCTX {
	indexURL := "http://ivideo.3g.qq.com/video/api/pos@getList?action=covervideolist&vl_cid=201605051050" +
		"&vl_pageNo=1&vl_pageSize=20"
	req, _ := http.NewRequest("GET", indexURL, nil)
	req.Header.Add("User-Agent", consts.MobileUA)
	return video.NewSimpleVideoCTX(req, &updateMQQVideoExtractor{}, &updateIndexProcessor{},
		video.GenSimpleVideoDownloader(vRedisCli))
}

type UpdateQQVideoProducer struct {
}

func (p *UpdateQQVideoProducer) Produce(c chan *video.SimpleVideoCTX, vRedisCli *redis.Client) {
	c <- newQQVideoUpdateIndexCTX(vRedisCli)
}

type updateMQQVideoExtractor struct {
}

func (p *updateMQQVideoExtractor) ExtractVideo(vPage *page.Page) *video.VideoItem {
	metaMap := vPage.GetMeta()

	vTitle := metaMap[video.VideoTitle]
	vDate := metaMap[video.VideoDate]
	vLink := metaMap[video.VideoLink]
	v := video.NewVideoItem(vTitle, vDate, vLink)
	return v
}

type updateIndexProcessor struct {
}

func (p *updateIndexProcessor) ProcessPage(indexPage *page.Page) []*item.ItemCTX {
	return handleMobileQQVideoJson(indexPage)
}

func (p *updateIndexProcessor) GetIndexName() string {
	return fmt.Sprint("update qq video")
}

func handleMobileQQVideoJson(p *page.Page) []*item.ItemCTX {
	ctxList := []*item.ItemCTX{}
	content := p.GetBody()
	var data map[string]interface{}
	if err := json.Unmarshal([]byte(content), &data); err != nil {
		log.Println(err.Error())
		return ctxList
	}
	posData, found := data["pos@getList"]
	if !found {
		log.Println("json pos@getList not found")
		return ctxList
	}
	tmpData, found := posData.(map[string]interface{})["data"]
	if !found {
		log.Println("json pos@getList -> data not found")
		return ctxList
	}
	tmpVideoList, found := tmpData.(map[string]interface{})["list"]
	if !found {
		log.Println("json pos@getList -> data -> list not found")
		return ctxList
	}
	for _, videoStruct := range tmpVideoList.([]interface{}) {
		videoMap := videoStruct.(map[string]interface{})
		tmpTitle := videoMap["title"].(string)
		tmpVid := videoMap["vid"].(string)
		tmpDatetime := videoMap["pubTime"].(string)
		if len(tmpTitle) == 0 || len(tmpVid) == 0 {
			continue
		}
		tmpLink := "http://v.qq.com/iframe/player.html?tiny=0&auto=0&vid=" + tmpVid

		req, err := http.NewRequest("GET", tmpLink, nil)
		if err != nil {
			log.Println("error while generate http request, passed:", tmpLink, err.Error())
		}

		c := item.NewItemCTX(req, nil, nil)
		c.AddMeta(video.VideoDate, tmpDatetime)
		c.AddMeta(video.VideoTitle, tmpTitle)
		c.AddMeta(video.VideoLink, tmpLink)

		ctxList = append(ctxList, c)
	}

	fmt.Println(time.Now().Format(consts.TimeFormat), "get mobile qq video context length", len(ctxList))

	return ctxList
}
