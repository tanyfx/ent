//author tyf
//date   2017-02-13 23:58
//desc

package main

import (
	"database/sql"
	"errors"
	"flag"
	"fmt"
	"log"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql"
	"github.com/huichen/sego"
	"github.com/tanyfx/ent/app/newsapp/qqnews"
	"github.com/tanyfx/ent/app/newsapp/sinanews"
	"github.com/tanyfx/ent/comm"
	"github.com/tanyfx/ent/comm/consts"
	"github.com/tanyfx/ent/comm/news"
	"github.com/tanyfx/ent/comm/textutil"
	"github.com/tanyfx/ent/comm/wordpressutil"
	"github.com/tanyfx/ent/core/download"
	"github.com/tanyfx/ent/core/index"
	"github.com/tanyfx/ent/core/item"
	"gopkg.in/redis.v5"
)

var confFile = flag.String("c", "db.conf", "db conf file")
var dictFile = flag.String("d", "dict/dictionary.txt", "dictionary for sego")
var starFile = flag.String("i", "stars/c1.txt", "stars name for news search")
var search = flag.Bool("s", false, "-s for search news")
var update = flag.Bool("u", false, "-u for update news")

var db *sql.DB
var seg *sego.Segmenter
var newsRedisCli *redis.Client
var folderPath string
var urlPrefix string
var starTagger *comm.StarTagger //extract star tag from title
var newsDeduper *textutil.Deduper
var starTaxonomyMap map[string]string      //for saving to wordpress: star_id -> term_taxonomy_id
var starPairMap map[string]comm.StarIDPair //for news search
var itemDownloader *news.NewsDownloader

var updateProducers = []news.NewsUpdateProducer{
	&sinanews.SinaUpdateProducer{},
	//&kuaibaonews.KuaibaoUpdateProducer{},
	//&qqnews.QQ3gIndexProducer{},
	&qqnews.XWIndexProducer{},
}

var searchProducers = []news.NewsSearchProducer{
	//&kuaibaonews.KuaibaoSearchProducer{},
	&qqnews.QQSearchIndexProducer{},
}

func Init() error {
	dbHandler, redisAddr, redisPasswd, err := comm.ReadConf(*confFile)
	if err != nil {
		log.Println("failed reading db config file, use default config", err.Error())
		dbHandler = consts.WordpressDBHandler
		redisAddr = consts.RedisAddr
		redisPasswd = consts.RedisPasswd
	}

	db, err = sql.Open("mysql", dbHandler)
	if err != nil {
		errMsg := fmt.Sprintln("error while open mysql, exit", err.Error())
		return errors.New(errMsg)
	}

	seg = &sego.Segmenter{}
	seg.LoadDictionary(*dictFile)

	newsRedisCli = redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: redisPasswd,
		DB:       consts.RedisNewsDB,
	})

	if err = newsRedisCli.Ping().Err(); err != nil {
		errMsg := fmt.Sprintln("error ping news redis db, exit", err.Error())
		return errors.New(errMsg)
	}

	starRedisCli := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: redisPasswd,
		DB:       consts.RedisStarDB,
	})

	if err = starRedisCli.Ping().Err(); err != nil {
		errMsg := fmt.Sprintln("error ping star redis db, exit", err.Error())
		return errors.New(errMsg)
	}
	defer starRedisCli.Close()

	folderPath, urlPrefix, err = news.GenImgFolderPrefix()
	if err != nil {
		return err
	}

	nickNameMap, err := comm.GetNickname(db)
	if err != nil {
		return err
	}

	starIDMap, idStarMap, err := comm.GetRedisStarID(starRedisCli)
	//starIDMap, _, err := comm.GetRedisStarID(starRedisCli)
	if err != nil {
		errMsg := fmt.Sprintln("error while get star name->id map", err.Error())
		return errors.New(errMsg)
	}
	fmt.Println(time.Now().Format(consts.TimeFormat), "star_id map length:", len(starIDMap))
	fmt.Println(time.Now().Format(consts.TimeFormat), "nickname map length:", len(nickNameMap))

	//starTagger = comm.NewStarTagger(starIDMap)
	starTagger = comm.NewStarNicknameTagger(idStarMap, nickNameMap)

	newsDeduper, err = news.GenNewsDeduper(newsRedisCli, seg)
	if err != nil {
		errMsg := fmt.Sprintln("error while generate news deduper", err.Error())
		return errors.New(errMsg)
	}

	starTaxonomyMap, err = wordpressutil.GetStarTaxonomyMap(db)
	if err != nil {
		errMsg := fmt.Sprintln("error while get wordpress star tag", err.Error())
		return errors.New(errMsg)
	}

	if *search {
		starPairMap = comm.GetSearchStarList(*starFile, starIDMap)
	}

	itemDownloader = news.GenNewsDownloader(newsRedisCli)

	return nil
}

