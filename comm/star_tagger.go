//author tyf
//date   2017-02-06 17:01
//desc

package comm

import "github.com/tanyfx/ent/comm/ahocorasick"

//StarID, NameCN string
type StarIDPair struct {
	StarID string
	NameCN string
}

type StarTagger struct {
	acMatcher *ahocorasick.Matcher

	useNickname bool
	idStarMap   map[string]string //starMap star_id -> star_name
	starIDMap   map[string]string //star_name/nickname -> star_id
}

//starMap: star_name -> star_id
func NewStarTagger(starMap map[string]string) *StarTagger {
	idStarMap := map[string]string{}
	return &StarTagger{
		acMatcher:   ahocorasick.NewMapMatcher(starMap),
		idStarMap:   idStarMap,
		starIDMap:   starMap,
		useNickname: false,
	}
}

//idStarMap: star_id -> star_name
//nicknameMap: nickname -> star_id
func NewStarNicknameTagger(idStarMap, nicknameMap map[string]string) *StarTagger {
	return &StarTagger{
		acMatcher:   ahocorasick.NewMapMatcher(nicknameMap),
		idStarMap:   idStarMap,
		starIDMap:   nicknameMap,
		useNickname: true,
	}
}

func (p *StarTagger) TagStar(inputStr string) []StarIDPair {
	return p.tagStarAC(inputStr)
}

func (p *StarTagger) tagStarAC(str string) []StarIDPair {
	stars := []StarIDPair{}
	matches := p.acMatcher.SimpleMatch([]byte(str))
	if p.useNickname {
		for word := range matches {
			if id, ok := p.starIDMap[word]; ok {
				if starName, ok := p.idStarMap[id]; ok {
					stars = append(stars, StarIDPair{
						StarID: id,
						NameCN: starName,
					})
				}
			}
		}
	} else {
		for starName := range matches {
			if id, ok := p.starIDMap[starName]; ok {
				stars = append(stars, StarIDPair{
					StarID: id,
					NameCN: starName,
				})
			}
		}
	}
	return stars
}
