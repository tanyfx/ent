//author tyf
//date   2017-02-28 18:16
//desc

package main

import (
	"errors"
	"flag"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/tanyfx/ent/app/videoapp/qqvideo"
	"github.com/tanyfx/ent/comm"
	"github.com/tanyfx/ent/comm/consts"
	"github.com/tanyfx/ent/comm/video"
	"github.com/tanyfx/ent/core/download"
	"github.com/tanyfx/ent/core/index"
	"github.com/tanyfx/ent/core/item"
	"github.com/tanyfx/ent/core/page"
)

type SimpleVideoProcessor struct {
	extractor video.VideoExtractor
}

func (p *SimpleVideoProcessor) ProcessPage(itemPage *page.Page) error {

	for k, v := range itemPage.GetMeta() {
		fmt.Printf("%s = %s\n", k, v)
	}

	v := p.extractor.ExtractVideo(itemPage)
	if !v.Valid() {
		errMsg := fmt.Sprintf("video not valid: %s", v.Link)
		return errors.New(errMsg)
	}
	fmt.Printf("get video: \ntitle:\t%s\nlink:\t%s\ndate:\t%s\n", v.Title, v.Link, v.Date)
	return nil
}

func (p *SimpleVideoProcessor) Valid() bool {
	return true
}

var starPairMap map[string]comm.StarIDPair
var itemDownloader download.Downloader
var err error

var updateProducers = []video.VideoUpdateProducer{
//&qqvideo.UpdateQQVideoProducer{},
}

var searchProducers = []video.VideoSearchProducer{
	&qqvideo.SearchQQVideoProducer{},
}

func Init() error {
	starPairMap = map[string]comm.StarIDPair{
		"范冰冰": {
			StarID: "17",
			NameCN: "范冰冰",
		},
		"孙俪": {
			StarID: "3",
			NameCN: "孙俪",
		},
	}

	itemDownloader = &download.HttpDownloader{}
	return nil
}

func main() {
	a := time.Now().Unix()
	flag.Parse()
	log.SetFlags(log.Lshortfile | log.LstdFlags)

	if err := Init(); err != nil {
		log.Fatalln(err.Error())
	}

	simpleChan := make(chan *video.SimpleVideoCTX, 10)
	simpleWG := &sync.WaitGroup{}

	itemChan := make(chan *item.ItemCTX, 10)
	itemWG := &sync.WaitGroup{}
	itemWG.Add(consts.ThreadNum)
	counter := comm.NewCounter()

	for i := 0; i < consts.ThreadNum; i++ {
		go item.ItemWorker(itemChan, counter, itemWG, i)
	}

	indexWG := &sync.WaitGroup{}
	indexWG.Add(1)
	go func() {
		for simpleCTX := range simpleChan {
			processor := &SimpleVideoProcessor{
				extractor: simpleCTX.Extractor,
			}
			//if simpleCTX.ItemDownloader == nil {
			//	simpleCTX.ItemDownloader = itemDownloader
			//}
			indexCTX := index.NewIndexCTX(simpleCTX.Req, &download.HttpDownloader{},
				itemDownloader, simpleCTX.IndexProcessor, processor)

			for _, itemCTX := range indexCTX.ExtractItemCTX() {
				itemChan <- itemCTX

				//DEBUG
				//for k, v := range itemCTX.Meta {
				//	fmt.Printf("%s = %s\n", k, v)
				//}
			}
		}
		indexWG.Done()
	}()

	simpleWG.Add(len(updateProducers))
	for _, tmp := range updateProducers {
		go func(producer video.VideoUpdateProducer) {
			producer.Produce(simpleChan, nil)
			simpleWG.Done()
		}(tmp)
	}

	simpleWG.Add(len(searchProducers))
	for _, tmp := range searchProducers {
		go func(producer video.VideoSearchProducer) {
			producer.Produce(simpleChan, nil, starPairMap)
			simpleWG.Done()
		}(tmp)
	}

	simpleWG.Wait()
	close(simpleChan)
	indexWG.Wait()
	close(itemChan)
	itemWG.Wait()

	b := time.Now().Unix()
	duration := b - a
	minutes := duration / 60
	seconds := duration % 60

	fmt.Printf("%s get news done, %d video added, %dm %ds used\n", time.Now().Format(consts.TimeFormat),
		counter.Count(), minutes, seconds)

}
