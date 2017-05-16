//author tyf
//date   2017-02-09 18:04
//desc 

package redisutil

import (
	"strings"
	"time"
	"fmt"
	"gopkg.in/redis.v5"
	"errors"
	"github.com/tanyfx/ent/comm"
	"github.com/tanyfx/ent/comm/consts"
)

const DELM string = "\001"
const SEP string = "\002"

//NewsID, PostID, Title, Link, NewsDate string, StarPairs []util.StarIDPair
type SimpleNews struct {
	NewsID    string
	PostID    string
	Title     string
	Link      string
	NewsDate  string
	StarPairs []comm.StarIDPair
}

//VideoID, Title, Link, VideoDate string, StarPairs []util.StarIDPair
type SimpleVideo struct {
	VideoID   string
	Title     string
	Link      string
	VideoDate string
	StarPairs []comm.StarIDPair
}

func (p *SimpleNews) SetRedisKey(client *redis.Client) error {

	newsID := strings.TrimSpace(p.NewsID)
	title := strings.TrimSpace(p.Title)
	link := strings.TrimSpace(p.Link)

	duration := comm.GetDuration(p.NewsDate)
	if duration >= consts.Year {
		duration = time.Hour * 24 * 180
	}
	duration = consts.Year - duration

	if len(p.Link) == 0 {
		fmt.Println(p.NewsID)
	}

	if len(p.Title) == 0 {
		fmt.Println(p.NewsID)
	}

	stars := ""
	starList := []string{}
	for _, pair := range p.StarPairs {
		tmpStar := pair.NameCN + DELM + pair.StarID
		starList = append(starList, tmpStar)
	}
	stars = strings.Join(starList, SEP)
	newsMap := map[string]string{
		"post_id": p.PostID,
		"news_title": title,
		"news_link": link,
		"news_date": p.NewsDate,
	}
	if len(stars) > 0 {
		newsMap["star_pairs"] = stars
	}
	//if err := client.HMSet("news_id:" + p.NewsID, newsMap).Err(); err != nil {
	if err := client.HMSet(consts.RedisNIDPrefix + newsID, newsMap).Err(); err != nil {
		return errors.New("error while hset news:" + p.NewsID + " " + err.Error())
	}

	//if err := client.Set("news_title:" + p.Title, "news_id:" + p.NewsID, 0).Err(); err != nil {
	if err := client.Set(consts.RedisNTitlePrefix + title, consts.RedisNIDPrefix + newsID, 0).Err(); err != nil {
		return errors.New("error while set news title key:" + p.NewsID + " " + p.Title + " " + err.Error())
	}
	//if err := client.Set("news_link:" + p.Link, "news_id:" + p.NewsID, 0).Err(); err != nil {
	if err := client.Set(consts.RedisNLinkPrefix + link, consts.RedisNIDPrefix + newsID, 0).Err(); err != nil {
		return errors.New("error while set news link key:" + p.NewsID + " " + p.Link + " " + err.Error())
	}

	client.Expire(consts.RedisNTitlePrefix + title, duration)
	client.Expire(consts.RedisNLinkPrefix + link, duration)
	client.Expire(consts.RedisNIDPrefix + newsID, duration)
	return nil
}

func (p *SimpleVideo) SetRedisKey(client *redis.Client) error {
	p.Link = strings.TrimSpace(p.Link)
	p.Title = strings.TrimSpace(p.Title)
	p.VideoID = strings.TrimSpace(p.VideoID)

	duration := comm.GetDuration(p.VideoDate)
	if duration >= consts.Year {
		duration = time.Hour * 24 * 180
	}
	duration = consts.Year - duration

	stars := ""
	starList := []string{}
	for _, pair := range p.StarPairs {
		tmpStar := pair.NameCN + DELM + pair.StarID
		starList = append(starList, tmpStar)
	}
	stars = strings.Join(starList, SEP)
	newsMap := map[string]string{
		"video_title": p.Title,
		"video_link": p.Link,
		"video_date": p.VideoDate,
	}
	if len(stars) > 0 {
		newsMap["star_pairs"] = stars
	}
	//if err := client.HMSet("video_id:" + p.VideoID, newsMap).Err(); err != nil {
	if err := client.HMSet(consts.RedisVIDPrefix + p.VideoID, newsMap).Err(); err != nil {
		return errors.New("error while hset video:" + p.VideoID + " " + err.Error())
	}

	//if err := client.Set("video_title:" + p.Title, "video_id:" + p.VideoID, 0).Err(); err != nil {
	if err := client.Set(consts.RedisVTitlePrefix + p.Title, consts.RedisVIDPrefix + p.VideoID, 0).Err(); err != nil {
		return errors.New("error while set video title key:" + p.VideoID + " " + p.Title + " " + err.Error())
	}
	if err := client.Set(consts.RedisVLinkPrefix + p.Link, consts.RedisVIDPrefix + p.VideoID, 0).Err(); err != nil {
		return errors.New("error while set video link key:" + p.VideoID + " " + p.Link + " " + err.Error())
	}

	client.Expire(consts.RedisVTitlePrefix + p.Title, duration)
	client.Expire(consts.RedisVLinkPrefix + p.Link, duration)
	client.Expire(consts.RedisVIDPrefix + p.VideoID, duration)
	return nil
}

func existKey(client *redis.Client, key string) (bool, error) {
	flag := false
	value, err := client.Get(key).Result()

	if len(value) > 0 {
		flag = true
	}
	if err == redis.Nil {
		flag = false
		err = nil
	}
	return flag, err
}
