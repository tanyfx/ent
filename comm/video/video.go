//author tyf
//date   2017-02-17 22:34
//desc

package video

import "github.com/tanyfx/ent/comm"

const (
	VideoDate  string = "video_date"
	VideoTitle string = "video_title"
	VideoLink  string = "video_link"
)

type VideoItem struct {
	videoIndex int
	videoID    string
	postID     string

	Link  string
	Title string
	Date  string
	Stars []comm.StarIDPair
}

//termTaxonomyID, starNameCN, content string, videos []string
type VideoPost struct {
	termTaxonomyID string
	starNameCN     string
	content        string
	videoList      []string
}

func NewVideoItem(title, date, link string) *VideoItem {
	return &VideoItem{
		Title: title,
		Date:  date,
		Link:  link,
	}
}

func (p *VideoItem) GetVideoID() string {
	return p.videoID
}

func (p *VideoItem) SetVideoID(vid string) *VideoItem {
	p.videoID = vid
	return p
}

func (p *VideoItem) SetPostID(pid string) *VideoItem {
	p.postID = pid
	return p
}

func (p *VideoItem) GetPostID() string {
	return p.postID
}

func (p *VideoItem) SetLink(link string) *VideoItem {
	p.Link = link
	return p
}

func (p *VideoItem) SetTitle(title string) *VideoItem {
	p.Title = title
	return p
}

func (p *VideoItem) SetDate(date string) *VideoItem {
	p.Date = date
	return p
}

func (p *VideoItem) Valid() bool {
	return len(p.Title) > 0 && len(p.Link) > 0
}
