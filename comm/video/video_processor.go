//author tyf
//date   2017-02-17 22:44
//desc

package video

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

type VideoExtractor interface {
	ExtractVideo(*page.Page) *VideoItem
}

//inherit from page.PageProcessor
//starTagMap map[string]string: star_id -> term_taxonomy_id
type VideoProcessor struct {
	extractor       VideoExtractor
	db              *sql.DB
	redisCli        *redis.Client
	deduper         *textutil.Deduper
	starTagger      *comm.StarTagger
	starTaxonomyMap map[string]string
}

func GenVideoProcessor(extractor VideoExtractor) *VideoProcessor {
	return &VideoProcessor{
		extractor: extractor,
	}
}

func InitVideoProcessor(extractor VideoExtractor, db *sql.DB, redisCli *redis.Client, deduper *textutil.Deduper,
	starTagger *comm.StarTagger, starTaxonomyMap map[string]string) *VideoProcessor {
	return &VideoProcessor{
		extractor:       extractor,
		db:              db,
		redisCli:        redisCli,
		deduper:         deduper,
		starTagger:      starTagger,
		starTaxonomyMap: starTaxonomyMap,
	}
}

func (p *VideoProcessor) ProcessPage(vItemPage *page.Page) error {
	v := p.extractor.ExtractVideo(vItemPage)
	if !v.Valid() {
		return errors.New("video not valid")
	}
	tmpID, err := redisutil.MGetVideoID(p.redisCli, v.Title, v.Link)
	if err != nil {
		errMsg := fmt.Sprintf("error while find video in redis: %s", err.Error())
		return errors.New(errMsg)
	}
	if len(tmpID) > 0 {
		errMsg := fmt.Sprintf("video title or link exists: %s %s", v.Title, v.Link)
		fmt.Println(time.Now().Format(consts.TimeFormat), errMsg)
		return errors.New(errMsg)
	}
	indexID, ok := p.deduper.PushOne(v.Title, "")
	if !ok {
		return errors.New("find repeated video")
	}

	metaMap := vItemPage.GetMeta()
	if searchName, ok := metaMap[consts.SearchStar]; ok {
		if searchID, ok := metaMap[consts.SearchID]; ok {
			v.Stars = append(v.Stars, comm.StarIDPair{
				NameCN: searchName,
				StarID: searchID,
			})
		}
	}

	stars := p.starTagger.TagStar(v.Title)
	for _, pair := range stars {
		if comm.FindStar(pair.NameCN, v.Stars) {
			continue
		}
		v.Stars = append(v.Stars, pair)
	}

	err = saveVideo(p.db, p.redisCli, v, p.starTaxonomyMap)
	if err != nil {
		log.Println(err.Error())
		return err
	}

	err = p.deduper.UpdateDocID(indexID, v.videoID)
	if err != nil {
		log.Println("error while update deduper's document", err.Error())
	}

	for _, pair := range v.Stars {
		fmt.Printf("%s get star:\t%s\t%s\t%s\t%s\n", time.Now().Format(consts.TimeFormat), v.videoID,
			pair.StarID, pair.NameCN, v.Title)
	}

	return nil
}

func (p *VideoProcessor) Init(extractor VideoExtractor, db *sql.DB, redisCli *redis.Client, deduper *textutil.Deduper,
	starTagger *comm.StarTagger, starTaxonomyMap map[string]string) *VideoProcessor {
	if extractor != nil {
		p.extractor = extractor
	}
	if db != nil {
		p.db = db
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
	if starTaxonomyMap != nil {
		p.starTaxonomyMap = starTaxonomyMap
	}
	return p
}

func (p *VideoProcessor) SetExtractor(extractor VideoExtractor) *VideoProcessor {
	p.extractor = extractor
	return p
}

func (p *VideoProcessor) SetDB(db *sql.DB) *VideoProcessor {
	p.db = db
	return p
}

func (p *VideoProcessor) SetRedis(redisCli *redis.Client) *VideoProcessor {
	p.redisCli = redisCli
	return p
}

func (p *VideoProcessor) SetDeduper(deduper *textutil.Deduper) *VideoProcessor {
	p.deduper = deduper
	return p
}

func (p *VideoProcessor) SetStarTagger(starTagger *comm.StarTagger) *VideoProcessor {
	p.starTagger = starTagger
	return p
}

//starTaxonomyMap map[star_id] = term_taxonomy_id
func (p *VideoProcessor) SetStarTaxonomyMap(starTaxonomyMap map[string]string) *VideoProcessor {
	p.starTaxonomyMap = starTaxonomyMap
	return p
}

func (p *VideoProcessor) Valid() bool {
	if p.extractor == nil || p.db == nil {
		return false
	}
	if p.redisCli == nil || p.deduper == nil {
		return false
	}
	if p.starTagger == nil || p.starTaxonomyMap == nil {
		return false
	}
	return true
}
