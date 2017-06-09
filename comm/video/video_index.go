//author tyf
//date   2017-02-18 13:49
//desc

package video

import (
	"net/http"

	"github.com/tanyfx/ent/comm"
	"github.com/tanyfx/ent/core/download"
	"github.com/tanyfx/ent/core/index"
	"gopkg.in/redis.v5"
)

type VideoUpdateProducer interface {
	Produce(c chan *SimpleVideoCTX, vRedisCli *redis.Client)
}

type VideoSearchProducer interface {
	Produce(c chan *SimpleVideoCTX, vRedisCli *redis.Client, starPairMap map[string]comm.StarIDPair)
}

type SimpleVideoCTX struct {
	Req            *http.Request
	Extractor      VideoExtractor
	IndexProcessor index.IndexProcessor
	ItemDownloader download.Downloader
}

func NewSimpleVideoCTX(req *http.Request, extractor VideoExtractor, processor index.IndexProcessor,
	itemDownloader download.Downloader) *SimpleVideoCTX {
	return &SimpleVideoCTX{
		Req:            req,
		Extractor:      extractor,
		IndexProcessor: processor,
		ItemDownloader: itemDownloader,
	}
}
