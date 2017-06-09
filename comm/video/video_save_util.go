//author tyf
//date   2017-02-17 22:44
//desc

package video

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/tanyfx/ent/comm"
	"github.com/tanyfx/ent/comm/consts"
	"github.com/tanyfx/ent/comm/wordpressutil"
	"gopkg.in/redis.v5"
)

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

	err = saveToWordpress(db, v, starTaxonomyMap)
	if err != nil {
		errMsg := fmt.Sprintln("error while save video to wordpress", err.Error())
		return errors.New(errMsg)
	}

	if err = saveVideoRedisKey(redisCli, v); err != nil {
		errMsg := fmt.Sprintln("error while set video redis key", err.Error())
		return errors.New(errMsg)
	}

	updateVideoStatus(db, videoID)
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
	insertStr := "insert ignore into `video_star` set star_id = ?, star_name_cn = ?, star_id_video_id = ?, " +
		"video_id = ?, video_title = ?"
	var err error
	for _, pair := range v.Stars {
		starVideoID := pair.StarID + "_" + v.videoID
		_, err = db.Exec(insertStr, pair.StarID, pair.NameCN, starVideoID, v.videoID, v.Title)
		if err != nil {
			log.Println("error while add star label:", pair.NameCN, v.Title, err.Error(), insertStr)
		}
	}
	return err
}

func saveToWordpress(db *sql.DB, v *VideoItem, starTaxonomyMap map[string]string) error {
	videoPara := genVideoParagraph(v.Link, v.Title)
	vp := VideoPost{
		videoList: []string{videoPara},
	}
	var err error
	for _, pair := range v.Stars {
		starID := pair.StarID
		taxonomyID, found := starTaxonomyMap[starID]
		if !found {
			continue
		}
		vp.termTaxonomyID = taxonomyID
		vp.starNameCN = pair.NameCN
		if _, err = saveVideoPost(db, vp); err != nil {
			log.Println("error while save video post", err.Error())
		}
	}
	return err
}

//如果wp_posts表中存在该明星视频，则增量更新，否则新增post
func saveVideoPost(db *sql.DB, vp VideoPost) (postID string, err error) {
	postID = ""
	if len(vp.videoList) == 0 {
		return postID, nil
	}
	brk := "\n" + consts.WPBreak + "\n"
	tmpPost := &wordpressutil.Post{
		PostDate:           time.Now().Format("2006-01-02 15:04:05"),
		PostTitle:          vp.starNameCN + "视频",
		PostName:           "video" + vp.termTaxonomyID,
		PostContent:        "",
		PostCateID:         consts.VideoCateID,
		CommentStatus:      "closed",
		PostExcerpt:        "",
		TermTaxonomyIDList: []string{vp.termTaxonomyID},
	}
	queryStr := "select object_id from wp_term_relationships where object_id in(select object_id from " +
		"wp_term_relationships where term_taxonomy_id = ?) and term_taxonomy_id = ?"
	row := db.QueryRow(queryStr, consts.VideoCateID, vp.termTaxonomyID)
	var tmpID sql.NullString
	err = row.Scan(&tmpID)
	switch {
	//post not exists, do insert post
	case err == sql.ErrNoRows:
		tmpPost.PostContent = comm.JoinWithBreak(vp.videoList, brk, "\n", consts.WPBreakGap)
		id, saveErr := wordpressutil.SavePost(db, tmpPost)
		postID = strconv.FormatInt(id, 10)
		return postID, saveErr
	case err != nil:
		return postID, errors.New("error while get post id from wp_term_relationships: " + err.Error())
	//post exists, do upate post
	default:
		postID = tmpID.String
		updateErr := addToVideoPost(db, tmpPost, vp.videoList, postID)
		if updateErr != nil {
			return postID, updateErr
		}
	}
	return postID, nil
}

//updat video status in video and video_star
func updateVideoStatus(db *sql.DB, videoID string) error {
	updateStr := "update video set status = 2 where video_id = ?"
	if _, err := db.Exec(updateStr, videoID); err != nil {
		log.Println("error while update video status:", videoID)
	}

	updateVideoStr := "update video_star set status = 2 where video_id = ?"
	if _, err := db.Exec(updateVideoStr, videoID); err != nil {
		log.Println("error while update video_star status:", videoID)
	}
	return nil
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
		VideoTitle: videoTitle,
		VideoLink:  videoLink,
		VideoDate:  v.Date,
	}
	if len(stars) > 0 {
		videoMap["star_pairs"] = stars
	}
	if err := redisCli.HMSet(consts.RedisVIDPrefix+v.videoID, videoMap).Err(); err != nil {
		return errors.New("error while hmset video:" + v.videoID + " " + err.Error())
	}

	if err := redisCli.Set(consts.RedisVTitlePrefix+videoTitle, consts.RedisVIDPrefix+v.videoID, 0).Err(); err != nil {
		errStr := fmt.Sprintln("error while set video title key:", v.videoID, v.Title, err.Error())
		return errors.New(errStr)
	}

	if err := redisCli.Set(consts.RedisVLinkPrefix+videoLink, consts.RedisVIDPrefix+v.videoID, 0).Err(); err != nil {
		errStr := fmt.Sprintln("error while set video link key:", v.videoID, v.Link, err.Error())
		return errors.New(errStr)
	}

	redisCli.Expire(consts.RedisVTitlePrefix+videoTitle, consts.Year)
	redisCli.Expire(consts.RedisVLinkPrefix+videoLink, consts.Year)
	redisCli.Expire(consts.RedisVIDPrefix+v.videoID, consts.Year)
	return nil
}
