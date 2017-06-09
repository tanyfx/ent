//author tyf
//date   2017-02-21 11:20
//desc

package main

import (
	"github.com/tanyfx/ent/comm"
	"github.com/tanyfx/ent/comm/video"
	"github.com/tanyfx/ent/core/item"
	"github.com/tanyfx/ent/core/page"
)

type updateProducer struct {
}

func (p *updateProducer) Produce(c chan *video.SimpleVideoCTX) {
}

type searchProducer struct {
}

func (p *searchProducer) Produce(c chan *video.SimpleVideoCTX, pairMap map[string]comm.StarIDPair) {
}

type videoExtractor struct {
}

func (p *videoExtractor) ExtractVideo(vPage *page.Page) *video.VideoItem {
	return nil
}

type indexProcessor struct {
}

func (p *indexProcessor) ProcessPage(indexPage *page.Page) []*item.ItemCTX {
	return []*item.ItemCTX{}
}

func (p *indexProcessor) GetIndexName() string {
	return ""
}
