//author tyf
//date   2017-02-15 18:44
//desc 

package toutiaonews

import (
	"gopkg.in/xmlpath.v2"
	"strings"
	"log"
	"github.com/tanyfx/ent/core/item"
	"github.com/tanyfx/ent/core/page"
)

type ToutiaoSearchIndexProcessor struct {
}

//TODO
func (p *ToutiaoSearchIndexProcessor) ProcessPage(indexPage *page.Page) []*item.ItemCTX {
	resultList := []*item.ItemCTX{}
	root, err := xmlpath.ParseHTML(strings.NewReader(indexPage.GetBody()))
	if err != nil {
		log.Println("error while parse toutiao index page:", err.Error())
		return resultList
	}
}