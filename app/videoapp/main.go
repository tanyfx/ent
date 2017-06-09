//author tyf
//date   2017-02-21 11:20
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
	"gopkg.in/redis.v5"

	"github.com/tanyfx/ent/app/videoapp/qqvideo"
	"github.com/tanyfx/ent/comm"
	"github.com/tanyfx/ent/comm/consts"
	"github.com/tanyfx/ent/comm/textutil"
	"github.com/tanyfx/ent/comm/video"
	"github.com/tanyfx/ent/comm/wordpressutil"
	"github.com/tanyfx/ent/core/download"
	"github.com/tanyfx/ent/core/index"
	"github.com/tanyfx/ent/core/item"
)

var confFile = flag.String("c", "db.conf", "db conf file")
var dictFile = flag.String("d", "dict/dictionary.txt", "dictionary for sego")
var starFile = flag.String("i", "stars/c1.txt", "stars name for video search")
var search = flag.Bool("s", false, "-s for search video")
var update = flag.Bool("u", false, "-u for update video")

var db *sql.DB
var seg *sego.Segmenter
var videoRedisCli *redis.Client
var starTagger *comm.StarTagger //extract star tag from title
var videoDeduper *textutil.Deduper
var starTaxonomyMap map[string]string      //for saving to wordpress: star_id -> term_taxonomy_id
var starPairMap map[string]comm.StarIDPair //for video search
var itemDownloader *video.VideoDownloader

var updateProducers = []video.VideoUpdateProducer{
	&qqvideo.UpdateQQVideoProducer{},
}

var searchProducers = []video.VideoSearchProducer{
	&qqvideo.SearchQQVideoProducer{},
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
	//defer db.Close()

	seg = &sego.Segmenter{}
	seg.LoadDictionary(*dictFile)

	videoRedisCli = redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: redisPasswd,
		DB:       consts.RedisVideoDB,
	})

	if err = videoRedisCli.Ping().Err(); err != nil {
		errMsg := fmt.Sprintln("error ping video redis db, exit", err.Error())
		return errors.New(errMsg)
	}
	//defer videoRedisCli.Close()

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

	nickNameMap, err := comm.GetNickname(db)
	if err != nil {
		return err
	}

	starIDMap, idStarMap, err := comm.GetRedisStarID(starRedisCli)
	if err != nil {
		errMsg := fmt.Sprintln("error while get star name->id map", err.Error())
		return errors.New(errMsg)
	}
	fmt.Println(time.Now().Format(consts.TimeFormat), "star_id map length:", len(starIDMap))
	fmt.Println(time.Now().Format(consts.TimeFormat), "nickname map length:", len(nickNameMap))

	//starTagger = comm.NewStarTagger(starIDMap)
	starTagger = comm.NewStarNicknameTagger(idStarMap, nickNameMap)

	videoDeduper, err = video.GenVideoDeduper(videoRedisCli, seg)
	if err != nil {
		errMsg := fmt.Sprintln("error while generate video deduper", err.Error())
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

	itemDownloader = video.GenVideoDownloader(videoRedisCli)

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
		if videoRedisCli != nil {
			videoRedisCli.Close()
		}
		log.Fatalln(err.Error())
	}

	//reader.ReadLine()

	defer func() {
		db.Close()
		videoRedisCli.Close()
	}()

	simpleChan := make(chan *video.SimpleVideoCTX, 10)
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
			videoProcessor := video.InitVideoProcessor(c.Extractor, db, videoRedisCli, videoDeduper,
				starTagger, starTaxonomyMap)
			if !videoProcessor.Valid() {
				log.Fatalln("video processor not valid")
			}
			if c.ItemDownloader == nil {
				c.ItemDownloader = itemDownloader
			}
			indexCTX := index.NewIndexCTX(c.Req, &download.HttpDownloader{}, c.ItemDownloader,
				c.IndexProcessor, videoProcessor)
			ctxList := indexCTX.ExtractItemCTX()

			//DEBUG
			//log.Println("[DEBUG]", indexCTX.GetRequest().URL.String(), "context length:", len(ctxList))

			fmt.Printf("%s\t%d\tget %s context length: %d\n",
				time.Now().Format(consts.TimeFormat), count,
				c.IndexProcessor.GetIndexName(), len(ctxList))

			for _, itemCTX := range ctxList {

				//fmt.Println("video meta map length:", len(itemCTX.Meta))
				//for k, v := range itemCTX.Meta {
				//	fmt.Println(k, " = ", v)
				//}

				itemChan <- itemCTX
			}

			//DEBUG
			log.Println("[DEBUG]", counter.Count(), "video added to wordpress up to now")

		}

		log.Println("simple index receive worker stopped")
		simpleWG.Done()
	}()

	indexWG := &sync.WaitGroup{}

	if *update {
		indexWG.Add(len(updateProducers))
		for i, tmp := range updateProducers {
			log.Println("run update index producer:", i)
			go func(c int, producer video.VideoUpdateProducer) {
				producer.Produce(simpleChan, videoRedisCli)
				log.Println("update index producer", c, "stopped")
				indexWG.Done()
			}(i, tmp)
		}
	}

	if *search {
		indexWG.Add(len(searchProducers))
		for i, tmp := range searchProducers {
			log.Println("run search index producer:", i)
			go func(c int, producer video.VideoSearchProducer) {
				producer.Produce(simpleChan, videoRedisCli, starPairMap)
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

	fmt.Printf("%s get video done, %d video added, %dm %ds used\n", time.Now().Format(consts.TimeFormat),
		counter.Count(), minutes, seconds)
}
