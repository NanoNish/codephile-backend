package scrappers

import (
	"errors"
	. "github.com/mdg-iitr/Codephile/conf"
	. "github.com/mdg-iitr/Codephile/errors"
	"github.com/mdg-iitr/Codephile/models/types"
	"github.com/mdg-iitr/Codephile/scrappers/codechef"
	"github.com/mdg-iitr/Codephile/scrappers/codeforces"
	"github.com/mdg-iitr/Codephile/scrappers/hackerrank"
	"github.com/mdg-iitr/Codephile/scrappers/spoj"
	"time"
)

type Scrapper interface {
	CheckHandle() bool
	GetSubmissions(after time.Time) []types.Submission
	GetProfileInfo() types.ProfileInfo
}

func NewScrapper(site string, handle string) (Scrapper, error) {
	if handle == "" {
		return nil, HandleNotFoundError
	}
	switch site {
	case CODEFORCES:
		return codeforces.Scrapper{Handle: handle}, nil
	case CODECHEF:
		return codechef.Scrapper{Handle: handle}, nil
	case HACKERRANK:
		return hackerrank.Scrapper{Handle: handle}, nil
	case SPOJ:
		return spoj.Scrapper{Handle: handle}, nil
	default:
		return nil, errors.New("site invalid")
	}
}