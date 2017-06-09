//author tyf
//date   2017-02-05 17:01
//desc

package download

import (
	"io/ioutil"
	"log"
	"net/http"
	"time"

	"github.com/tanyfx/ent/core/page"
)

type Downloader interface {
	Download(req *http.Request) *page.Page
}

type HttpDownloader struct {
}

func (p *HttpDownloader) Download(req *http.Request) *page.Page {

	respPage := page.NewPage(req)
	resp, err := downloadThrice(req)
	if err != nil {
		respPage.SetStatus(false, err.Error())
		return respPage
	}
	defer resp.Body.Close()
	//respPage.AddCookies(resp.Cookies())
	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		respPage.SetStatus(false, err.Error())
		return respPage
	}
	respPage.SetBody(string(content)).SetStatus(true, "")
	return respPage
}

func downloadThrice(req *http.Request) (*http.Response, error) {
	client := &http.Client{
		Timeout: time.Duration(5 * time.Second),
	}
	var err error
	var resp *http.Response
	for i := 0; i < 3; i++ {
		if i > 0 {
			log.Println("error while query url", req.URL.String(), "try again:", err.Error())
		}
		resp, err = client.Do(req)
		if err == nil {
			break
		}
	}
	if err != nil {
		return resp, err
	}

	return resp, nil
}
