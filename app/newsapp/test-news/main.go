//author tyf
//date   2017-02-18 22:16
//desc

package main

import (
	"errors"
	"fmt"
	"log"
	"runtime"
	"sync"
	"time"

	"github.com/tanyfx/ent/app/newsapp/qqnews"
	"github.com/tanyfx/ent/app/newsapp/sinanews"
	"github.com/tanyfx/ent/comm"
	"github.com/tanyfx/ent/comm/consts"
	"github.com/tanyfx/ent/comm/news"
	"github.com/tanyfx/ent/core/download"
	"github.com/tanyfx/ent/core/index"
	"github.com/tanyfx/ent/core/item"
	"github.com/tanyfx/ent/core/page"
)

type SimpleProcessor struct {
	extractor   news.NewsExtractor
	imgReplacer news.ImgReplacer
	folderPath  string
	urlPrefix   string
}

func (p *SimpleProcessor) ProcessPage(page *page.Page) error {
	n := p.extractor.ExtractNews(page)
	if !n.Valid() {
		return errors.New("news not valid")
	}
	n.Content, _ = p.imgReplacer.ReplaceImgs(n, p.folderPath, p.urlPrefix)
	fmt.Printf("get news:\ntitle:\t%s\nlink:\t%s\ndate\t%s\n%s", n.Title, n.Link, n.Date, n.Content)
	return nil
}

func (p *SimpleProcessor) Valid() bool {
	return true
}

var folderPath string
var urlPrefix string
var err error

var updateProducers = []news.NewsUpdateProducer{
	&sinanews.SinaUpdateProducer{},
	//&qqnews.QQ3gIndexProducer{},
	&qqnews.XWIndexProducer{},
}

var searchProducers = []news.NewsSearchProducer{
	&qqnews.QQSearchIndexProducer{},
}

func Init() error {
	folderPath, urlPrefix, err = news.GenImgFolderPrefix()
	if err != nil {
		return err
	}
	return nil
}

func main() {

	runtime.GOMAXPROCS(runtime.NumCPU())

	if err := Init(); err != nil {
		log.Fatalln(err.Error())
	}

	a := time.Now().Unix()
	simpleChan := make(chan *news.SimpleCTX, 10)
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
			processor := &SimpleProcessor{
				extractor:   simpleCTX.Extractor,
				imgReplacer: simpleCTX.ImgReplacer,
				folderPath:  folderPath,
				urlPrefix:   urlPrefix,
			}
			indexCTX := index.NewIndexCTX(simpleCTX.Req, &download.HttpDownloader{},
				&download.HttpDownloader{}, simpleCTX.IndexProcessor, processor)

			for _, itemCTX := range indexCTX.ExtractItemCTX() {
				itemChan <- itemCTX
			}
		}
		indexWG.Done()
	}()

	simpleWG.Add(len(updateProducers))
	for _, producer := range updateProducers {
		go func() {
			producer.Produce(simpleChan)
			simpleWG.Done()
		}()
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

	fmt.Printf("%s get news done, %d news added, %dm %ds used\n", time.Now().Format(consts.TimeFormat),
		counter.Count(), minutes, seconds)

}
