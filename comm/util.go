//author tyf
//date   2017-02-09 18:06
//desc 

package comm

import (
	"time"
	"strings"
	"gopkg.in/redis.v5"
	"errors"
	"database/sql"
	"strconv"
	"net/http"
	"log"
	"regexp"
	"github.com/jmcvetta/randutil"
	"io/ioutil"
	"os"
	"bufio"
	"fmt"
	"github.com/tanyfx/ent/comm/consts"
)

func ReadConf(filename string) (dbHandler, redisAddr, redisPasswd string, err error) {
	user := "root"
	passwd := ""
	host := "localhost"
	port := "3306"
	dbName := "ent"
	dbHandler = ""
	redisHost := "localhost"
	redisPort := "6379"
	redisPasswd = ""
	redisAddr = consts.RedisAddr
	f, err := os.Open(filename)
	if err != nil {
		return
	}
	r := bufio.NewReader(f)
	for {
		line, _, err := r.ReadLine()
		if err != nil {
			break
		}

		if strings.HasPrefix(strings.TrimSpace(string(line)), "#") {
			continue
		}
		m := strings.Split(string(line), "=")
		if len(m) < 2 {
			continue
		}
		k := strings.TrimSpace(m[0])
		v := strings.TrimSpace(strings.Join(m[1:], "="))

		switch strings.TrimSpace(k) {
		case "user":
			user = v
		case "host":
			host = v
		case "port":
			port = v
		case "passwd":
			passwd = v
		case "db_name":
			dbName = v
		case "redis_host":
			redisHost = v
		case "redis_port":
			redisPort = v
		case "redis_passwd":
			redisPasswd = v
		default:
			continue
		}
	}

	//dbHandler = "root:123456@tcp(localhost:3306)/ent"
	dbHandler = fmt.Sprintf("%s:%s@tcp(%s:%s)/%s", user, passwd, host, port, dbName)
	//dbHandler = fmt.Sprint(user, ":", passwd, "@tcp(", host, ":", port, ")/", dbName)
	redisAddr = fmt.Sprintf("%s:%s", redisHost, redisPort)
	return
}

func DownloadImage(imgURL, folderPath string) (string, error) {
	emptyStr := ""
	suffix := ".jpg"
	suffixRegexp := regexp.MustCompile(`(\.\w+)$`)
	m := suffixRegexp.FindStringSubmatch(imgURL)
	if len(m) == 2 {
		suffix = m[1]
	}
	fileName, err := randutil.AlphaStringRange(30, 40)
	if err != nil {
		return emptyStr, err
	}
	imgName := fileName + suffix

	resp, err := getThrice(imgURL)
	if err != nil {
		return emptyStr, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return emptyStr, errors.New("return code not 200 OK")
	}
	content, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return emptyStr, err
	}

	//fmt.Println(time.Now().Format(TimeFormat), "img", imgURL, "length", len(content))

	imgWriter, err := os.Create(strings.TrimSuffix(folderPath, "/") + "/" + imgName)
	if err != nil {
		return emptyStr, err
	}
	defer imgWriter.Close()

	_, err = imgWriter.Write(content)
	if err != nil {
		return emptyStr, err
	}
	return imgName, nil
}

