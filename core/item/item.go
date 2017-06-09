//author tyf
//date   2017-02-08 15:53
//desc

package item

import (
	"errors"
	"log"
	"net/http"
	"sync"

	"github.com/tanyfx/ent/comm"
	"github.com/tanyfx/ent/core/download"
	"github.com/tanyfx/ent/core/page"
)

// Req *http.Request
// ItemPage *page.Page
// Downloader download.Downloader
// Processor  pageproc.PageProcessor
type ItemCTX struct {
	req        *http.Request       // http request
	itemPage   *page.Page          // response page
	Downloader download.Downloader // download page via Req
	Processor  page.PageProcessor
	Meta       map[string]string
}

func NewItemCTX(req *http.Request, downloader download.Downloader, processor page.PageProcessor) *ItemCTX {
	ctx := &ItemCTX{
		req:        req,
		Downloader: downloader,
		Processor:  processor,
		Meta:       map[string]string{},
	}
	return ctx
}

func (p *ItemCTX) Run() error {
	p.itemPage = p.Downloader.Download(p.req)
	if !p.itemPage.IsSucc() {
		log.Println("download item failed:", p.itemPage.ErrorMsg())
		return errors.New(p.itemPage.ErrorMsg())
	}
	p.itemPage.SetMetaMap(p.Meta)
	return p.Processor.ProcessPage(p.itemPage)
}

func (p *ItemCTX) GetItemPage() *page.Page {
	return p.itemPage
}

func (p *ItemCTX) GetRequest() *http.Request {
	return p.req
}

//func (p *ItemCTX) AddField(key string, m interface{}) {
//	p.Meta[key] = m
//}

func (p *ItemCTX) AddMeta(k, v string) {
	p.Meta[k] = v
}

func ItemWorker(ctxChan chan *ItemCTX, counter *comm.Counter, wg *sync.WaitGroup, index int) {
	defer wg.Done()
	log.Println("start item worker:", index)
	for itemCTX := range ctxChan {
		err := itemCTX.Run()
		if err == nil {
			counter.AddOne()
		}

		//DEBUG
		if err != nil {
			log.Println("item worker get error:", err.Error())
		}
	}
	log.Println("item worker", index, "stopped")
}
