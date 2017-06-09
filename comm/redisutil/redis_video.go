//author tyf
//date   2017-02-09 18:10
//desc

package redisutil

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strings"

	"github.com/tanyfx/ent/comm"
	"github.com/tanyfx/ent/comm/consts"
	"gopkg.in/redis.v5"
)

func GenVideoList(db *sql.DB) []SimpleVideo {
	videoList := getVideoList(db)
	videoStarMap := getVideoStarMap(db)
	for i := range videoList {
		videoID := videoList[i].VideoID
		if pairs, found := videoStarMap[videoID]; found {
			videoList[i].StarPairs = pairs
		}
	}
	return videoList
}

func getVideoList(db *sql.DB) []SimpleVideo {
	resultList := []SimpleVideo{}
	queryStr := "select video_id, video_date, video_title, video_link from video"
	rows, err := db.Query(queryStr)
	if err != nil {
		log.Println("error while query from table video:", err.Error())
		return resultList
	}
	var tmpVideoID, tmpDate, tmpTitle, tmpLink sql.NullString
	for rows.Next() {
		err = rows.Scan(&tmpVideoID, &tmpDate, &tmpTitle, &tmpLink)
		if err != nil {
			log.Println("error while scan video row:", err.Error())
			continue
		}
		if !tmpVideoID.Valid || !tmpTitle.Valid || !tmpLink.Valid {
			log.Println("not valid video, pass:",
				tmpVideoID.String, tmpTitle.String, tmpLink.String)
			continue
		}
		tmpVideo := SimpleVideo{
			VideoID:   tmpVideoID.String,
			Title:     tmpTitle.String,
			Link:      tmpLink.String,
			VideoDate: tmpDate.String,
		}
		resultList = append(resultList, tmpVideo)
	}
	return resultList
}

func getVideoStarMap(db *sql.DB) map[string][]comm.StarIDPair {
	resultMap := map[string][]comm.StarIDPair{}
	queryStr := "select video_id, star_id, star_name_cn from video_star"
	rows, err := db.Query(queryStr)
	if err != nil {
		log.Println("error while query from table video_star:", err.Error())
		return resultMap
	}
	var tmpVideoID, tmpStarID, tmpStarName sql.NullString
	for rows.Next() {
		err = rows.Scan(&tmpVideoID, &tmpStarID, &tmpStarName)
		if err != nil {
			log.Println("error while scan video row:", err.Error())
			continue
		}
		if !tmpVideoID.Valid || !tmpStarID.Valid || !tmpStarName.Valid {
			fmt.Println("not valid video star row, pass:",
				tmpVideoID.String, tmpStarID.String)
			continue
		}
		tmpPair := comm.StarIDPair{
			NameCN: tmpStarName.String,
			StarID: tmpStarID.String,
		}
		if pairs, found := resultMap[tmpVideoID.String]; found {
			resultMap[tmpVideoID.String] = append(pairs, tmpPair)
		} else {
			resultMap[tmpVideoID.String] = []comm.StarIDPair{tmpPair}
		}
	}
	return resultMap
}

func ExistVideoTitle(client *redis.Client, title string) (bool, error) {
	//titleKey := "video_title:" + strings.TrimSpace(title)
	titleKey := consts.RedisVTitlePrefix + strings.TrimSpace(title)
	return existKey(client, titleKey)
}

func ExistVideoLink(client *redis.Client, link string) (bool, error) {
	//linkKey := "video_link:" + strings.TrimSpace(link)
	linkKey := consts.RedisVLinkPrefix + strings.TrimSpace(link)
	return existKey(client, linkKey)
}

func MGetVideoID(client *redis.Client, title, link string) (string, error) {
	title = strings.TrimSpace(title)
	link = strings.TrimSpace(link)

	resultID := ""
	if !strings.HasPrefix(link, "http") {
		err := errors.New("video link not start with http: " + link)
		return resultID, err
	}

	//titleKey := "video_title:" + title
	titleKey := consts.RedisVTitlePrefix + title
	//linkKey := "video_link:" + link
	linkKey := consts.RedisVLinkPrefix + link
	idList, err := client.MGet(titleKey, linkKey).Result()

	for _, tmpID := range idList {
		if tmpID == nil {
			continue
		}
		if len(tmpID.(string)) > 0 {
			resultID = tmpID.(string)
			break
		}
	}
	//return strings.TrimPrefix(resultID, "video_id:"), err
	return strings.TrimPrefix(resultID, consts.RedisVIDPrefix), err
}
