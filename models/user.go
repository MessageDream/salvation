package models

import (
	"time"
	"errors"
	"strings"
	"fmt"
	"crypto/sha256"
	"encoding/hex"
	"sync"

	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"github.com/Unknwon/com"

	"github.com/MessageDream/salvation/modules/setting"
	"github.com/MessageDream/salvation/modules/log"
	"github.com/MessageDream/salvation/modules/base"
)

type UserType int

const (
	INDIVIDUAL UserType = iota // Historic reason to make it starts at 0.
	ORGANIZATION
)

var (
	ErrUserOwnRepos          = errors.New("User still have ownership of repositories")
	ErrUserHasOrgs           = errors.New("User still have membership of organization")
	ErrUserAlreadyExist      = errors.New("User already exist")
	ErrUserNotExist          = errors.New("User does not exist")
	ErrUserNotKeyOwner       = errors.New("User does not the owner of public key")
	ErrEmailAlreadyUsed      = errors.New("E-mail already used")
	ErrUserNameIllegal       = errors.New("User name contains illegal characters")
	ErrLoginSourceNotExist   = errors.New("Login source does not exist")
	ErrLoginSourceNotActived = errors.New("Login source is not actived")
	ErrUnsupportedLoginType  = errors.New("Login source is unknown")
)

var (
	lock sync.Locker = &sync.Mutex{}
	userIndex=0;
	illegalEquals  = []string{"debug", "raw", "install", "api", "avatar", "user", "org", "help", "stars", "issues", "pulls", "commits", "repo", "template", "admin", "new"}
)

//主题id和评论id，用于定位到专门的评论
type At struct {
	User      string
	ContentId string
	CommentId string
}

//主题id和主题标题
type Reply struct {
	ContentId  string
	TopicTitle string
}

//收藏的话题
type CollectTopic struct {
	TopicId       string
	TimeCollected time.Time
}

func (ct *CollectTopic) Topic(db *mgo.Database) *Topic {
	c := db.C(CONTENTS)
	var topic Topic
	err := c.Find(bson.M{"_id": bson.ObjectIdHex(ct.TopicId), "content.type": TypeTopic}).One(&topic)
	if err != nil {
		panic(err)
		return nil
	}
	return &topic

}

// 用户
type User struct {
	Id_             bson.ObjectId `bson:"_id"`
	UserName        string
	LowerName     	string
	FullName      	string
	Password        string
	Salt            string 			`bson:"salt"`
	Email           string
	Avatar          string
	LoginType     	LoginType
	LoginSource   	int64
	Website         string
	Location        string
	Tagline         string
	Bio             string
	Twitter         string
	Weibo           string
	JoinedAt        time.Time
	Follow          []string
	Fans            []string
	RecentReplies   []Reply        //存储的是最近回复的主题的objectid.hex
	RecentAts       []At           //存储的是最近评论被AT的主题的objectid.hex
	TopicsCollected []CollectTopic //用户收藏的topic数组
	IsAdmin     	bool
	IsActive        bool
	ValidateCode    string
	ResetCode       string
	Rands			string
	Index           int
}

// 是否是默认头像
func (u *User) IsDefaultAvatar(avatar string) bool {
	filename := u.Avatar
	if filename == "" {
		filename = DefaultAvatar
	}

	return filename == avatar
}

// 头像的图片地址
func (u *User) AvatarImgSrc() string {
	// 如果没有设置头像，用默认头像
	filename := u.Avatar
	if filename == "" {
		filename = DefaultAvatar
	}

	return "http://gopher.qiniudn.com/avatar/" + filename
}

// 用户发表的最近10个主题
func (u *User) LatestTopics(db *mgo.Database) *[]Topic {
	c := db.C(CONTENTS)
	var topics []Topic

	c.Find(bson.M{"content.createdby": u.Id_, "content.type": TypeTopic}).Sort("-content.createdat").Limit(10).All(&topics)

	return &topics
}

// 用户的最近10个回复
func (u *User) LatestReplies(db *mgo.Database) *[]Comment {
	c := db.C(COMMENTS)
	var replies []Comment

	c.Find(bson.M{"createdby": u.Id_, "type": TypeTopic}).Sort("-createdat").Limit(10).All(&replies)

	return &replies
}

// 是否被某人关注
func (u *User) IsFollowedBy(who string) bool {
	for _, username := range u.Fans {
		if username == who {
			return true
		}
	}

	return false
}

// 是否关注某人
func (u *User) IsFans(who string) bool {
	for _, username := range u.Follow {
		if username == who {
			return true
		}
	}

	return false
}

// HomeLink returns the user home page link.
func (u *User) HomeLink() string {
	return "/user/" + u.Name
}

// AvatarLink returns user gravatar link.
func (u *User) AvatarLink() string {
	if setting.DisableGravatar {
		return "/img/avatar_default.jpg"
	} else if setting.Service.EnableCacheAvatar {
		return "/avatar/" + u.Avatar
	}
	return "//1.gravatar.com/avatar/" + u.Avatar
}


