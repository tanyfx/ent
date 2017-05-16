//author tyf
//date   2017-02-07 11:35
//desc 

package textutil

import (
	"strings"
	"github.com/huichen/sego"
	"log"
	"gopkg.in/redis.v5"
	"sync"
	"errors"
)

type Doc struct {
	indexID  int
	DocID    string
	Text     string
	wordsBag map[string]int
}

type Score struct {
	docID string
	score float32
}

type docCTX struct {
	doc     Doc
	findSim bool
}

func NewDoc(str, docID string, seg *sego.Segmenter) Doc {
	document := Doc{
		DocID: docID,
		Text:  str,
	}
	if seg != nil {
		replacer := NewStopWordsReplacer()
		document.wordsBag = genWordsBag(str, seg, replacer)
	}
	return document
}

func GetRedisTitles(client *redis.Client, titlePrefix, idPrefix string) []Doc {
	result := []Doc{}
	titles, err := client.Keys(titlePrefix + "*").Result()
	if err != nil {
		log.SetFlags(log.Lshortfile | log.LstdFlags)
		log.Println("error while get redis title keys for:", titlePrefix, err.Error())
		return result
	}
	for _, titleStr := range titles {
		tmpTitle := strings.TrimPrefix(titleStr, titlePrefix)

		srcIDStr := client.Get(titleStr).Val()

		srcID := strings.TrimPrefix(srcIDStr, idPrefix)
		tmpDoc := Doc{
			DocID: srcID,
			Text: tmpTitle,
		}
		result = append(result, tmpDoc)
	}
	return result
}

func (p *Doc) Valid() bool {
	return len(p.wordsBag) > 0
}

func (p *Doc) Init(seg *sego.Segmenter) {
	replacer := NewStopWordsReplacer()
	p.wordsBag = genWordsBag(p.Text, seg, replacer)
}

//docs: at least one doc's words-bag length > 0
func isValidDocs(docs []Doc) bool {
	//flag := false
	//if len(docs) == 0 {
	//	flag = true
	//}
	for i := range docs {
		if len(docs[i].wordsBag) > 0 {
			return true
		}
	}
	return false
}

func InitDocs(docs []Doc, limitCount int, seg *sego.Segmenter, replacer *strings.Replacer) ([]Doc, error) {
	if seg == nil {
		return docs, errors.New("ERROR! word segmenter not initialized!")
	}

	if replacer == nil {
		replacer = NewStopWordsReplacer()
	}

	if len(docs) == 0 {
		return docs, nil
	}

	limitChan := make(chan int, limitCount)
	for i := 0; i < limitCount; i++ {
		limitChan <- 1
	}
	defer close(limitChan)
	docChan := make(chan Doc, len(docs))
	defer close(docChan)
	for i, tmpDoc := range docs {
		tmpDoc.indexID = i
		go genWordsBagWorker(tmpDoc, limitChan, docChan, replacer, seg)
	}

	for i := 0; i < len(docs); i++ {
		tmpDoc := <-docChan
		index := tmpDoc.indexID
		docs[index].wordsBag = tmpDoc.wordsBag
	}

	for i := 0; i < limitCount; i++ {
		<-limitChan
	}
	return docs, nil
}

func genWordsBagWorker(doc Doc, threadLimitChan chan int, docChan chan Doc, replacer *strings.Replacer,
seg *sego.Segmenter) {
	<-threadLimitChan
	defer func() {
		threadLimitChan <- 1
	}()
	doc.wordsBag = genWordsBag(doc.Text, seg, replacer)
	docChan <- doc
	return
}

func genWordsBag(s string, seg *sego.Segmenter, replacer *strings.Replacer) map[string]int {
	str := replacer.Replace(s)
	result := map[string]int{}
	segments := seg.Segment([]byte(str))
	tokens := sego.SegmentsToSlice(segments, true)
	for _, token := range tokens {
		result[token] = 1
	}
	return result
}

func NewStopWordsReplacer() *strings.Replacer {
	stopWords := []string{
		",", "?", "、", "。",
		"“", "”", "《", "》",
		"！", "，", "：", "；",
		"？", "的", "了", "在",
		"是", "我", "有", "和",
		"就", "不", "人", "都",
		"一", "一个", "上", "也",
		"很", "到", "说", "要",
		"去", "你", "会", "着",
		"没有", "看", "好", "自己",
		"这", ",", ".", "?",
		"/", "<", ">", "|", "\\",
		"`", ":", ";", "!", "@",
		"#", "$", "%", "%", "^",
		"&", "*", "(", ")", "-",
		"+", "_", "=", "\n", "{",
		"}", "[", "]", "~", " ",
	}
	oldNew := []string{}
	for _, stopWord := range stopWords {
		oldNew = append(oldNew, stopWord, "")
	}
	return strings.NewReplacer(oldNew...)
}

func similarity(a, b map[string]int) float32 {
	count := 0
	small := a
	big := b
	if len(a) > len(b) {
		small = b
		big = a
	}
	if len(big) == 0 {
		return 0
	}
	for word := range small {
		if _, found := big[word]; found {
			count = count + 1
		}
	}
	return float32(count) / float32(len(big))
}

func FindSimDoc(input Doc, docList []Doc, limitCount int, simScore float32) (Doc, bool) {
	if limitCount < 1 {
		limitCount = 1
	}

	found := false
	var res Doc
	wg := &sync.WaitGroup{}
	wg.Add(limitCount)

	sendWG := &sync.WaitGroup{}
	foundChan := make(chan docCTX, limitCount)
	inputChan := make(chan Doc, limitCount)

	defer func() {
		wg.Wait()
		close(foundChan)
	}()

	for i := 0; i < limitCount; i++ {
		go simWorker(input, inputChan, foundChan, wg, simScore, i)
	}

	sCount := 0
	rCount := 0
	go func() {
		sendWG.Add(1)
		defer sendWG.Done()
		for _, doc := range docList {

			if found {
				break
			}
			inputChan <- doc
			sCount++
		}
	}()

	for i := 0; i < len(docList); i++ {
		c := <-foundChan
		rCount++
		if c.findSim {
			res = c.doc
			found = true
			break
		}
	}

	sendWG.Wait()
	close(inputChan)


	for i := rCount; i < sCount; i++ {
		<-foundChan
	}

	return res, found
}

func simWorker(inputDoc Doc, inputChan chan Doc, foundChan chan docCTX, wg *sync.WaitGroup, simScore float32, index int) {
	//log.Printf("start sim worker %d\n", index)
	defer wg.Done()
	for doc := range inputChan {
		score := similarity(inputDoc.wordsBag, doc.wordsBag)
		if score >= simScore {
			foundChan <- docCTX{
				doc: doc,
				findSim: true,
			}
			//log.Printf("sim worker %d find repeated doc\n", index)
		} else {
			foundChan <- docCTX{
				doc: doc,
				findSim: false,
			}
		}
	}
	//log.Printf("sim worker %d exit\n", index)
}
