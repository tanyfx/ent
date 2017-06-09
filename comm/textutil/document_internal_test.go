//author tyf
//date   2017-03-07 14:49
//desc

package textutil

import (
	"log"
	"testing"

	"github.com/huichen/sego"
	"github.com/tanyfx/ent/comm/consts"
)

func TestFindSimDoc(t *testing.T) {

	log.SetFlags(log.Lshortfile | log.LstdFlags)

	seg := &sego.Segmenter{}
	seg.LoadDictionary("/home/tyf/go/lib/dict/dictionary.txt")

	s := "《懒人美食日记》郭品超x吴映洁，没整容之前的鬼鬼还是漂亮些"
	docList := []Doc{
		//NewDoc("汤唯为韩国电影青龙奖颁奖 流利英语和韩语 韩国主持都败了", "13", seg),
		//NewDoc("金喜善因女儿丑被疑整容，如今七岁女儿近照曝光，长得越来越漂亮", "12", seg),
		//NewDoc("《最好的我们》你总是说青春从不曾永远，而那时候的我们，就是最好的我们", "50", seg),
		NewDoc(s, "33", seg),
		//NewDoc(s, "90", seg),
		NewDoc("金喜善因女儿丑被疑整容，如今七岁女儿近照曝光，长得越来越漂亮", "60", seg),
		//NewDoc(s, "100", seg),
		//NewDoc(s, "110", seg),
		//NewDoc(s, "120", seg),

		//NewDoc("三星主厨创业摆小吃摊人气爆棚，改良油封鸭打败隔壁西餐厅", "70", seg),
	}

	for _, tmp := range docList {
		log.Printf("text: %s\tdoc id: %s\twords bag length: %d\n", tmp.Text, tmp.DocID, len(tmp.wordsBag))
	}

	log.Println("doc list length:", len(docList))

	doc := NewDoc(s, "2", seg)

	log.Printf("test doc: text: %s\tdoc id: %s\twords bag length: %d\n", doc.Text, doc.DocID, len(doc.wordsBag))

	tmpDoc, found := findSimDoc(doc, docList, consts.ThreadNum, consts.SimScore)
	if found {
		t.Logf("find repeated doc, text: %s\tdoc id: %s\n", tmpDoc.Text, tmpDoc.DocID)
	}
	log.Printf("found: %v tmp doc: text: %s\tdoc id: %s\twords bag length: %d\n",
		found, tmpDoc.Text, tmpDoc.DocID, len(tmpDoc.wordsBag))
}
