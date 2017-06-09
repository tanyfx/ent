//author tyf
//date   2017-02-15 18:44
//desc

package toutiaonews

import (
	"fmt"
	"log"
	"strings"

	"github.com/tanyfx/ent/comm"
	"github.com/tanyfx/ent/core/item"
	"github.com/tanyfx/ent/core/page"
	"gopkg.in/xmlpath.v2"
)

type ToutiaoSearchIndexProcessor struct {
	searchIndex int
	total       int
	starPair    comm.StarIDPair
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

func (p *ToutiaoSearchIndexProcessor) GetIndexName() string {
	prefix := fmt.Sprintf("(%d/%d)", p.searchIndex, p.total)
	name := fmt.Sprintf("search qq %s %s\t%s\tnews", prefix, p.starPair.NameCN, p.starPair.StarID)
	return name
}
