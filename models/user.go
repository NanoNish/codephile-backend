package models

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/globalsign/mgo/bson"
	"github.com/mdg-iitr/Codephile/models/db"
	"github.com/mdg-iitr/Codephile/models/profile"
	"github.com/mdg-iitr/Codephile/models/submission"
    Follow "github.com/mdg-iitr/Codephile/models/Follow"
	"github.com/mdg-iitr/Codephile/scripts"
	search "github.com/mdg-iitr/Codephile/services/elastic"
	"golang.org/x/crypto/bcrypt"
	"log"
	"time"
)

type User struct {
	ID          bson.ObjectId          `bson:"_id" json:"id" schema:"-"`
	Username    string                 `bson:"username" json:"username" schema:"username"`
	Password    string                 `bson:"password" json:"-" schema:"password"`
	Picture     string                 `bson:"picture" json:"picture"`
	Handle      Handle                 `bson:"handle" json:"handle" schema:"handle"`
	Submissions submission.Submissions `bson:"submission" json:"-" schema:"-"`
	Last        LastFetchedSubmission  `bson:"lastfetched" json:"-"`
	FollowingUsers []Follow.Following  `bson:"followingUsers" json:"followingUsers"`
}
type LastFetchedSubmission struct {
	Codechef   time.Time `bson:"codechef"`
	Codeforces time.Time `bson:"codeforces"`
	Hackerrank time.Time `bson:"hackerrank"`
	Spoj       time.Time `bson:"spoj"`
}
type Handle struct {
	Codeforces  string `bson:"codeforces" json:"codeforces" schema:"codeforces"`
	Codechef    string `bson:"codechef" json:"codechef" schema:"codechef"`
	Spoj        string `bson:"spoj" json:"spoj" schema:"spoj"`
	Hackerrank  string `bson:"hackerrank" json:"hackerrank" schema:"hackerrank"`
	Hackerearth string `bson:"hackerearth" json:"hackerearth" schema:"hackerearth"`
}

func (u *User) UnmarshalJSON(b []byte) error {
	var m map[string]interface{}
	err := json.Unmarshal(b, &m)
	if val, ok := m["password"]; ok {
		u.Password = val.(string)
	}
	if val, ok := m["username"]; ok {
		u.Username = val.(string)
	}
	if val, ok := m["handle"]; ok {
		d, _ := json.Marshal(val)
		err = json.Unmarshal(d, &u.Handle)
	}
	return err
}
func AddUser(u User) (string, error) {
	u.ID = bson.NewObjectId()
	client := search.GetElasticClient()
	_, err := client.Index().Index("codephile").BodyJson(u).Id(u.ID.String()).Refresh("true").Do(context.Background())
	if err != nil {
		log.Println(err.Error())
	}
	collection := db.NewCollectionSession("coduser")
	defer collection.Close()
	//hashing the password
	hash, err := bcrypt.GenerateFromPassword([]byte(u.Password), bcrypt.DefaultCost)
	//data type of hash is []byte
	u.Password = string(hash)
	if err != nil {
		log.Println(err)
	}
	err = collection.Session.Insert(u)
	if err != nil {
		return "", errors.New("Could not create user: Username already exists")
	}
	return u.ID.Hex(), nil
}

func GetUser(uid bson.ObjectId) (*User, error) {
	var user User
	collection := db.NewCollectionSession("coduser")
	defer collection.Close()
	err := collection.Session.FindId(uid).Select(bson.M{"_id": 1, "username": 1, "handle": 1, "lastfetched": 1, "picture": 1,}).One(&user)
	//fmt.Println(err.Error())
	if err != nil {
		return nil, errors.New("user not exists")
	}
	return &user, nil
}

func GetAllUsers() []User {
	var users []User
	collection := db.NewCollectionSession("coduser")
	defer collection.Close()
	err := collection.Session.Find(nil).Select(bson.M{"_id": 1, "username": 1, "handle": 1, "picture": 1}).All(&users)
	if err != nil {
		panic(err)
	}
	return users
}