func main() {
	a := time.Now().Unix()
	flag.Parse()
	log.SetFlags(log.Lshortfile | log.LstdFlags)

	if err := Init(); err != nil {
		if db != nil {
			db.Close()
		}
		if newsRedisCli != nil {
			newsRedisCli.Close()
		}
		log.Fatalln(err.Error())
	}

	defer func() {
		db.Close()
		newsRedisCli.Close()
	}()

	simpleChan := make(chan *news.SimpleCTX, 10)
	itemChan := make(chan *item.ItemCTX, 10)
	itemWG := &sync.WaitGroup{}
	counter := comm.NewCounter()

	itemWG.Add(consts.ThreadNum)
	for i := 0; i < consts.ThreadNum; i++ {
		go item.ItemWorker(itemChan, counter, itemWG, i)
	}

	simpleWG := &sync.WaitGroup{}
	simpleWG.Add(1)
	go func() {
		log.Println("run simple index receive worker")
		count := 0
		for c := range simpleChan {
			count++
			newsProcessor := news.InitNewsProcessor(c.Extractor, c.ImgReplacer, folderPath, urlPrefix,
				db, newsRedisCli, newsDeduper, starTagger, starTaxonomyMap)

			if !newsProcessor.Valid() {
				log.Fatalln("news processor not valid")
			}
			indexCTX := index.NewIndexCTX(c.Req, &download.HttpDownloader{}, itemDownloader,
				c.IndexProcessor, newsProcessor)

			itemCTXList := indexCTX.ExtractItemCTX()

			fmt.Printf("%s\t%d\tget %s context length: %d\n",
				time.Now().Format(consts.TimeFormat), count,
				c.IndexProcessor.GetIndexName(), len(itemCTXList))

			for _, itemCTX := range itemCTXList {

				//fmt.Println("news meta map length:", len(itemCTX.Meta))
				//for k, v := range itemCTX.Meta {
				//	fmt.Println(k, " = ", v)
				//}

				itemChan <- itemCTX
			}

			//DEBUG
			log.Println("[DEBUG]", counter.Count(), "news added to wordpress up to now")
		}

		log.Println("simple index receive worker stopped")
		simpleWG.Done()
	}()

	indexWG := &sync.WaitGroup{}

	if *update {
		indexWG.Add(len(updateProducers))
		for i, tmp := range updateProducers {
			log.Println("run update index producer:", i)
			go func(c int, producer news.NewsUpdateProducer) {
				producer.Produce(simpleChan)
				log.Println("update index producer", c, "stopped")
				indexWG.Done()
			}(i, tmp)
		}
	}

	if *search {
		indexWG.Add(len(searchProducers))
		for i, tmp := range searchProducers {
			log.Println("run search index producer:", i)
			go func(c int, producer news.NewsSearchProducer) {
				producer.Produce(simpleChan, starPairMap)
				log.Println("search index producer", c, "stopped")
				indexWG.Done()
			}(i, tmp)
		}
	}

	indexWG.Wait()
	close(simpleChan)
	simpleWG.Wait()
	close(itemChan)
	itemWG.Wait()

	b := time.Now().Unix()
	duration := b - a
	minutes := duration / 60
	seconds := duration % 60

	fmt.Printf("%s get news done, %d news added, %dm %ds used\n", time.Now().Format(consts.TimeFormat),
		counter.Count(), minutes, seconds)
}
