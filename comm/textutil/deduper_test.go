//author tyf
//date   2017-02-19 10:41
//desc

package textutil

import (
	"testing"
	"github.com/huichen/sego"
	"github.com/tanyfx/ent/comm/consts"
	"log"
)

func TestDeduper_PushOne(t *testing.T) {
	log.SetFlags(log.Lshortfile | log.LstdFlags)
	seg := &sego.Segmenter{}
	seg.LoadDictionary("/home/tyf/go/lib/dict/dictionary.txt")
	//a := NewDoc("这是测试案例", "13", seg)
	s := "这是测试案例"

	deduper, err := NewDeduper(consts.SimScore, []Doc{}, []Doc{}, seg)
	if err != nil {
		log.Println(err.Error())
	}

	deduper.PushOne(s, "")
	for i := 0; i < consts.ThreadNum; i++ {
		c := i
		//go func(c int) {
		log.Printf("worker %d start\n", c)
		//time.Sleep(time.Second)
		id, ok := deduper.PushOne(s, "")
		if ok {
			t.Failed()
		}
		log.Printf("worker %d push result: %d %v\n", c, id, ok)
		//}(i)
	}
}
