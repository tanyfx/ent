//author tyf
//date   2017-02-09 18:20
//desc 

package wordpressutil

import (
	"strings"
	"bufio"
	"database/sql"
	"log"
	"strconv"
	"errors"
	"bytes"
	"text/template"
	"github.com/tanyfx/ent/comm/consts"
)

//postDate, postContent, postTitle, postName, postExcerpt, commentStatus string,
//postCateID int, termTaxonomyIDList []string
type Post struct {
	PostDate           string
	PostContent        string
	PostTitle          string
	PostName           string
	PostExcerpt        string
	CommentStatus      string
	PostCateID         int
	TermTaxonomyIDList []string
}

//star_id -> term_taxonomy_id
func GetStarTaxonomyMap(db *sql.DB) (map[string]string, error) {
	starTaxonomyMap := map[string]string{}

	queryStr := "select star_id, term_taxonomy_id from `star_name`"
	rows, err := db.Query(queryStr)
	if err != nil {
		return starTaxonomyMap, err
	}

	for rows.Next() {
		var starID, tagID sql.NullString
		err = rows.Scan(&starID, &tagID)
		if err != nil {
			log.Println(err.Error())
			continue
		}
		if tagID.Valid {
			//fmt.Println(starID.String, tagID.String)
			starTaxonomyMap[starID.String] = tagID.String
		}
	}
	return starTaxonomyMap, nil
}

//update one post in wp_posts
func UpdatePostContent(db *sql.DB, post *Post, postID string) error {
	updateStr := "update `wp_posts` set post_date = '" + post.PostDate + "', post_content = '" +
		post.PostContent + "', post_title = '" + post.PostTitle + "', post_name = '" + post.PostName +
		"', comment_status = '" + post.CommentStatus + "' where ID = '" + postID + "';"
	_, err := db.Exec(updateStr)
	if err != nil {
		return errors.New("error while update post: " + postID + " " + err.Error())
	}
	return nil
}

//save one post to the wp_posts
//func SavePost(db *sql.DB, postDate, postContent, postTitle, postName, postExcerpt, commentStatus string,
//postCateID int, termTaxonomyIDList []string) (postID int64, err error) {
func SavePost(db *sql.DB, post *Post) (postID int64, err error) {
	postID = -1
	insertStr := "insert into `wp_posts` set post_date = '" + post.PostDate + "', post_content = '" +
		post.PostContent + "', post_title = '" + post.PostTitle + "', post_name = '" + post.PostName +
		"', comment_status = '" + post.CommentStatus + "'"

	insertStr = insertStr + ", post_excerpt = '" + post.PostExcerpt + "';"

	//fmt.Print("\n", insertStr, "\n")

	//if len(post.PostExcerpt) > 0 {
	//	insertStr = insertStr + ", post_excerpt = '" + post.PostExcerpt + "';"
	//} else {
	//	insertStr = insertStr + ";"
	//}
	res, err := db.Exec(insertStr)
	if err != nil {
		return postID, err
	}
	postID, err = res.LastInsertId()
	if err != nil {
		return postID, err
	}
	//guid := "http://blackmomo.cn?p=" + strconv.FormatInt(postID, 10)
	guid := consts.HostPrefix + "?p=" + strconv.FormatInt(postID, 10)
	updateStr := "update `wp_posts` set guid = ? where ID = ?;"
	if _, err = db.Exec(updateStr, guid, strconv.FormatInt(postID, 10)); err != nil {
		return postID, err
	}

	//add category for post
	insertPostCateStr := "insert into `wp_term_relationships` set object_id = ?, term_taxonomy_id = ?"
	if _, err = db.Exec(insertPostCateStr, strconv.FormatInt(postID, 10), post.PostCateID); err != nil {
		return postID, err
	}

	updateCateCountStr := "update `wp_term_taxonomy` set count = count + 1 where term_taxonomy_id = ?"
	if _, err = db.Exec(updateCateCountStr, post.PostCateID); err != nil {
		return postID, err
	}

	for _, termTaxonomyID := range post.TermTaxonomyIDList {
		//update star tag on post content
		insertPostTagStr := "insert into `wp_term_relationships` set object_id = ?, term_taxonomy_id = ?"
		if _, err = db.Exec(insertPostTagStr, strconv.FormatInt(postID, 10), termTaxonomyID); err != nil {
			return postID, err
		}
		//update count of star tag
		updateTermStr := "update `wp_term_taxonomy` set count = count + 1 where term_taxonomy_id = ?"
		if _, err = db.Exec(updateTermStr, termTaxonomyID); err != nil {
			return postID, err
		}
	}
	return postID, nil
}

//func GenProfileTemplate(fields []string) (string, error) {
func GenProfileTemplate(imgSrc, nameCN, birthday, height, weight, nationality,
constellation, intro string) (string, error) {
	profileTemplate := `<ul class="star_profile">
    <li class="star_img"><img src="{{.ImgSrc}}" alt=""></li>
    <li>姓名：{{.NameCN}}</li>
                        <li>生日：{{.Birthday}}</li>
                        <li>身高：{{.Height}}</li>
                        <li>体重：{{.Weight}}</li>
                        <li>国籍：{{.Nationality}}</li>
                        <li>星座：{{.Constellation}}</li>
                        <li class="star_info">简介：<br>
                                {{.IntroFirst}}
                        </li>

			{{if .IntroLast}}
                        <li class='star_info'>
                            <div class='info_hide'>{{.IntroLast}}</div>
                             <button class='printMore'>更多</button>
                        </li>
                        {{end}}
                    </ul>`
	//intro := intro
	m := strings.Split(intro, "\n")
	introFirst := ""
	introLast := ""
	if len(m) >= 2 {
		introFirst = strings.Join(m[:2], "<br>")
		introLast = strings.Join(m[2:], "<br>")
	} else {
		introFirst = strings.Join(m, "<br>")
	}
	profileData := struct {
		ImgSrc        string
		NameCN        string
		Birthday      string
		Height        string
		Weight        string
		Nationality   string
		Constellation string
		IntroFirst    string
		IntroLast     string
	}{
		ImgSrc: imgSrc,
		NameCN: nameCN,
		Birthday: birthday,
		Height: height,
		Weight: weight,
		Nationality: nationality,
		Constellation: constellation,
		IntroFirst: introFirst,
		IntroLast: introLast,
	}

	b := bytes.NewBuffer(make([]byte, 0))
	bw := bufio.NewWriter(b)
	//buffer := bytes.NewBufferString(content)
	t, err := template.New("webpage").Parse(profileTemplate)
	if err != nil {
		return b.String(), err
	}
	if err = t.Execute(bw, profileData); err != nil {
		return b.String(), err
	}
	if err = bw.Flush(); err != nil {
		return b.String(), err
	}
	return b.String(), nil
}