func UpdateUser(uid bson.ObjectId, uu *User) (a *User, err error) {
	if u, err := GetUser(uid); err == nil {
		if uu.Username != "" {
			u.Username = uu.Username
		}
		if uu.Handle.Codechef != "" {
			u.Handle.Codechef = uu.Handle.Codechef
		}
		if uu.Handle.Codeforces != "" {
			u.Handle.Codeforces = uu.Handle.Codeforces
		}
		if uu.Handle.Hackerearth != "" {
			u.Handle.Hackerearth = uu.Handle.Hackerearth
		}
		if uu.Handle.Hackerrank != "" {
			u.Handle.Hackerrank = uu.Handle.Hackerrank
		}
		if uu.Handle.Hackerearth != "" {
			u.Handle.Hackerearth = uu.Handle.Hackerearth
		}
		client := search.GetElasticClient()
		_, err = client.Update().Index("codephile").Id(uid.String()).Doc(u).Upsert(u).Do(context.Background())
		if err != nil {
			log.Println(err.Error())
		}
		collection := db.NewCollectionSession("coduser")
		_, err := collection.Session.UpsertId(uid, &u)
		if err != nil {
			err = errors.New("username already exists")
		}
		return u, err
	}
	return nil, errors.New("User Not Exist")
}
func AutheticateUser(username string, password string) (*User, bool) {
	var user User
	collection := db.NewCollectionSession("coduser")
	defer collection.Close()
	err := collection.Session.Find(bson.M{"username": username}).One(&user)
	//fmt.Println(err.Error())
	if err != nil {
		log.Println(err)
		return nil, false
	}

	err2 := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err2 != nil {
		log.Println(err2)
		return nil, false
	} else {
		return &user, true
	}

}

func AddSubmissions(user *User, site string) error {
	var handle string
	coll := db.NewCollectionSession("coduser")
	switch site {
	case "codechef":
		handle = user.Handle.Codechef
		if handle == "" {
			return errors.New("handle not available")
		}
		addSubmissions := scripts.GetCodechefSubmissions(handle, user.Last.Codechef)
		if len(addSubmissions) != 0 {
			user.Last.Codechef = addSubmissions[0].CreationDate
			change := bson.M{"$push": bson.M{"submission.codechef": bson.M{"$each": addSubmissions}}, "$set": bson.M{"lastfetched": user.Last}}
			err := coll.Session.UpdateId(user.ID, change)
			if err != nil {
				log.Fatal(err.Error())
			}
		}
		return nil
	case "codeforces":
		handle = user.Handle.Codeforces
		if handle == "" {
			return errors.New("handle not available")
		}
		fmt.Println(user.Last.Codeforces)
		addSubmissions := scripts.GetCodeforcesSubmissions(handle, user.Last.Codeforces).Data
		if len(addSubmissions) != 0 {
			user.Last.Codeforces = addSubmissions[0].CreationDate
			change := bson.M{"$push": bson.M{"submission.codeforces": bson.M{"$each": addSubmissions}}, "$set": bson.M{"lastfetched": user.Last}}
			err := coll.Session.UpdateId(user.ID, change)
			if err != nil {
				log.Fatal(err.Error())
			}
		}
		return nil
	case "spoj":
		handle = user.Handle.Spoj
		if handle == "" {
			return errors.New("handle not available")
		}
		addSubmissions := scripts.GetSpojSubmissions(handle, user.Last.Spoj)
		if len(addSubmissions) != 0 {
			user.Last.Spoj = addSubmissions[0].CreationDate
			change := bson.M{"$push": bson.M{"submission.spoj": bson.M{"$each": addSubmissions}}, "$set": bson.M{"lastfetched": user.Last}}
			err := coll.Session.UpdateId(user.ID, change)
			if err != nil {
				log.Fatal(err.Error())
			}
		}
		return nil
	case "hackerrank":
		handle = user.Handle.Hackerrank
		if handle == "" {
			return errors.New("handle not available")
		}
		addSubmissions := scripts.GetHackerrankSubmissions(handle, user.Last.Hackerrank).Data
		if len(addSubmissions) != 0 {
			user.Last.Hackerrank = addSubmissions[0].CreationDate
			change := bson.M{"$push": bson.M{"submission.hackerrank": bson.M{"$each": addSubmissions}}, "$set": bson.M{"lastfetched": user.Last}}
			err := coll.Session.UpdateId(user.ID, change)
			if err != nil {
				log.Fatal(err.Error())
			}
		}
		return nil
	}
	return nil
}

