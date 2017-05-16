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
	acMatcher   *ahocorasick.Matcher

	idStarMap   map[string]string //starMap star_id -> star_name
	nicknameMap map[string]string //nickname -> star_id
}

//starMap: star_name -> star_id
func NewStarTagger(starMap map[string]string) *StarTagger {
	idStarMap := map[string]string{}
	for name, id := range starMap {
		idStarMap[id] = name
	}
	return &StarTagger{
		acMatcher: ahocorasick.NewMapMatcher(starMap),
		idStarMap: idStarMap,
		nicknameMap: starMap,
	}
}

//idStarMap: star_id -> star_name
//nicknameMap: nickname -> star_id
func NewStarNicknameTagger(idStarMap, nicknameMap map[string]string) *StarTagger {
	return &StarTagger{
		acMatcher: ahocorasick.NewMapMatcher(nicknameMap),
		idStarMap: idStarMap,
		nicknameMap: nicknameMap,
	}
}

func (p *StarTagger) TagStar(inputStr string) []StarIDPair {
	return p.tagStarAC(inputStr)
}

func (p *StarTagger) tagStarAC(str string) []StarIDPair {
	stars := []StarIDPair{}
	matches := p.acMatcher.SimpleMatch([]byte(str))
	for word := range matches {
		if id, ok := p.nicknameMap[word]; ok {
			if starName, ok := p.idStarMap[id]; ok {
				stars = append(stars, StarIDPair{
					StarID: id,
					NameCN: starName,
				})
			}
		}
	}
	return stars
}
