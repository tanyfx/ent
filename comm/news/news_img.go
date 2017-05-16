//author tyf
//date   2017-02-09 19:00
//desc 

package news

import (
	"database/sql"
	"fmt"
	"errors"
)

type NewsImg struct {
	newsID    string
	newsDate  string
	newsTitle string
	imgFolder string
	imgName   string
	imgURL    string
}

func GenNewsImg(newsID, newsDate, newsTitle, imgFolder, imgName, imgURL string) NewsImg {
	return NewsImg{
		newsID: newsID,
		newsDate: newsDate,
		newsTitle: newsTitle,
		imgFolder: imgFolder,
		imgName: imgName,
		imgURL: imgURL,
	}
}

func (p *NewsImg) SetNewsID(id string) *NewsImg {
	p.newsID = id
	return p
}

func (p *NewsImg) SaveToMySQL(db *sql.DB) error {
	if len(p.imgURL) == 0 {
		return errors.New("err save to MySQL: news_img url is nil")
	}

	insertStr := fmt.Sprintf("insert `news_img` set news_id = '%s', datetime = '%s', news_title = '%s', " +
		"img_folder = '%s', img_name = '%s', img_url = '%s';", p.newsID, p.newsDate,
		p.newsTitle, p.imgFolder, p.imgName, p.imgURL)

	_, err := db.Exec(insertStr)
	return err
}
