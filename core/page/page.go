//author tyf
//date   2017-02-08 15:55
//desc

package page

import "net/http"

type Page struct {
	isSucc   bool
	errorMsg string
	body     string
	header   http.Header
	req      *http.Request
	cookies  []*http.Cookie
	meta     map[string]string
}

func NewPage(req *http.Request) *Page {
	return &Page{
		req:     req,
		isSucc:  false,
		header:  req.Header,
		cookies: req.Cookies(),
		meta:    map[string]string{},
	}
}

func (p *Page) GetRequest() *http.Request {
	return p.req
}

func (p *Page) SetRequest(req *http.Request) *Page {
	p.req = req
	return p
}

func (p *Page) SetHeader(header http.Header) *Page {
	p.header = header
	return p
}

func (p *Page) GetHeader() http.Header {
	return p.header
}

func (p *Page) SetCookies(cookies []*http.Cookie) *Page {
	p.cookies = cookies
	return p
}

func (p *Page) AddCookies(cookies []*http.Cookie) *Page {
	p.cookies = append(p.cookies, cookies...)
	return p
}

func (p *Page) GetCookies() []*http.Cookie {
	return p.cookies
}

func (p *Page) SetMetaMap(meta map[string]string) *Page {
	for k, v := range meta {
		p.meta[k] = v
	}
	return p
}

func (p *Page) GetMeta() map[string]string {
	return p.meta
}

func (p *Page) IsSucc() bool {
	return p.isSucc
}

func (p *Page) ErrorMsg() string {
	return p.errorMsg
}

func (p *Page) SetStatus(isSucc bool, errMsg string) *Page {
	p.isSucc = isSucc
	p.errorMsg = errMsg
	return p
}

func (p *Page) SetBody(body string) *Page {
	p.body = body
	return p
}

func (p *Page) GetBody() string {
	return p.body
}
