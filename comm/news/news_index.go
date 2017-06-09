//author tyf
//date   2017-02-09 15:54
//desc

package news

import (
	"net/http"

	"github.com/tanyfx/ent/comm"
	"github.com/tanyfx/ent/core/index"
)

type NewsUpdateProducer interface {
	Produce(chan *SimpleCTX)
}

type NewsSearchProducer interface {
	Produce(chan *SimpleCTX, map[string]comm.StarIDPair)
}

type SimpleCTX struct {
	Req            *http.Request
	Extractor      NewsExtractor
	ImgReplacer    ImgReplacer
	IndexProcessor index.IndexProcessor
}

func NewSimpleCTX(req *http.Request, extractor NewsExtractor, imgReplacer ImgReplacer,
	processor index.IndexProcessor) *SimpleCTX {
	return &SimpleCTX{
		Req:            req,
		Extractor:      extractor,
		ImgReplacer:    imgReplacer,
		IndexProcessor: processor,
	}
}
