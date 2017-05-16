//author tyf
//date   2017-02-17 18:24
//desc 

package main

import (
	"github.com/tanyfx/ent/comm/news"
	"github.com/tanyfx/ent/core/page"
	"github.com/tanyfx/ent/core/item"
	"github.com/tanyfx/ent/comm"
)

type indexProducer struct {
}

func (p *indexProducer) Produce(ctxChan chan *news.SimpleCTX) {

}

type searchIndexProducer struct {
}

func (p *searchIndexProducer) Produce(ctxChan chan *news.SimpleCTX, starPairMap map[string]comm.StarIDPair) {
}

type imgReplacer struct {
}

func (p *imgReplacer) ReplaceImgs(n *news.NewsItem, folderPath, urlPrefix string) (string, []news.NewsImg) {
	return "", []news.NewsImg{}
}

type newsExtractor struct {
}

func (p *newsExtractor) ExtractNews(newsPage *page.Page) *news.NewsItem {
	return nil
}

type indexProcessor struct {
}

func (p *indexProcessor) ProcessPage(indexPage *page.Page) []*item.ItemCTX {
	return []*item.ItemCTX{}
}