package models

import (
	"fmt"
	"log"
	"os"
	"path"

	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"

	"github.com/MessageDream/salvation/modules/setting"
)

const (
	TypeTopic     = 'T'
	TypeArticle   = 'A'
	DefaultAvatar = "gopher_teal.jpg"

	ADS                = "ads"
	ARTICLE_CATEGORIES = "articlecategories"
	COMMENTS           = "comments"
	CONTENTS           = "contents"
	NODES              = "nodes"
	LINK_EXCHANGES     = "link_exchanges"
	STATUS             = "status"
	USERS              = "users"
	LOGIN_SOURCE       = "LoginSource"
	OAUTH_2			   = "Oauth2"

	DB_LOG_MODE_DEBUG = "log_debug"
)

var (
	DbCfg struct {
		Type, Host, Port, Name, User, Pwd, Path, SslMode, LogMode string
	}
)

func LoadModelsConfig() {
	DbCfg.Type = setting.Cfg.MustValue("database", "DB_TYPE")
	DbCfg.Host = setting.Cfg.MustValue("database", "HOST")
	DbCfg.Name = setting.Cfg.MustValue("database", "NAME")
	DbCfg.User = setting.Cfg.MustValue("database", "USER")
	if len(DbCfg.Pwd) == 0 {
		DbCfg.Pwd = setting.Cfg.MustValue("database", "PASSWD")
	}
	DbCfg.SslMode = setting.Cfg.MustValue("database", "SSL_MODE")
	DbCfg.LogMode = setting.Cfg.MustValue("database", "LOG_MODE")
	DbCfg.Path = setting.Cfg.MustValue("database", "PATH", "data/gogs.db")
}

func GetSession() (*mgo.Session, error) {
	cnnstr := fmt.Sprintf("mongodb://%s:%s@%s:%s/%s",
		DbCfg.User, DbCfg.Pwd, DbCfg.Host, DbCfg.Port, DbCfg.Name)
	return mgo.Dial(cnnstr)
}

func GetDb(se *mgo.Session) *mgo.Database {
	return se.DB(DbCfg.Name)
}

// 状态,MongoDB中只存储一个状态
type Status struct {
	Id_        bson.ObjectId `bson:"_id"`
	UserCount  int
	TopicCount int
	ReplyCount int
	UserIndex  int
}

func SetDB() (err error) {

	if DbCfg.LogMode == DB_LOG_MODE_DEBUG {
		logPath := path.Join(setting.LogRootPath, "mongo.log")
		os.MkdirAll(path.Dir(logPath), os.ModePerm)

		f, err := os.Create(logPath)
		if err != nil {
			return fmt.Errorf("models.init(fail to create xorm.log): %v", err)
		}
		mgo.SetLogger(log.New(f, "mongodb", log.Ltime))
		mgo.SetDebug(true)
	}

	return nil
}

func Ping() error {
	se,err:=GetSession()
	if err!=nil{
		return err
	}
	defer se.Close()
	return se.Ping()
}
