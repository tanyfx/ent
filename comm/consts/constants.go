//author tyf
//date   2017-02-07 15:01
//desc 

package consts

import "time"

const ThreadNum int = 4

const DELM string = "\001"
const SEP string = "\002"

const Year time.Duration = time.Hour * 24 * 365
const Duration time.Duration = time.Hour * 24 * 365

const UserAgent string = "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) " +
	"Chrome/52.0.2743.116 Safari/537.36"
const MobileUA string = `Mozilla/5.0 (Linux; Android 6.0;` +
	`Nexus 5 Build/MRA58N) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/55.0.2883.87 Mobile Safari/537.36`
const ContentType string = `application/x-www-form-urlencoded`
const TimeFormat string = "2006/01/02 15:04:05"

const WPBreak string = "<!--nextpage-->"
const WPBreakGap int = 5

const SearchStar string = "search_star"
const SearchID string = "search_id"
