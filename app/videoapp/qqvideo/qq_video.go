//author tyf
//date   2017-02-18 15:10
//desc

package qqvideo

import (
	"regexp"
	"strings"
	"time"

	"github.com/tanyfx/ent/comm/video"
	"github.com/tanyfx/ent/core/page"
)

const qqVideoPrefix string = "http://v.qq.com/iframe/player.html?tiny=0&auto=0&vid="

type mobileQQVideoExtractor struct {
}

func (p *mobileQQVideoExtractor) ExtractVideo(vPage *page.Page) *video.VideoItem {
	metaMap := vPage.GetMeta()
	v := &video.VideoItem{}
	vInfoRegexp := regexp.MustCompile(`tlux\.dispatch\('\$video'[\w\W]+?;`)
	titleRegexp := regexp.MustCompile(`"title":"(.*?)"`)
	vidRegexp := regexp.MustCompile(`"vid":"(.*?)"`)

	info := vInfoRegexp.FindString(string(vPage.GetBody()))
	if len(info) == 0 {
		return v
	}
	titleMatch := titleRegexp.FindStringSubmatch(info)
	if len(titleMatch) > 1 {
		v.Title = strings.TrimSpace(titleMatch[1])
	}
	vidMatch := vidRegexp.FindStringSubmatch(info)
	if len(vidMatch) > 1 {
		v.Link = qqVideoPrefix + strings.TrimSpace(vidMatch[1])
	}
	v.Date = metaMap[video.VideoDate]
	if len(v.Date) == 0 {
		v.Date = time.Now().Format("2006-01-02 15:04:05")
	}
	return v
}
