//author tyf
//date   2017-02-17 22:44
//desc

package video

import (
	"database/sql"
	"log"
	"strings"

	"regexp"
	"sync"

	"github.com/huichen/sego"
	"github.com/tanyfx/ent/comm"
	"github.com/tanyfx/ent/comm/consts"
	"github.com/tanyfx/ent/comm/textutil"
	"github.com/tanyfx/ent/comm/wordpressutil"
	"gopkg.in/redis.v5"
)

func GenVideoDeduper(videoRedisCli *redis.Client, seg *sego.Segmenter) (*textutil.Deduper, error) {
	simScore := consts.SimScore
	recentDocs := []textutil.Doc{}
	oldDocs := textutil.GetRedisTitles(videoRedisCli, consts.RedisVTitlePrefix, consts.RedisVIDPrefix)
	return textutil.NewDeduper(simScore, recentDocs, oldDocs, seg)
}

//生成视频段落
func genVideoParagraph(videoLink, videoTitle string) string {
	if len(videoLink) == 0 {
		return ""
	}
	result := `<h2>` + videoTitle + `</h2>
<iframe src="` + videoLink + `" width="300" height="150" frameborder="0" allowfullscreen="allowfullscreen"></iframe>
<hr />`

	if strings.Contains(videoLink, "letv.com") {
		result = `<h2>` + videoTitle + `</h2>
<object style="border:0px;width:640px;height:498px" type="application/x-shockwave-flash" data="` + videoLink +
			`" ></object>`
	}
	return result
}

//增量更新
func addToVideoPost(db *sql.DB, videoPost *wordpressutil.Post, videoList []string, postID string) error {
	queryStr := "select post_content from wp_posts where id = ?"
	row := db.QueryRow(queryStr, postID)
	var tmpContent sql.NullString
	err := row.Scan(&tmpContent)
	if err != nil {
		return err
	}
	brk := "\n" + consts.WPBreak + "\n"
	if len(tmpContent.String) == 0 {
		videoPost.PostContent = comm.JoinWithBreak(videoList, brk, "\n", consts.WPBreakGap)
	} else {
		videoParas := strings.SplitN(tmpContent.String, consts.WPBreak, 2)
		videoParas[0] = addWithBreak(videoList, brk, videoParas[0])
		for i := range videoParas {
			videoParas[i] = strings.TrimSpace(videoParas[i])
		}
		videoPost.PostContent = strings.Join(videoParas, brk)
	}
	//videoPost.PostContent = videoPost.PostContent + tmpContent.String
	return wordpressutil.UpdatePostContent(db, videoPost, postID)
}

//
func addWithBreak(videoList []string, brk, content string) string {
	videoRegexp := regexp.MustCompile(`<h2[\w\W]+?/>`)
	match := videoRegexp.FindAllString(content, -1)
	videoList = append(videoList, match...)
	sep := "\n"
	return comm.JoinWithBreak(videoList, brk, sep, consts.WPBreakGap)
}

//collect videos and save video post
func SaveVideoWorker(vChan chan *VideoItem, wg *sync.WaitGroup, db *sql.DB, videoCli *redis.Client,
	starTaxonomyMap map[string]string) {
	defer wg.Done()

	starIDVListMap := map[string][]string{}     //star_id -> video_paragraph list
	starPairMap := map[string]comm.StarIDPair{} //star_id -> star_id_pair
	for v := range vChan {

		vID, err := saveToVideoTable(db, v)
		if err != nil {
			log.Println("error while save video to video table, passed:", err.Error())
			continue
		}
		v.videoID = vID
		if err := saveStarTag(db, v); err != nil {
			log.Println("error while save video star tag into video_star table", err.Error())
		}

		if err := saveVideoRedisKey(videoCli, v); err != nil {
			log.Println("error while save video into redis", err.Error())
		}

		vPara := genVideoParagraph(v.Link, v.Title)
		for _, pair := range v.Stars {
			starID := pair.StarID
			starPairMap[starID] = pair
			vList, found := starIDVListMap[starID]
			if found {
				starIDVListMap[starID] = append(vList, vPara)
			} else {
				starIDVListMap[starID] = []string{vPara}
			}
		}
		updateVideoStatus(db, vID)
	}

	for starID, vList := range starIDVListMap {
		pair := starPairMap[starID]
		taxonomyID := starTaxonomyMap[starID]
		vp := VideoPost{
			termTaxonomyID: taxonomyID,
			starNameCN:     pair.NameCN,
			videoList:      vList,
		}
		_, err := saveVideoPost(db, vp)
		if err != nil {
			log.Println("error while save video post:", vp.starNameCN, err.Error())
		}
	}
}
