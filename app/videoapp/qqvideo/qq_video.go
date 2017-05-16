//author tyf
//date   2017-02-18 15:10
//desc 

package qqvideo

import (
	"regexp"
	"strings"
	"github.com/tanyfx/ent/core/page"
	"github.com/tanyfx/ent/comm/video"
)

const qqVideoPrefix string = "http://v.qq.com/iframe/player.html?tiny=0&auto=0&vid="

func extractMobileQQVideo(p *page.Page) *video.VideoItem {
	v := &video.VideoItem{
	}
	vInfoRegexp := regexp.MustCompile(`tlux\.dispatch\('\$video'[\w\W]+?;`)
	titleRegexp := regexp.MustCompile(`"title":"(.*?)"`)
	vidRegexp := regexp.MustCompile(`"vid":"(.*?)"`)

	info := vInfoRegexp.FindString(string(p.GetBody()))
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

	return v
}
