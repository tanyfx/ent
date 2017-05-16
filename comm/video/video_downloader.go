//author tyf
//date   2017-02-17 22:44
//desc

package video

import (
	"gopkg.in/redis.v5"
	"github.com/tanyfx/ent/core/download"
	"net/http"
	"github.com/tanyfx/ent/core/page"
	"github.com/tanyfx/ent/comm/redisutil"
	"log"
	"fmt"
)

type VideoDownloader struct {
	RedisCli *redis.Client
	downloader *download.HttpDownloader
}

func GenVideoDownloader(redisCli *redis.Client) *VideoDownloader {
	return &VideoDownloader{
		RedisCli: redisCli,
		downloader: &download.HttpDownloader{},
	}
}

func (p *VideoDownloader) Download(req *http.Request) *page.Page {
	respPage := page.NewPage(req)
	exists, err := redisutil.ExistVideoLink(p.RedisCli, req.URL.String())
	if err != nil {
		log.Println("error while find video link in redis", err.Error(), req.URL.String())
	}
	if exists {
		errMsg := fmt.Sprintf("video link exists: %s", req.URL.String())
		respPage.SetStatus(false, errMsg)
		return respPage
	}
	return p.downloader.Download(req)
}