// EncodePasswd encodes password to safe format.
func (u *User) EncodePasswd() {
	newPasswd := base.PBKDF2([]byte(u.Password), []byte(u.Salt), 10000, 50, sha256.New)
	u.Password = fmt.Sprintf("%x", newPasswd)
}

// ValidtePassword checks if given password matches the one belongs to the user.
func (u *User) ValidtePassword(passwd string) bool {
	newUser := &User{Password: passwd, Salt: u.Salt}
	newUser.EncodePasswd()
	return u.Password == newUser.Password
}


// IsUserExist checks if given user name exist,
// the user name should be noncased unique.
func IsUserExist(db *mgo.Database,name string) (bool, error) {
	if len(name) == 0 {
		return false, nil
	}
	c:=db.C(USERS)
	count,err:=c.Find(bson.M{"lowername":strings.ToLower(name)}).Count()
	return count > 0, err
}

// IsEmailUsed returns true if the e-mail has been used.
func IsEmailUsed(db *mgo.Database,email string) (bool, error) {
	if len(email) == 0 {
		return false, nil
	}
	c:=db.C(USERS)
	count,err:=c.Find(bson.M{"email":strings.ToLower(email)}).Count()
	return count > 0, err
}

// GetUserSalt returns a user salt token
func GetUserSalt() string {
	return base.GetRandomString(10)
}



// IsLegalName returns false if name contains illegal characters.
func IsLegalName(repoName string) bool {
	repoName = strings.ToLower(repoName)
	for _, char := range illegalEquals {
		if repoName == char {
			return false
		}
	}
	return true
}
// CreateUser creates record of a new user.
func CreateUser(db *mgo.Database,u *User) error {
	if !IsLegalName(u.Name) {
		return ErrUserNameIllegal
	}

	isExist, err := IsUserExist(u.Name)
	if err != nil {
		return err
	} else if isExist {
		return ErrUserAlreadyExist
	}

	isExist, err = IsEmailUsed(u.Email)
	if err != nil {
		return err
	} else if isExist {
		return ErrEmailAlreadyUsed
	}

	u.Id_=bson.NewObjectId()

	u.LowerName = strings.ToLower(u.Name)
	u.Avatar = base.EncodeMd5(u.Email)
	u.AvatarEmail = u.Email
	u.Rands = GetUserSalt()
	u.Salt = GetUserSalt()
	u.EncodePasswd()

	lock.Lock()
	userIndex++
	u.Index=userIndex
	lock.Unlock()

	if u.Index == 1 {
		u.IsAdmin = true
		u.IsActive = true
	}

	c:=db.C(USERS)
	return c.Insert(u)
}

// CountUsers returns number of users.
func CountUsers(db *mgo.Database) int64 {
	c:=db.C(USERS)
	count,_:=c.Find(bson.M{"isactive",true}).Count()
	return count
}

// GetUsers returns given number of user objects with offset.
func GetUsers(db *mgo.Database,num, offset int) ([]*User, error) {
	users := make([]*User, 0, num)
	c:=db.C(USERS)
	err:=c.Find(bson.M{"isactive",true}).Skip(offset).Limit(num).All(&users)
	if err==mgo.ErrNotFound {
		return nil, ErrUserNotExist
	}else if err != nil {
		return nil, err
	}
	return users, err
}

// GetUserById returns the user object by given ID if exists.
func GetUserById(db *mgo.Database,id  bson.ObjectId) (*User, error) {
	u := new(User)
	c:=db.C(USERS)
	err:=c.FindId(id).One(u)

	if err==mgo.ErrNotFound {
		return nil, ErrUserNotExist
	}else if err != nil {
		return nil, err
	}
	return u, nil
}

// GetUserByName returns the user object by given name if exists.
func GetUserByName(db *mgo.Database,name string) (*User, error) {
	if len(name) == 0 {
		return nil, ErrUserNotExist
	}
	u:=new(User)
	c:=db.C(USERS)
	err:=c.Find(bson.M{"lowername": strings.ToLower(name)}).One(u)

	if err==mgo.ErrNotFound {
		return nil, ErrUserNotExist
	}else if err != nil {
		return nil, err
	}
	return u, nil
}

// get user by erify code
func getVerifyUser(code string) (user *User) {
	if len(code) <= base.TimeLimitCodeLength {
		return nil
	}

	// use tail hex username query user
	hexStr := code[base.TimeLimitCodeLength:]
	if b, err := hex.DecodeString(hexStr); err == nil {
		if user, err = GetUserByName(string(b)); user != nil {
			return user
		}
		log.Error(4, "user.getVerifyUser: %v", err)
	}

	return nil
}

// verify active code when active account
func VerifyUserActiveCode(code string) (user *User) {
	minutes := setting.Service.ActiveCodeLives

	if user = getVerifyUser(code); user != nil {
		// time limit code
		prefix := code[:base.TimeLimitCodeLength]
		data := com.ToStr(user.Id) + user.Email + user.LowerName + user.Password + user.Rands

		if base.VerifyTimeLimitCode(data, minutes, prefix) {
			return user
		}
	}
	return nil
}
