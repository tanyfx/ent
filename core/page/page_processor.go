//author tyf
//date   2017-02-17 11:09
//desc 

package page

type PageProcessor interface {
	ProcessPage(p *Page) error
	Valid() bool
}
