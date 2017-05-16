//author tyf
//date   2017-02-10 15:34
//desc 

package news

import (
	"gopkg.in/redis.v5"
	"net/http"
	"log"
	"fmt"
	"github.com/tanyfx/ent/core/page"
	"github.com/tanyfx/ent/core/download"
	"github.com/tanyfx/ent/comm/redisutil"
)

type NewsDownloader struct {
	RedisCli *redis.Client
	downloader *download.HttpDownloader
}

func GenNewsDownloader(newsRedisCli *redis.Client) *NewsDownloader {
	return &NewsDownloader{
		RedisCli: newsRedisCli,
		downloader: &download.HttpDownloader{},
	}
}

func (p *NewsDownloader) Download(req *http.Request) *page.Page {
	respPage := page.NewPage(req)
	exists, err := redisutil.ExistNewsLink(p.RedisCli, req.URL.String())
	if err != nil {
		log.Println("error while find news link in redis", err.Error(), req.URL.String())
	}
	if exists {
		errMsg := fmt.Sprintf("news link exists: %s", req.URL.String())
		respPage.SetStatus(false, errMsg)
		return respPage
	}

	return p.downloader.Download(req)
}