func GetSubmissions(ID bson.ObjectId) (*submission.Submissions, error) {
	coll := db.NewCollectionSession("coduser")
	var user User
	err := coll.Session.FindId(ID).Select(bson.M{"submission": 1}).One(&user)
	if err != nil {
		return nil, errors.New("user not found")
	}
	return &user.Submissions, nil
}

func AddorUpdateProfile(uid bson.ObjectId, site string) (*User, error) {
	user, err := GetUser(uid)
	if err != nil {
		//handle the error (Invalid user)
		return nil, err
	}
	var UserProfile profile.ProfileInfo
	//runs code to fetch the particular script's getProfile function
	switch site {
	case "codechef":
		UserProfile = scripts.GetCodechefProfileInfo(user.Handle.Codechef)
		break
	case "codeforces":
		UserProfile = scripts.GetCodeforcesProfileInfo(user.Handle.Codeforces)
		break
	case "spoj":
		UserProfile = scripts.GetSpojProfileInfo(user.Handle.Spoj)
		break
	case "hackerrank":
		UserProfile = scripts.GetHackerrankProfileInfo(user.Handle.Hackerrank)
		break
	} // add a default case for non-existent website
	//Profile fetched. Store in database 
		var ProfileTobeInserted profile.Profile
		ProfileTobeInserted.Website = site
		ProfileTobeInserted.Profileinfo = UserProfile
		// ProfileTobeInserted is all set to be put in the database
		collection := db.NewCollectionSession("coduser")
		defer collection.Close()
		// err2 := collection.Session.Update(bson.D{{"_id" , user.ID}},bson.D{{"$set" , ProfileTobeInserted}})
		NewNode := site + "Profile"
		SelectedUser := bson.D{{"_id", user.ID}}
		Update := bson.D{{"$set", bson.D{{NewNode, ProfileTobeInserted}}}}
		_, err2 := collection.Session.Upsert(SelectedUser, Update)
		//inserted into the document
		if err2 == nil {
			return user, nil
		} else {
			return nil, err2
		}
}

func GetProfiles(ID bson.ObjectId) (profile.AllProfiles, error) {
	coll := db.NewCollectionSession("coduser")
	var profiles profile.AllProfiles
	var profilesToBeReturned profile.AllProfiles //appends the profile to this variable which will be returned
	err1 := coll.Session.FindId(ID).Select(bson.M{"codechefProfile": 1}).One(&profiles)
	profilesToBeReturned.CodechefProfile = profiles.CodechefProfile
	err2 := coll.Session.FindId(ID).Select(bson.M{"codeforcesProfile": 1}).One(&profiles)
	profilesToBeReturned.CodeforcesProfile = profiles.CodeforcesProfile
	err3 := coll.Session.FindId(ID).Select(bson.M{"hackerrankProfile": 1}).One(&profiles)
	profilesToBeReturned.HackerrankProfile = profiles.HackerrankProfile
	err4 := coll.Session.FindId(ID).Select(bson.M{"spojProfile": 1}).One(&profiles)
	profilesToBeReturned.SpojProfile = profiles.SpojProfile
	if err1 == nil && err2 == nil && err3 == nil && err4 == nil {
		return profilesToBeReturned, nil
	} else {
		if err1 != nil {
			return profilesToBeReturned, err1
		} else if err2 != nil {
			return profilesToBeReturned, err2
		} else if err3 != nil {
			return profilesToBeReturned, err3
		} else {
			return profilesToBeReturned, err4
		}
	}
}
func FilterSubmission(uid bson.ObjectId, status string, tag string, site string) ([]map[string]interface{}, error) {
	c := db.NewCollectionSession("coduser")
	fmt.Println(status)
	match1 := bson.M{
		"$match": bson.M{
			"_id": uid,
		},
	}
	unwind := bson.M{
		"$unwind": "$submission." + site,
	}
	match2 := bson.M{
		"$match": bson.M{"submission." + site + ".status": status},
	}
	project := bson.M{
		"$project": bson.M{
			"_id":                0,
			"submission." + site: 1,
		},
	}
	all := []bson.M{match1, unwind, match2, project}
	pipe := c.Session.Pipe(all)

	var result map[string]interface{}
	iter := pipe.Iter()
	var final []map[string]interface{}
	for iter.Next(&result) {
		final = append(final, result["submission"].(map[string]interface{})[site].(map[string]interface{}))
	}
	return final, nil
}

