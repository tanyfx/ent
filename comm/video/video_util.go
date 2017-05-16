//author tyf
//date   2017-02-17 22:44
//desc

package video

import (
	"fmt"
	"log"
	"time"
	"errors"
	"strconv"
	"strings"
	"database/sql"
	"gopkg.in/redis.v5"
	"github.com/huichen/sego"
	"github.com/tanyfx/ent/comm/consts"
	"github.com/tanyfx/ent/comm/textutil"
)

func genVideoDeduper(videoRedisCli *redis.Client, seg *sego.Segmenter) (*textutil.Deduper, error) {
	simScore := consts.SimScore
	recentDocs := []textutil.Doc{}
	oldDocs := textutil.GetRedisTitles(videoRedisCli, consts.RedisVTitlePrefix, consts.RedisVIDPrefix)
	return textutil.NewDeduper(simScore, recentDocs, oldDocs, seg)
}

func saveVideo(db *sql.DB, redisCli *redis.Client, v *VideoItem, starTaxonomyMap map[string]string) error {
	videoID, err := saveToVideoTable(db, v)
	if err != nil {
		errMsg := fmt.Sprintln("error while save video to video table", err.Error())
		log.Println(errMsg)
		return errors.New(errMsg)
	}
	v.videoID = videoID

	err = saveStarTag(db, v)
	if err != nil {
		log.Println(err.Error())
	}

	postID, err := saveToWordpress(db, v, starTaxonomyMap)
	if err != nil {
		errMsg := fmt.Sprintln("error while save video to wordpress", err.Error())
		return errors.New(errMsg)
	}
	v.postID = postID

	if err = saveVideoRedisKey(redisCli, v); err != nil {
		errMsg := fmt.Sprintln("error while set video redis key", err.Error())
		return errors.New(errMsg)
	}
	return nil
}

func saveToVideoTable(db *sql.DB, v *VideoItem) (videoID string, err error) {
	videoID = ""
	if len(v.Date) == 0 {
		v.Date = time.Now().Format("2006-01-02 15:04:05")
	}
	insertStr := "insert `video` set `video_link` = ?, `video_date` = ?, `video_title` = ?"
	result, err := db.Exec(insertStr, v.Link, v.Date, v.Title)
	if err != nil {
		return videoID, err
	}
	tmpID, err := result.LastInsertId()
	if err != nil {
		return videoID, err
	}
	videoID = strconv.FormatInt(tmpID, 10)
	v.videoID = videoID
	return videoID, nil
}

func saveStarTag(db *sql.DB, v *VideoItem) error {
	insertFormat := "insert ignore into `video_star` set star_id = %s, star_name_cn = %s, star_id_video_id = %s, " +
		"video_id = %s, video_title = %s"
	var err error
	for _, pair := range v.Stars {
		starVideoID := pair.StarID + "_" + v.videoID
		insertStr := fmt.Sprintf(insertFormat, pair.StarID, pair.NameCN, starVideoID, v.videoID, v.Title)
		_, err = db.Exec(insertStr)
		if err != nil {
			log.Println("error while add star label:", pair.NameCN, v.Title, err.Error())
		}
	}
	return err
}

func saveToWordpress(db *sql.DB, v *VideoItem, starTaxonomyMap map[string]string) (postID string, err error) {

}

func saveVideoRedisKey(redisCli *redis.Client, v *VideoItem) error {
	if len(v.videoID) == 0 {
		return errors.New("error while set video redis key: video id is nil." + v.Title)
	}
	stars := ""
	starList := []string{}
	for _, pair := range v.Stars {
		tmpStar := pair.NameCN + consts.DELM + pair.StarID
		starList = append(starList, tmpStar)
	}
	stars = strings.Join(starList, consts.SEP)
	videoTitle := strings.TrimSpace(v.Title)
	videoLink := strings.TrimSpace(v.Link)

	videoMap := map[string]string{
		VideoTitle : videoTitle,
		VideoLink : videoLink,
		VideoDate : v.Date,
	}
	if len(stars) > 0 {
		videoMap["star_pairs"] = stars
	}
	if err := redisCli.HMSet(consts.RedisVIDPrefix + v.videoID, videoMap).Err(); err != nil {
		return errors.New("error while hmset video:" + v.videoID + " " + err.Error())
	}

	if err := redisCli.Set(consts.RedisVTitlePrefix + videoTitle, consts.RedisVIDPrefix + v.videoID, 0).Err(); err != nil {
		errStr := fmt.Sprintln("error while set video title key:", v.videoID, v.Title, err.Error())
		return errors.New(errStr)
	}

	if err := redisCli.Set(consts.RedisVLinkPrefix + videoLink, consts.RedisVIDPrefix + v.videoID, 0).Err(); err != nil {
		errStr := fmt.Sprintln("error while set video link key:", v.videoID, v.Link, err.Error())
		return errors.New(errStr)
	}

	redisCli.Expire(consts.RedisVTitlePrefix + videoTitle, consts.Year)
	redisCli.Expire(consts.RedisVLinkPrefix + videoLink, consts.Year)
	redisCli.Expire(consts.RedisVIDPrefix + v.videoID, consts.Year)
	return nil
}