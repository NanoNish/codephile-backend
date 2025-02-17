package conf

const (
	CODECHEF   = "codechef"
	CODEFORCES = "codeforces"
	HACKERRANK = "hackerrank"
	SPOJ       = "spoj"
	LEETCODE   = "leetcode"
)

var ValidSites = []string{HACKERRANK, CODECHEF, CODEFORCES, SPOJ, LEETCODE}

func GetRegexSite(site string) string {
	switch site {
	case CODECHEF:
		return "https://www.codechef.com"
	case CODEFORCES:
		return "http://codeforces.com"
	case HACKERRANK:
		return "https://www.hackerrank.com"
	case SPOJ:
		return "https://www.spoj.com"
	case LEETCODE:
		return "https://leetcode.com/"
	}
	return " "
}

func IsSiteValid(s string) bool {
	for _, vs := range ValidSites {
		if s == vs {
			return true
		}
	}
	return false
}

const (
	StatusCorrect             = "AC"
	StatusWrongAnswer         = "WA"
	StatusCompilationError    = "CE"
	StatusRuntimeError        = "RE"
	StatusTimeLimitExceeded   = "TLE"
	StatusMemoryLimitExceeded = "MLE"
	StatusPartial             = "PTL"
)
