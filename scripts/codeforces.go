package scripts

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/mdg-iitr/Codephile/models/types"
	"log"
	"strconv"
	"time"
)

// CodeforcesGraphPoint represents a single point for codeforces
type CodeforcesGraphPoint struct {
	ContestName string
	Date        time.Time
	Rating      float64
}

// CodeforcesGraphPoints represents the graph points for codeforces
type CodeforcesGraphPoints struct {
	Count  int
	Points []CodeforcesGraphPoint
}

//CodeforcesContests represents the codeforces contest
type CodeforcesContests struct {
	Data  []CodeforcesContest
	Count int
}
type CodeforcesContest struct {
	ContestName string `json:"name"`
	Rated       bool   `json:"rated"`
	EpochStart  int64  `json:"epoch_starttime"`
	EpochEnd    int64  `json:"epoch_endtime"`
	Archived    bool   `json:"archived"`
}

// UnmarshalJSON implements the unmarshaler interface for CodeforcesGraphPoint
func (points *CodeforcesGraphPoints) UnmarshalJSON(b []byte) error {
	var data map[string]interface{}
	err := json.Unmarshal(b, &data)
	if data["status"] != "OK" {
		return errors.New("Bad Request")
	}
	results := data["result"].([]interface{})
	points.Count = len(results)
	for _, result := range results {
		point := CodeforcesGraphPoint{
			ContestName: result.(map[string]interface{})["contestName"].(string),
			Date:        time.Unix(int64(result.(map[string]interface{})["ratingUpdateTimeSeconds"].(float64)), 0),
			Rating:      result.(map[string]interface{})["newRating"].(float64),
		}
		points.Points = append(points.Points, point)
	}
	return err
}

// UnmarshalJSON implements the unmarshaler interface for CodeforcesContests
func (contests *CodeforcesContests) UnmarshalJSON(b []byte) error {
	var data map[string]interface{}
	err := json.Unmarshal(b, &data)
	if data["status"] != "OK" {
		return errors.New("Bad Request")
	}
	results := data["result"].([]interface{})[0:20]
	contests.Count = 20
	for _, result := range results {
		resultMap := result.(map[string]interface{})
		Contest := CodeforcesContest{
			ContestName: resultMap["name"].(string),
			Rated:       true,
			EpochStart:  int64(resultMap["startTimeSeconds"].(float64)),
		}
		Contest.EpochEnd = int64(resultMap["durationSeconds"].(float64)) + Contest.EpochStart
		phase := resultMap["phase"].(string)
		if phase == "FINISHED" {
			Contest.Archived = true
		}
		contests.Data = append(contests.Data, Contest)
	}
	return err
}

func GetCodeforcesProfileInfo(handle string) types.ProfileInfo {
	var profile types.ProfileInfo
	url := "http://codeforces.com/api/user.info?handles=" + handle
	data := GetRequest(url)
	err := json.Unmarshal(data, &profile)
	if err != nil {
		log.Println(err.Error())
	}
	return profile
}

func GetCodeforcesGraphData(handle string) CodeforcesGraphPoints {
	var points CodeforcesGraphPoints
	url := "http://codeforces.com/api/user.rating?handle=" + handle
	data := GetRequest(url)
	err := json.Unmarshal(data, &points)
	if err != nil {
		log.Println(err.Error())
	}
	return points
}
func getCodeforcesSubmissionParts(handle string, afterIndex int) types.CodeforcesSubmissions {
	url := "http://codeforces.com/api/user.status?handle=" + handle + "&from=" + strconv.Itoa(afterIndex) + "&count=50"
	fmt.Println(url)
	data := GetRequest(url)
	var submissions types.CodeforcesSubmissions
	err := json.Unmarshal(data, &submissions)
	if err != nil {
		log.Println(err.Error())
	}
	return submissions
}

func GetCodeforcesSubmissions(handle string, after time.Time) types.CodeforcesSubmissions {
	var oldestSubIndex, current int
	var oldestSubFound = false
	var subs types.CodeforcesSubmissions
	//Fetch submission until oldest submission not found
	for !oldestSubFound {
		newSub := getCodeforcesSubmissionParts(handle, current+1);
		//Check for repetition of previous fetched submission
		if len(newSub.Data) != 0 {
			for i, sub := range newSub.Data {
				subs.Data = append(subs.Data, sub)
				oldestSubIndex = current + i + 1
				if sub.CreationDate.Equal(after) || sub.CreationDate.Before(after) {
					oldestSubFound = true
					break
				}
			}
			//50 submissions per page
			current += 50
		} else {
			break
		}
	}
	subs.Data = subs.Data[0:oldestSubIndex]
	subs.Count = len(subs.Data)
	return subs

}

func GetCodeforcesContests() CodeforcesContests {
	data := GetRequest("https://codeforces.com/api/contest.list?gym=false")
	var contests CodeforcesContests
	err := json.Unmarshal(data, &contests)
	if err != nil {
		log.Println(err.Error())
	}
	return contests
}
func CheckCodeforcesHandle(handle string) bool {
	data := GetRequest("http://codeforces.com/api/user.info?handles=" + handle)
	var i interface{}
	err := json.Unmarshal(data, &i)
	if err != nil {
		log.Println(err.Error())
	}
	return i.(map[string]interface{})["status"] != "FAILED"
}
