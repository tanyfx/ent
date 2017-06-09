//author tyf
//date   2017-02-21 11:37
//desc

package qqvideo

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/tanyfx/ent/comm"
	"github.com/tanyfx/ent/comm/consts"
	"github.com/tanyfx/ent/comm/video"
	"github.com/tanyfx/ent/core/download"
	"github.com/tanyfx/ent/core/item"
	"github.com/tanyfx/ent/core/page"
	"gopkg.in/redis.v5"
)

type SearchQQVideoProducer struct {
}

func (p *SearchQQVideoProducer) Produce(c chan *video.SimpleVideoCTX, vRedisCLi *redis.Client,
	pairMap map[string]comm.StarIDPair) {
	baseSearchURL := "http://node.video.qq.com/x/api/msearch?contextValue=last_end%3D15%26sort%3D2%26Video_play_time_max%3D1%26%26response%3D1&filterValue=sort%3D1&"
	suffix := "&contextType=3"
	//suffix := "&stag=0&smartbox_ab=#!filter=1"
	count := 0
	length := len(pairMap)
	for searchName, pair := range pairMap {
		count++
		query, err := url.ParseQuery("keyWord=" + searchName)
		if err != nil {
			log.Println("error while parse search query:", err.Error())
		}
		searchURL := baseSearchURL + query.Encode() + suffix
		req, err := http.NewRequest("GET", searchURL, nil)
		if err != nil {
			log.Println("error while generate http request, passed:", searchURL, err.Error())
			continue
		}
		indexProcessor := &searchIndexProcessor{
			searchIndex: count,
			total:       length,
			starPair:    pair,
		}
		req.Header.Add("User-Agent", consts.MobileUA)
		c <- video.NewSimpleVideoCTX(req, &mobileQQVideoExtractor{}, indexProcessor,
			&download.HttpDownloader{})
		fmt.Println(time.Now().Format(consts.TimeFormat), "sleep for 1 second")
		time.Sleep(time.Second)
	}
}

type searchIndexProcessor struct {
	searchIndex int
	total       int
	starPair    comm.StarIDPair
}

func (p *searchIndexProcessor) ProcessPage(indexPage *page.Page) []*item.ItemCTX {
	ctxList := handleMobileSearchJson(indexPage)
	for i := range ctxList {
		ctxList[i].AddMeta(consts.SearchID, p.starPair.StarID)
		ctxList[i].AddMeta(consts.SearchStar, p.starPair.NameCN)
	}
	//prefix := fmt.Sprintf("%s (%d/%d)", time.Now().Format(consts.TimeFormat), p.searchIndex, p.total)
	//fmt.Printf("%s search %s\t%s\tqq video context length %d\n",
	//	prefix, p.starPair.NameCN, p.starPair.StarID, len(ctxList))
	return ctxList
}

func (p *searchIndexProcessor) GetIndexName() string {
	prefix := fmt.Sprintf("(%d/%d)", p.searchIndex, p.total)
	name := fmt.Sprintf("search qq %s %s\t%s\tvideo", prefix, p.starPair.NameCN, p.starPair.StarID)
	return name
}

func handleMobileSearchJson(p *page.Page) []*item.ItemCTX {
	resultList := []*item.ItemCTX{}
	var jsonObj map[string]interface{}
	err := json.Unmarshal([]byte(p.GetBody()), &jsonObj)
	if err != nil {
		log.Println(err.Error())
		return resultList
	}
	if data, found := jsonObj["uiData"]; found {
		dataList := data.([]interface{})
		if len(dataList) == 0 {
			fmt.Println(time.Now().Format(consts.TimeFormat), "json data is empty, do nothing")
			return resultList
		}
		for _, tmpData := range dataList {
			tmpMap := tmpData.(map[string]interface{})
			tmpInfoList, found := tmpMap["data"]
			if !found {
				continue
			}
			videoInfoList := tmpInfoList.([]interface{})
			if len(videoInfoList) == 0 {
				continue
			}
			videoInfo := videoInfoList[0]
			videoMap := videoInfo.(map[string]interface{})
			tmpURL, urlFound := videoMap["webPlayUrl"]
			tmpType, typeFound := videoMap["dataType"]
			tmpDate, dateFound := videoMap["publishDate"]
			if !typeFound || !urlFound {
				continue
			}
			vDate := time.Now().Format("2006-01-02 15:04:05")
			if dateFound {
				vDate = comm.InterfaceToString(tmpDate)
			}
			vType := comm.InterfaceToString(tmpType)
			if vType != "1" {
				continue
			}
			vLink := comm.InterfaceToString(tmpURL)
			req, err := http.NewRequest("GET", vLink, nil)
			if err != nil {
				log.Println("error while generate video request:", err.Error())
				continue
			}
			req.Header.Add("User-Agent", consts.MobileUA)
			ctx := item.NewItemCTX(req, nil, nil)
			ctx.AddMeta(video.VideoDate, vDate)
			resultList = append(resultList, ctx)
		}
	}
	return resultList
}