func UpdatePicture(uid bson.ObjectId, url string) error {
	client := search.GetElasticClient()
	_, err := client.Update().Index("codephile").Id(uid.String()).Doc(map[string]interface{}{"picture": url}).Do(context.Background())
	if err != nil {
		log.Println(err.Error())
	}
	coll := db.NewCollectionSession("coduser")
	_, err = coll.Session.UpsertId(uid, bson.M{"$set": bson.M{"picture": url}})
	if err != nil {
		return err
	}
	return nil
}

func GetFollowingUsers(ID bson.ObjectId) ([]Follow.Following, error) {
	coll := db.NewCollectionSession("coduser")
	defer coll.Close()
	var user User
	err := coll.Session.FindId(ID).Select(bson.M{"followingUsers": 1}).One(&user)
	if err != nil {
		return nil, errors.New("user not found")
	}
	return user.FollowingUsers, nil
}

func FollowUser(uid1 bson.ObjectId, uid2 string) error{
	//uid1 is of the person who wants to follow
	//uid2 is the person being followed
     if uid2 != ""  && bson.IsObjectIdHex(uid2) {
			user1 , err1 := GetUser(uid1)
			user2 , err2 := GetUser(bson.ObjectIdHex(uid2))
            if err1 == nil && err2 == nil {
				//add the uid2 in the database of uid1
				var following Follow.Following
				following.ID = user2.ID
				following.CodephileHandle = user2.Username
				update := bson.M{"$addToSet": bson.M{"followingUsers" : following}}
				collection := db.NewCollectionSession("coduser")
				defer collection.Close()
				err := collection.Session.UpdateId(user1.ID,update)
				return err
			} else {
				//unable to get the user from database
				return errors.New("Unable to fetch the user from the database")
			}
	 } else {
		    //uid is not valid
		    return errors.New("UID Invalid")	
	 }
}

func CompareUser(uid1 bson.ObjectId, uid2 string) (Follow.AllWorldRanks , error)   {
	var worldRanksComparison Follow.AllWorldRanks
	if uid2 != "" && bson.IsObjectIdHex(uid2) {
			//add the uid2 in the database of uid1
			collection := db.NewCollectionSession("coduser")
			defer collection.Close()
			//gets the different profiles to fetch world ranks
			profiles1 , err1 := GetProfiles(uid1)
			profiles2 , err2 := GetProfiles(bson.ObjectIdHex(uid2))
			
			//puts the world ranks in the struct fields
			worldRanksComparison.CodechefWorldRanks.WorldRank1 = profiles1.CodechefProfile.Profileinfo.WorldRank
			worldRanksComparison.CodechefWorldRanks.WorldRank2 = profiles2.CodechefProfile.Profileinfo.WorldRank
			
			worldRanksComparison.CodeforcesWorldRanks.WorldRank1 = profiles1.CodeforcesProfile.Profileinfo.WorldRank
			worldRanksComparison.CodeforcesWorldRanks.WorldRank2 = profiles2.CodeforcesProfile.Profileinfo.WorldRank
			
			worldRanksComparison.HackerrankWorldRanks.WorldRank1 = profiles1.HackerrankProfile.Profileinfo.WorldRank
			worldRanksComparison.HackerrankWorldRanks.WorldRank2 = profiles2.HackerrankProfile.Profileinfo.WorldRank
			
			worldRanksComparison.SpojWorldRanks.WorldRank1 = profiles1.SpojProfile.Profileinfo.WorldRank
			worldRanksComparison.SpojWorldRanks.WorldRank2 = profiles2.SpojProfile.Profileinfo.WorldRank
			
			//handle the errors
			if err1 != nil || err2 != nil {
				return worldRanksComparison, errors.New("Unable to fetch user from database")
			} else {
			    return worldRanksComparison, nil
			}
    } else {
	      //uid is not valid
	      return worldRanksComparison, errors.New("UID Invalid")	
    }     
}
