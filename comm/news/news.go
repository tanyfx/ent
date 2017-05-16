//author tyf
//date   2017-02-07 15:47
//desc 

package news

import (
	"regexp"
	"strings"
	"github.com/huichen/sego"
	"github.com/tanyfx/ent/comm"
	"github.com/tanyfx/ent/comm/textutil"
)

const (
	NewsDate string = "datetime"
	NewsTitle string = "title"
	Subtitle string = "subtitle"
	Summary string = "summary"
	NewsAuthor string = "author"
	NewsContent string = "content"
	NewsLink string = "link"
	FormatStr string = "`%s`='%s'"
)

type NewsItem struct {
	//deduper *textutil.Deduper
	doc       textutil.Doc
	newsIndex int
	newsID    string
	postID    string

	Title     string
	Date      string
	Author    string
	Subtitle  string
	Summary   string
	Content   string
	Link      string
	Stars     []comm.StarIDPair
	Imgs      []NewsImg
}

func GenNewsItem(title, date, author, link string) *NewsItem {
	return &NewsItem{
		Title: title,
		Date: date,
		Author: author,
		Link: link,
	}
}

func (p *NewsItem) initDoc(seg *sego.Segmenter) *NewsItem {
	p.doc = textutil.NewDoc(p.Title, "", seg)
	return p
}

func (p *NewsItem) GetNewsID() string {
	return p.newsID
}

func (p *NewsItem) SetNewsID(id string) *NewsItem {
	p.newsID = id
	return p
}

func (p *NewsItem) SetPostID(id string) *NewsItem {
	p.postID = id
	return p
}

func (p *NewsItem) SetTitle(title string) *NewsItem {
	p.Title = title
	return p
}

func (p *NewsItem) SetDate(date string) *NewsItem {
	p.Date = date
	return p
}

func (p *NewsItem) SetAuthor(author string) *NewsItem {
	p.Author = author
	return p
}

func (p *NewsItem) SetSubtitle(subtitle string) *NewsItem {
	p.Subtitle = subtitle
	return p
}

func (p *NewsItem) SetSummary(summary string) *NewsItem {
	p.Summary = summary
	return p
}

func (p *NewsItem) SetContent(content string) *NewsItem {
	p.Content = content
	return p
}

func (p *NewsItem) SetLink(link string) *NewsItem {
	p.Link = link
	return p
}

//func (p *NewsItem) SaveToMySQL(db *sql.DB) error {
//
//}

//func (p *NewsItem) SaveRedisKey(client *redis.Client) error {
//
//}

func (p *NewsItem) Valid() bool {
	return len(p.Content) > 0 && len(p.Title) > 0 && len(p.Link) > 0
}

//check if img in news content is valid
//valid news content: contains img and the first img has been downloaded to local
func (p *NewsItem) ValidContent() bool {
	flag := false
	imgRegexp := regexp.MustCompile(`<img.*?\Wsrc="(.*?)"`)
	match := imgRegexp.FindStringSubmatch(p.Content)
	if len(match) == 2 {
		if !strings.Contains(match[1], "http") {
			flag = true
		}
	}
	return flag
}