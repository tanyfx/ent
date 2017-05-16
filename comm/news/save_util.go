//author tyf
//date   2017-02-14 15:56
//desc 

package news

import (
	"strings"
	"fmt"
	"gopkg.in/redis.v5"
	"errors"
	"database/sql"
	"time"
	"log"
	"strconv"
	"github.com/tanyfx/ent/comm/consts"
	"github.com/tanyfx/ent/comm/wordpressutil"
)

func saveNews(db *sql.DB, redisCli *redis.Client, n *NewsItem, starTaxonomyMap map[string]string) error {
	newsID, err := saveToNewsTable(db, n)
	if err != nil {
		errMsg := fmt.Sprintln("error while save news to news table", err.Error())
		log.Println(errMsg)
		return errors.New(errMsg)
	}
	n.newsID = newsID

	err = saveStarTag(db, n)
	if err != nil {
		log.Println(err.Error())
	}

	for _, img := range n.Imgs {
		img.SetNewsID(newsID)
		if err := img.SaveToMySQL(db); err != nil {
			log.Println("error while save img to table news_img:", err.Error())
		}
	}
	postID, err := saveToWordpress(db, n, starTaxonomyMap)
	if err != nil {
		errMsg := fmt.Sprintln("error while save news to wordpress", err.Error())
		return errors.New(errMsg)
	}
	n.postID = postID
	if err = saveNewsRedisKey(n, redisCli); err != nil {
		errMsg := fmt.Sprintln("error while set news redis key", err.Error())
		return errors.New(errMsg)
	}
	return nil
}

func saveToNewsTable(db *sql.DB, n *NewsItem) (newsID string, err error) {
	newsID = "-1"
	if len(n.Title) == 0 {
		err := errors.New("error save to news table: news_title is nil")
		return newsID, err
	}

	insertStr := "insert into `news` set "
	if len(n.Date) == 0 {
		n.Date = time.Now().Format(consts.TimeFormat)
	}
	fields := []string{
		fmt.Sprintf(FormatStr, NewsTitle, n.Title),
		fmt.Sprintf(FormatStr, NewsContent, n.Content),
		fmt.Sprintf(FormatStr, NewsLink, n.Link),
		fmt.Sprintf(FormatStr, NewsDate, n.Date),
	}
	if len(n.Author) > 0 {
		fields = append(fields, fmt.Sprintf(FormatStr, NewsAuthor, n.Author))
	}
	if len(n.Subtitle) > 0 {
		fields = append(fields, fmt.Sprintf(FormatStr, Subtitle, n.Subtitle))
	}
	if len(n.Summary) > 0 {
		fields = append(fields, fmt.Sprintf(FormatStr, Summary, n.Summary))
	}

	insertStr = insertStr + strings.Join(fields, ", ") + ";"
	res, err := db.Exec(insertStr)
	if err != nil {
		log.Printf(err.Error())
		return newsID, err
	}
	tmpID, err := res.LastInsertId()
	if err != nil {
		return newsID, err
	}
	newsID = strconv.FormatInt(tmpID, 10)
	n.newsID = newsID
	return newsID, nil
}

func saveStarTag(db *sql.DB, n *NewsItem) error {
	insertFormat := "insert ignore into `news_star` set `star_id` = '%s', `star_name_cn` = '%s', " +
		"`star_id_news_id` = '%s', `news_id` = '%s', `news_title` = '%s';"
	var err error
	for _, pair := range n.Stars {
		starNewsID := pair.StarID + "_" + n.newsID
		insertStr := fmt.Sprintf(insertFormat, pair.StarID, pair.NameCN, starNewsID, n.newsID, n.Title)
		_, err = db.Exec(insertStr)
		if err != nil {
			log.Println("error while insert news star tag into news_star table", err.Error())
		}
	}
	return err
}