func getThrice(link string) (resp *http.Response, err error) {
	resp = nil
	client := http.Client{
		Timeout: time.Duration(10 * time.Second),
	}
	req, err := http.NewRequest("GET", link, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("User-Agent", consts.UserAgent)

	for i := 0; i < 3; i++ {
		if i > 0 {
			log.Println(err.Error(), "try again", link)
		}
		resp, err = client.Do(req)
		if err != nil {
			continue
		}
		if resp.StatusCode != 200 {
			resp.Body.Close()
			err = errors.New("return code not 200 OK")
			continue
		} else {
			break
		}
	}
	return resp, err

}

func FindStar(star string, pairs []StarIDPair) bool {
	flag := false
	for _, pair := range pairs {
		if star == pair.NameCN {
			flag = true
			break
		}
	}
	return flag
}

//input: 2017-01-02 15:09:02
func GetDuration(inputTime string) time.Duration {
	preTime, err := time.Parse("2006-01-02 15:04:05", inputTime)
	if err != nil {
		return 0
	}
	pre := preTime.Unix()
	cur := time.Now().Unix()
	if pre > cur {
		return 0
	}
	return time.Duration(cur - pre)
}

//starIDMap: map[starName] = starID, idStarMap: map[starID] = starName
func GetRedisStarID(client *redis.Client) (starIDMap, idStarMap map[string]string, err error) {
	starIDMap = map[string]string{}
	idStarMap = map[string]string{}

	names, err := client.Keys(consts.RedisNamePrefix + "*").Result()
	if err != nil {
		return starIDMap, idStarMap, errors.New("error while get star name key from redis: " + err.Error())
	}
	for _, tmpName := range names {
		name := strings.TrimPrefix(tmpName, consts.RedisNamePrefix)
		starID := client.Get(tmpName).Val()
		starIDMap[name] = starID
		idStarMap[starID] = name
	}
	return starIDMap, idStarMap, nil
}

//nicknameMap: map[nickname] = starID
func GetNickname(db *sql.DB) (NicknameMap map[string]string, err error) {
	NicknameMap = map[string]string{}
	queryStr := "select name, star_id from nickname"
	rows, err := db.Query(queryStr)
	if err != nil {
		return NicknameMap, errors.New("error while get stars from table nickname: " + err.Error())
	}
	for rows.Next() {
		var starID, nickname sql.NullString
		err = rows.Scan(&nickname, &starID)
		if err != nil {
			log.Println("error while scan nickname rows:", err.Error())
			continue
		}
		if nickname.Valid && starID.Valid {
			NicknameMap[nickname.String] = starID.String
		}
	}
	return NicknameMap, nil
}

//get input keys to lower case to make sure keys match
//starIDMap: star_name->star_id
//map[string]StarIDPair : star name -> star id pair
func GetSearchStarList(filename string, starIDMap map[string]string) map[string]StarIDPair {
	resultMap := map[string]StarIDPair{}

	pairMap := map[string]StarIDPair{}
	for name, starID := range starIDMap {
		tmpPair := StarIDPair{
			NameCN: name,
			StarID: starID,
		}
		name = strings.ToLower(name)
		pairMap[name] = tmpPair
	}
	content, err := ioutil.ReadFile(filename)
	if err != nil {
		log.Println("error while read file:", filename, err.Error())
		return resultMap
	}
	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if len(line) == 0 || strings.HasPrefix(line, "#") {
			continue
		}
		names := strings.Split(line, ",")
		if len(names) == 0 {
			continue
		}
		tmpName := strings.ToLower(names[0])
		tmpPair, found := pairMap[tmpName]
		if !found {
			log.Println("star name not found in star_name table:", names[0])
			continue
		}

		if len(names) > 1 {
			resultMap[names[1]] = tmpPair
			//tmpName = names[1]
		} else {
			resultMap[names[0]] = tmpPair
		}

		fmt.Println(time.Now().Format(consts.TimeFormat), "name in db:", tmpPair.NameCN, "search name:", tmpName,
			"star id:", tmpPair.StarID)
	}
	return resultMap
}

//转义字符串，用于MySQL存储
func EscapeStr(str string) string {
	result := strings.TrimSpace(str)
	result = strings.Replace(str, ";", "；", -1)
	result = strings.Replace(result, ",", "，", -1)
	result = strings.Replace(result, "'", "\"", -1)
	return result
}

//sql.NullString => string
func ConvertStrings(input []sql.NullString) []string {
	result := []string{}
	for _, tmpStr := range input {
		result = append(result, tmpStr.String)
	}
	return result
}


//用于插入分页符
//a	input string slice
//sep	separator
//m	gaps between two sep
func JoinWithBreak(a []string, brk string, sep string, m int) string {
	if len(a) == 0 || m < 1 {
		return ""
	}
	if len(a) <= m {
		return strings.Join(a, "")
	}

	remain := len(a) % m
	x := len(a) / m
	if remain == 0 {
		remain = m
		x--
	}

	n := len(brk) * x + len(sep) * (len(a) - 1 - x)
	for i := 0; i < len(a); i++ {
		n += len(a[i])
	}

	b := make([]byte, n)
	bp := copy(b, a[0])
	for i := 1; i < remain; i++ {
		bp += copy(b[bp:], sep)
		bp += copy(b[bp:], a[i])
	}

	count := 0
	for _, s := range a[remain:] {
		if count % m == 0 {
			bp += copy(b[bp:], brk)
		} else {
			bp += copy(b[bp:], sep)
		}
		bp += copy(b[bp:], s)
		count++
	}
	return string(b)
}

func InterfaceToString(x interface{}) string {
	result := ""
	switch x.(type) {
	case float64:
		result = strconv.Itoa(int(x.(float64)))
	case float32:
		result = strconv.Itoa(int(x.(float32)))
	case int:
		result = strconv.Itoa(x.(int))
	case string:
		result = x.(string)
	default:
		result = ""
	}
	return result
}
