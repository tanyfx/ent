//author tyf
//date   2017-02-19 11:41
//desc

package textutil

import (
	"testing"
	"github.com/huichen/sego"
	"github.com/tanyfx/ent/comm/consts"
	"log"
)

func TestFindSimDoc(t *testing.T) {
	seg := &sego.Segmenter{}
	seg.LoadDictionary("/home/tyf/go/lib/dict/dictionary.txt")

	s := "这是测试案例"
	docList := []Doc{
		NewDoc(s, "13", seg),
		//NewDoc(s, "12", seg),
	}

	for _, tmp := range docList {
		log.Printf("text: %s\tdoc id: %s\twords bag length: %d\n", tmp.Text, tmp.DocID, len(tmp.wordsBag))
	}

	log.Println("doc list length:", len(docList))

	doc := NewDoc(s, "2", seg)

	log.Printf("test doc: text: %s\tdoc id: %s\twords bag length: %d\n", doc.Text, doc.DocID, len(doc.wordsBag))

	tmpDoc, found := FindSimDoc(doc, docList, consts.ThreadNum, consts.SimScore)
	if found {
		t.Logf("find repeated doc, text: %s\tdoc id: %s\n", tmpDoc.Text, tmpDoc.DocID)
	}
	log.Printf("found: %v tmp doc: text: %s\tdoc id: %s\twords bag length: %d\n",
		found, tmpDoc.Text, tmpDoc.DocID, len(tmpDoc.wordsBag))
}