//starTagMap map[string]string: star_id -> term_taxonomy_id
func saveToWordpress(db *sql.DB, n *NewsItem, starTaxonomyMap map[string]string) (postID string, err error) {
	//the termTaxonomyIDList means the term taxonomy id list of one news
	termTaxonomyIDList := []string{}
	for _, star := range n.Stars {
		termTaxonomyIDList = append(termTaxonomyIDList, starTaxonomyMap[star.StarID])
	}

	tmpPost := &wordpressutil.Post{
		PostDate: n.Date,
		PostContent: n.Content,
		PostTitle: n.Title,
		PostName: n.newsID,
		PostExcerpt: n.Summary,
		CommentStatus: "open",
		PostCateID: consts.NewsCateID,
		TermTaxonomyIDList: termTaxonomyIDList,
	}

	id, err := wordpressutil.SavePost(db, tmpPost)
	if err != nil {
		errMsg := fmt.Sprintln("error while insert news into wp_posts:", err.Error())
		return postID, errors.New(errMsg)
	}

	postID = strconv.FormatInt(id, 10)

	updateStr := "update news set post_id = ?, status = 2 where news_id = ?;"
	if _, err = db.Exec(updateStr, postID, n.newsID); err != nil {
		errMsg := fmt.Sprintln("error while update post id in table news", err.Error())
		return postID, errors.New(errMsg)
	}

	updateNewsStarStr := "update `news_star` set status = 2 where news_id = ?"
	if _, err = db.Exec(updateNewsStarStr, n.newsID); err != nil {
		errMsg := fmt.Sprintln("error while update news status in table news_star", err.Error())
		return postID, errors.New(errMsg)
	}
	return postID, nil
}

func saveNewsRedisKey(n *NewsItem, redisCli *redis.Client) error {
	if len(n.newsID) == 0 {
		return errors.New("error while set news redis key: news id is nil. " + n.Title)
	}
	stars := ""
	starList := []string{}
	for _, pair := range n.Stars {
		tmpStar := pair.NameCN + consts.DELM + pair.StarID
		starList = append(starList, tmpStar)
	}
	stars = strings.Join(starList, consts.SEP)

	newsTitle := strings.TrimSpace(n.Title)
	newsLink := strings.TrimSpace(n.Link)
	newsMap := map[string]string{
		"news_title": newsTitle,
		"news_link":  newsLink,
		"news_date":  n.Date,
	}

	if len(n.postID) > 0 {
		newsMap["post_id"] = n.postID
	}

	if len(stars) > 0 {
		newsMap["star_pairs"] = stars
	}

	//if err := client.HMSet("news_id:" + n.NewsID, newsMap).Err(); err != nil {
	if err := redisCli.HMSet(consts.RedisNIDPrefix + n.newsID, newsMap).Err(); err != nil {
		return errors.New("error while HMSet news:" + n.newsID + " " + err.Error())
	}
	//if err := redisCli.Set("news_title:" + newsTitle, "news_id:" + n.newsID, 0).Err(); err != nil {
	if err := redisCli.Set(consts.RedisNTitlePrefix + newsTitle, consts.RedisNIDPrefix + n.newsID, 0).Err(); err != nil {
		errStr := fmt.Sprintln("error while Set news title key:", n.newsID, n.Title, err.Error())
		return errors.New(errStr)
	}
	//if err := redisCli.Set("news_link:" + newsLink, "news_id:" + n.newsID, 0).Err(); err != nil {
	if err := redisCli.Set(consts.RedisNLinkPrefix + newsLink, consts.RedisNIDPrefix + n.newsID, 0).Err(); err != nil {
		errStr := fmt.Sprintln("error while Set news link key:", n.newsID, n.Link, err.Error())
		return errors.New(errStr)
	}

	redisCli.Expire(consts.RedisNTitlePrefix + newsTitle, consts.Year)
	redisCli.Expire(consts.RedisNLinkPrefix + newsLink, consts.Year)
	redisCli.Expire(consts.RedisNIDPrefix + n.newsID, consts.Year)
	return nil
}
