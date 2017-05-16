//author tyf
//date   2017-02-08 15:53
//desc 

package index

import (
	"sync"
	"net/http"
	"github.com/tanyfx/ent/core/download"
	"github.com/tanyfx/ent/core/item"
	"github.com/tanyfx/ent/core/page"
)

type IndexProcessor interface {
	ProcessPage(*page.Page) []*item.ItemCTX
}

type IndexCTX struct {
	req            *http.Request
	indexPage      *page.Page
	downloader     download.Downloader
	processor      IndexProcessor
	itemDownloader download.Downloader
	itemProcessor  page.PageProcessor
}

func NewIndexCTX(req *http.Request, indexDownloader, itemDownloader download.Downloader,
processor IndexProcessor, itemProcessor page.PageProcessor) *IndexCTX {
	ctx := &IndexCTX{
		req: req,
		downloader: indexDownloader,
		processor: processor,
		itemDownloader: itemDownloader,
		itemProcessor: itemProcessor,
	}
	return ctx
}

func (p *IndexCTX) GetIndexPage() *page.Page {
	return p.indexPage
}

func (p *IndexCTX) GetRequest() *http.Request {
	return p.req
}

func (p *IndexCTX) GetProcessor() IndexProcessor {
	return p.processor
}

func (p *IndexCTX) GetItemProcessor() page.PageProcessor {
	return p.itemProcessor
}

func (p *IndexCTX) SetIndexDownloader(downloader download.Downloader) *IndexCTX {
	p.downloader = downloader
	return p
}

func (p *IndexCTX) SetIndexProcessor(processor IndexProcessor) *IndexCTX {
	p.processor = processor
	return p
}

func (p *IndexCTX) SetItemDownloader(downloader download.Downloader) *IndexCTX {
	p.itemDownloader = downloader
	return p
}

func (p *IndexCTX) SetItemProcessor(processor page.PageProcessor) *IndexCTX {
	p.itemProcessor = processor
	return p
}

func (p *IndexCTX) ExtractItemCTX() []*item.ItemCTX {
	p.indexPage = p.downloader.Download(p.req)

	//log.Printf("index page content:\n%s\n", p.indexPage.GetBody())

	ctxList := p.processor.ProcessPage(p.indexPage)
	for i := range ctxList {
		ctxList[i].Downloader = p.itemDownloader
		ctxList[i].Processor = p.itemProcessor
		//ctxList[i].Meta = p.indexPage.GetMeta()
		for k, v := range p.indexPage.GetMeta() {
			ctxList[i].AddMeta(k, v)
		}
	}
	return ctxList
}

func IndexWorker(indexCTXChan chan *IndexCTX, itemCTXChan chan *item.ItemCTX, wg *sync.WaitGroup) {
	defer wg.Done()
	for indexCTX := range indexCTXChan {
		for _, itemCTX := range indexCTX.ExtractItemCTX() {
			itemCTXChan <- itemCTX
		}
	}
}