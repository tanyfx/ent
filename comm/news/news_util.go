//author tyf
//date   2017-02-16 10:16
//desc

package news

import (
	"errors"
	"fmt"
	"log"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/huichen/sego"
	"github.com/tanyfx/ent/comm"
	"github.com/tanyfx/ent/comm/consts"
	"github.com/tanyfx/ent/comm/textutil"
	"gopkg.in/redis.v5"
)

func GenNewsDeduper(newsRedisCli *redis.Client, seg *sego.Segmenter) (*textutil.Deduper, error) {
	simScore := consts.SimScore
	recentDocs := []textutil.Doc{}
	oldDocs := textutil.GetRedisTitles(newsRedisCli, consts.RedisNTitlePrefix, consts.RedisNIDPrefix)
	return textutil.NewDeduper(simScore, recentDocs, oldDocs, seg)
}

func GenImgFolderPrefix() (folderPath, urlPrefix string, err error) {
	datePrefix := time.Now().Format("2006/01/02")
	urlPrefix = "/img/news_img/" + datePrefix + "/"

	folderPath = consts.NewsImgFolder + "/" + datePrefix
	//folderPath = strings.TrimSuffix(folderPath, "/") + "/" + datePrefix
	if err := os.MkdirAll(folderPath, 0755); err != nil {
		errMsg := fmt.Sprintf("error while make dir: %s, exit. %s", folderPath, err.Error())
		return folderPath, urlPrefix, errors.New(errMsg)
	}
	return folderPath, urlPrefix, nil
}

//isHidden 设置视频截图是否隐藏，true 隐藏，false 显示
func ReplaceQQVideoIframe(n *NewsItem, folderPath, urlPrefix string, isHidden bool) (string, []NewsImg) {
	content := n.Content
	imgList := []NewsImg{}
	result := n.Content
	jsRegexp := regexp.MustCompile(`<script[\w\W]*?</script>`)
	imgRegexp := regexp.MustCompile(`pic: "(.*?)"`)
	vidRegexp := regexp.MustCompile(`setVid\("(\w*?)"`)

	imgPrefix := `<img src="`
	imgSuffix := `" style="display:none;" />`

	if !isHidden {
		imgSuffix = `" class="video snapshot" />`
	}

	iframePrefix := `<iframe src="http://v.qq.com/iframe/player.html?vid=`
	iframeSuffix := `&amp;tiny=0&amp;auto=0" width="300" height="150" frameborder="0" ` +
		`allowfullscreen="allowfullscreen"></iframe>`
	matches := jsRegexp.FindAllString(content, -1)
	for _, m := range matches {
		newStrList := []string{}
		imgMatch := imgRegexp.FindStringSubmatch(m)
		if len(imgMatch) == 2 {
			imgURL := imgMatch[1]
			//fmt.Println(time.Now().Format(util.TimeFormat), "found img URL:", imgMatch[1])
			imgName, err := comm.DownloadImage(imgURL, folderPath)
			if err != nil {
				log.Println("error while download img:", imgMatch[1], err.Error())
			} else {
				imgSrc := strings.TrimSuffix(urlPrefix, "/") + "/" + imgName
				tmpNewsImg := GenNewsImg(n.newsID, n.Date, n.Title, folderPath, imgName, imgURL)
				imgList = append(imgList, tmpNewsImg)
				newStrList = append(newStrList, imgPrefix+imgSrc+imgSuffix)
				//newStrList = append(newStrList, imgPrefix + imgMatch[1] + imgSuffix)
			}
		}
		vidMatch := vidRegexp.FindStringSubmatch(m)
		if len(vidMatch) == 2 {
			newStrList = append(newStrList, iframePrefix+vidMatch[1]+iframeSuffix)
		}
		newStr := strings.Join(newStrList, "\n")
		result = strings.Replace(result, m, newStr, 1)
	}
	return result, imgList
}
