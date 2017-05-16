//author tyf
//date   2017-02-07 11:35
//desc 

package textutil

import (
	"strings"
	"sync"
	"fmt"
	"errors"
	"github.com/huichen/sego"
	"github.com/tanyfx/ent/comm/consts"
)

type Deduper struct {
	limitCount   int
	simScore     float32
	mu           *sync.Mutex
	seg          *sego.Segmenter
	replacer     *strings.Replacer
	newDocs      []Doc
	repeatedDocs []Doc
	oldDocs      []Doc
}

func NewDeduper(simScore float32, recentDocs, oldDocs []Doc, seg *sego.Segmenter) (*Deduper, error) {
	if seg == nil {
		return nil, errors.New("ERROR! word segmenter not initialized!")
	}
	limitCount := consts.ThreadNum
	deduper := &Deduper{
		limitCount:	limitCount,
		simScore:	simScore,
		seg:		seg,
		replacer:	NewStopWordsReplacer(),
	}
	deduper.mu	= &sync.Mutex{}
	deduper.newDocs, _ = InitDocs(recentDocs, limitCount, seg, deduper.replacer)
	deduper.oldDocs, _ = InitDocs(oldDocs, limitCount, seg, deduper.replacer)
	return deduper, nil
}

func (p *Deduper) addRepeatedDoc(doc Doc) {
	p.repeatedDocs = append(p.repeatedDocs, doc)
}

func (p *Deduper) PushOne(str, docID string) (int, bool) {
	tmpDoc := NewDoc(str, docID, p.seg)

	return p.PushDoc(tmpDoc)
}

func (p *Deduper) PushDoc(doc Doc) (int, bool) {

	//DEBUG
	//log.Printf("tmpdoc status: text: %s\twords bag length: %d\n", doc.Text, len(doc.wordsBag))
	//a := []string{}
	//for k := range doc.wordsBag {
	//	a = append(a, k)
	//}
	//log.Println("words bag:", strings.Join(a, " "))

	//DEBUG
	//log.Println("deduper doc length:", len(p.repeatedDocs), len(p.newDocs), len(p.oldDocs))

	p.mu.Lock()
	defer p.mu.Unlock()

	if len(doc.wordsBag) == 0 {
		doc.wordsBag = genWordsBag(doc.Text, p.seg, p.replacer)
	}

	tmp, found := FindSimDoc(doc, p.repeatedDocs, p.limitCount, p.simScore)
	if found {
		fmt.Printf("%s find repeated, origin: id: %s doc: %s\n", consts.TimeFormat, tmp.DocID, tmp.Text)
		return -1, false
	}

	tmp, found = FindSimDoc(doc, p.newDocs, p.limitCount, p.simScore)
	if found {
		p.addRepeatedDoc(tmp)
		fmt.Printf("%s find repeated, origin: id: %s doc: %s\n", consts.TimeFormat, tmp.DocID, tmp.Text)
		return -1, false
	}

	tmp, found = FindSimDoc(doc, p.oldDocs, p.limitCount, p.simScore)
	if found {
		p.addRepeatedDoc(tmp)
		fmt.Printf("%s find repeated, origin: id: %s doc: %s\n", consts.TimeFormat, tmp.DocID, tmp.Text)
		return -1, false
	}

	n := len(p.newDocs)
	p.newDocs = append(p.newDocs, doc)
	return n, true
}

func (p *Deduper) UpdateDocID(index int, id string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if index >= len(p.newDocs) {
		s := fmt.Sprintf("index out of bound, index: %d, length: %d", index, len(p.newDocs))
		return errors.New(s)
	}
	p.newDocs[index].DocID = id
	return nil
}

func (p *Deduper) FindDoc(doc Doc) (Doc, bool) {
	found := false
	var res Doc
	p.mu.Lock()
	defer p.mu.Unlock()

	res, found = FindSimDoc(doc, p.newDocs, p.limitCount, p.simScore)
	if found {
		return res, found
	}

	res, found = FindSimDoc(doc, p.newDocs, p.limitCount, p.simScore)
	if found {
		p.addRepeatedDoc(res)
		return res, found
	}
	res, found = FindSimDoc(doc, p.oldDocs, p.limitCount, p.simScore)
	if found {
		p.addRepeatedDoc(res)
	}
	return res, found
	//p.recentDocs = append(p.recentDocs, doc)
}