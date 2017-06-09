//author tyf
//date   2017-02-10 16:49
//desc

package news

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/tanyfx/ent/comm"
	"github.com/tanyfx/ent/comm/consts"
	"github.com/tanyfx/ent/comm/redisutil"
	"github.com/tanyfx/ent/comm/textutil"
	"github.com/tanyfx/ent/core/page"
	"gopkg.in/redis.v5"
)

type NewsExtractor interface {
	ExtractNews(*page.Page) *NewsItem
}

type ImgReplacer interface {
	ReplaceImgs(n *NewsItem, folderPath, urlPrefix string) (string, []NewsImg)
}

//inherit from page.PageProcessor
//starTagMap map[string]string: star_id -> term_taxonomy_id
type NewsProcessor struct {
	extractor       NewsExtractor
	imgReplacer     ImgReplacer
	db              *sql.DB
	redisCli        *redis.Client
	deduper         *textutil.Deduper
	starTagger      *comm.StarTagger
	folderPath      string
	urlPrefix       string
	starTaxonomyMap map[string]string
}

func GenNewsProcessor(extractor NewsExtractor, imgReplacer ImgReplacer) *NewsProcessor {
	return &NewsProcessor{
		extractor:   extractor,
		imgReplacer: imgReplacer,
	}
}

func InitNewsProcessor(extractor NewsExtractor, imgReplacer ImgReplacer, folderPath, urlPrefix string,
	db *sql.DB, newsRedisCli *redis.Client, newsDeduper *textutil.Deduper, starTagger *comm.StarTagger,
	starTaxonomyMap map[string]string) *NewsProcessor {
	return &NewsProcessor{
		extractor:       extractor,
		imgReplacer:     imgReplacer,
		db:              db,
		redisCli:        newsRedisCli,
		deduper:         newsDeduper,
		starTagger:      starTagger,
		folderPath:      folderPath,
		urlPrefix:       urlPrefix,
		starTaxonomyMap: starTaxonomyMap,
	}
}

func (p *NewsProcessor) ProcessPage(page *page.Page) error {
	n := p.extractor.ExtractNews(page)
	if n == nil || !n.Valid() {
		return errors.New("news not valid")
	}
	//exist, err := redisutil.ExistNewsTitle(p.redisCli, n.Title)
	tmpID, err := redisutil.MGetNewsID(p.redisCli, n.Title, n.Link)
	if err != nil {
		errMsg := fmt.Sprintf("error while find news in redis: %s %s %s", err.Error(), n.Title, n.Link)
		log.Println(errMsg)
	}

	if len(tmpID) > 0 {
		errMsg := fmt.Sprintf("news title or link exists: %s %s", n.Title, n.Link)
		fmt.Println(time.Now().Format(consts.TimeFormat), errMsg)

		return errors.New(errMsg)
	}

	//DEBUG
	//log.Println("do deduplication for:", n.Title, n.Link)

	indexID, ok := p.deduper.PushOne(n.Title, "")
	if !ok {
		return errors.New("find repeated news")
	}
	//DEBUG
	//log.Println("download img for:", n.Title, n.Link)

	n.Content, n.Imgs = p.imgReplacer.ReplaceImgs(n, p.folderPath, p.urlPrefix)
	fmt.Printf("%s %d\timgs downloaded for %s\n", time.Now().Format(consts.TimeFormat), len(n.Imgs), n.Title)
	if !n.ValidContent() {
		errMsg := fmt.Sprintf("first img not downloaded or contains no img, pass: %s %s", n.Title, n.Link)
		fmt.Println(time.Now().Format(consts.TimeFormat), errMsg)
		return errors.New(errMsg)
	}

	meta := page.GetMeta()
	if searchStar, ok := meta[consts.SearchStar]; ok {
		if searchID, ok := meta[consts.SearchID]; ok {
			n.Stars = append(n.Stars, comm.StarIDPair{
				NameCN: searchStar,
				StarID: searchID,
			})
		}
	}

	stars := p.starTagger.TagStar(n.Title)
	for _, pair := range stars {
		if comm.FindStar(pair.NameCN, n.Stars) {
			continue
		}
		n.Stars = append(n.Stars, pair)
	}

	err = saveNews(p.db, p.redisCli, n, p.starTaxonomyMap)
	if err != nil {
		log.Println(err.Error())
		return err
	}

	err = p.deduper.UpdateDocID(indexID, n.postID)
	if err != nil {
		log.Println("error while update deduper's document", err.Error())
	}

	for _, pair := range n.Stars {
		fmt.Printf("%s get star:\t%s\t%s\t%s\t%s\n", time.Now().Format(consts.TimeFormat), n.newsID,
			pair.StarID, pair.NameCN, n.Title)
	}

	return nil
}

func (p *NewsProcessor) Init(extractor NewsExtractor, imgReplacer ImgReplacer, folderPath, urlPrefix string,
	db *sql.DB, redisCli *redis.Client, deduper *textutil.Deduper, starTagger *comm.StarTagger,
	starTaxonomyMap map[string]string) *NewsProcessor {
	if extractor != nil {
		p.extractor = extractor
	}
	if imgReplacer != nil {
		p.imgReplacer = imgReplacer
	}
	if len(folderPath) > 0 {
		p.folderPath = folderPath
	}
	if len(urlPrefix) > 0 {
		p.urlPrefix = urlPrefix
	}
	if db != nil {
		p.db = db
		//p.wpProcessor = genNewsWPProcessor(db)
	}
	if len(starTaxonomyMap) > 0 {
		p.starTaxonomyMap = starTaxonomyMap
	}
	if redisCli != nil {
		p.redisCli = redisCli
	}
	if deduper != nil {
		p.deduper = deduper
	}
	if starTagger != nil {
		p.starTagger = starTagger
	}
	return p
}

func (p *NewsProcessor) SetExtractor(extractor NewsExtractor) *NewsProcessor {
	p.extractor = extractor
	return p
}

func (p *NewsProcessor) SetImgReplacer(imgReplacer ImgReplacer) *NewsProcessor {
	p.imgReplacer = imgReplacer
	return p
}

func (p *NewsProcessor) SetDB(db *sql.DB) *NewsProcessor {
	p.db = db
	return p
}

func (p *NewsProcessor) SetRedis(client *redis.Client) *NewsProcessor {
	p.redisCli = client
	return p
}

func (p *NewsProcessor) SetDeduper(deduper *textutil.Deduper) *NewsProcessor {
	p.deduper = deduper
	return p
}

func (p *NewsProcessor) SetStarTagger(starTagger *comm.StarTagger) *NewsProcessor {
	p.starTagger = starTagger
	return p
}

func (p *NewsProcessor) SetStarTaxonomyMap(starTaxonomyMap map[string]string) *NewsProcessor {
	p.starTaxonomyMap = starTaxonomyMap
	return p
}

func (p *NewsProcessor) SetFolderPath(folderPath string) *NewsProcessor {
	p.folderPath = folderPath
	return p
}

func (p *NewsProcessor) SetURLPrefix(urlPrefix string) *NewsProcessor {
	p.urlPrefix = urlPrefix
	return p
}

func (p *NewsProcessor) Valid() bool {
	if p.extractor == nil || p.imgReplacer == nil || p.db == nil {
		return false
	}
	if p.redisCli == nil || p.deduper == nil || p.starTagger == nil {
		return false
	}
	if len(p.folderPath) == 0 || len(p.urlPrefix) == 0 || len(p.starTaxonomyMap) == 0 {
		return false
	}
	return true
}
