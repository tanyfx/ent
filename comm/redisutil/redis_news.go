//author tyf
//date   2017-02-09 18:08
//desc

package redisutil

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/tanyfx/ent/comm"
	"github.com/tanyfx/ent/comm/consts"
	"gopkg.in/redis.v5"
)

func GenNewsList(db *sql.DB) []SimpleNews {
	newsList := getNewsList(db)
	newsStarMap := getNewsStarMap(db)
	for i := range newsList {
		newsID := newsList[i].NewsID
		if pairs, found := newsStarMap[newsID]; found {
			newsList[i].StarPairs = pairs
		}
	}
	return newsList
}

func getNewsList(db *sql.DB) []SimpleNews {
	resultList := []SimpleNews{}
	queryStr := "select news_id, post_id, datetime, title, link from news"
	rows, err := db.Query(queryStr)
	if err != nil {
		log.Println("error while query from table news:", err.Error())
		return resultList
	}
	var tmpNewsID, tmpPostID, tmpDate, tmpTitle, tmpLink sql.NullString
	for rows.Next() {
		err = rows.Scan(&tmpNewsID, &tmpPostID, &tmpDate, &tmpTitle, &tmpLink)
		if err != nil {
			log.Println("error while scan news row:", err.Error())
			continue
		}
		if !tmpNewsID.Valid || !tmpTitle.Valid || !tmpLink.Valid {
			fmt.Println(time.Now().Format(consts.TimeFormat), "not valid news, pass:",
				tmpNewsID.String, tmpTitle.String, tmpLink.String)
			continue
		}
		tmpNews := SimpleNews{
			NewsID:   tmpNewsID.String,
			PostID:   tmpPostID.String,
			Title:    tmpTitle.String,
			Link:     tmpLink.String,
			NewsDate: tmpDate.String,
		}
		resultList = append(resultList, tmpNews)
	}
	return resultList
}

func getNewsStarMap(db *sql.DB) map[string][]comm.StarIDPair {
	resultMap := map[string][]comm.StarIDPair{}
	queryStr := "select news_id, star_id, star_name_cn from news_star"
	starPairRows, err := db.Query(queryStr)
	if err != nil {
		log.Println("error while query from table news_star:", err.Error())
		return resultMap
	}
	var tmpNewsID, tmpStarID, tmpStarName sql.NullString
	//resultMap := map[string][]consts.StarIDPair{}
	for starPairRows.Next() {
		err = starPairRows.Scan(&tmpNewsID, &tmpStarID, &tmpStarName)
		if err != nil {
			log.Println("error while scan news star row", err.Error())
			continue
		}
		if !tmpNewsID.Valid || !tmpStarID.Valid || !tmpStarName.Valid {
			fmt.Println(time.Now().Format(consts.TimeFormat), "not valid news star row, pass:",
				tmpNewsID.String, tmpStarID.String)
			continue
		}
		tmpPair := comm.StarIDPair{
			NameCN: tmpStarName.String,
			StarID: tmpStarID.String,
		}
		if pairs, found := resultMap[tmpNewsID.String]; found {
			resultMap[tmpNewsID.String] = append(pairs, tmpPair)
		} else {
			resultMap[tmpNewsID.String] = []comm.StarIDPair{tmpPair}
		}
	}
	return resultMap
}

func ExistNewsLink(client *redis.Client, link string) (bool, error) {
	//linkKey := "news_link:" + strings.TrimSpace(link)
	linkKey := consts.RedisNLinkPrefix + strings.TrimSpace(link)
	return existKey(client, linkKey)
}

func ExistNewsTitle(client *redis.Client, title string) (bool, error) {
	//titleKey := "news_title:" + strings.TrimSpace(title)
	titleKey := consts.RedisNTitlePrefix + strings.TrimSpace(title)
	return existKey(client, titleKey)
}

func GetNewsIDTitle(client *redis.Client, title string) (string, error) {
	//titleKey := "news_title:" + strings.TrimSpace(title)
	titleKey := consts.RedisNTitlePrefix + strings.TrimSpace(title)
	idStr, err := client.Get(titleKey).Result()
	//return strings.TrimPrefix(idStr, "news_id:"), err
	return strings.TrimPrefix(idStr, consts.RedisNIDPrefix), err
}

func MGetNewsID(client *redis.Client, title, link string) (string, error) {
	title = strings.TrimSpace(title)
	link = strings.TrimSpace(link)

	resultID := ""
	if !strings.HasPrefix(link, "http") {
		err := errors.New("news link not start with http: " + link)
		return resultID, err
	}

	//titleKey := "news_title:" + title
	titleKey := consts.RedisNTitlePrefix + title
	//linkKey := "news_link:" + link
	linkKey := consts.RedisNLinkPrefix + link
	idList, err := client.MGet(titleKey, linkKey).Result()

	for _, tmpID := range idList {
		if tmpID == nil {
			continue
		}

		newsID := tmpID.(string)
		if len(newsID) > 0 {
			resultID = newsID
			break
		}
	}
	//return strings.TrimPrefix(resultID, "news_id:"), err
	return strings.TrimPrefix(resultID, consts.RedisNIDPrefix), err
}
