package models

import (

	"html/template"
	"time"

	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

// 评论
type Comment struct {
	Id_       bson.ObjectId `bson:"_id"`
	Type      int
	ContentId bson.ObjectId
	Markdown  string
	Html      template.HTML
	CreatedBy bson.ObjectId
	CreatedAt time.Time
	UpdatedBy string
	UpdatedAt time.Time
}

// 评论人
func (c *Comment) Creater(db *mgo.Database) *User {
	c_ := db.C(USERS)
	user := User{}
	c_.Find(bson.M{"_id": c.CreatedBy}).One(&user)

	return &user
}

// 是否有权删除评论，只允许管理员删除
func (c *Comment) CanDelete(username string, db *mgo.Database) bool {
	var user User
	c_ := db.C(USERS)
	err := c_.Find(bson.M{"username": username}).One(&user)
	if err != nil {
		return false
	}
	return user.IsSuperuser
}

// 主题
func (c *Comment) Topic(db *mgo.Database) *Topic {
	// 内容
	var topic Topic
	c_ := db.C(CONTENTS)
	c_.Find(bson.M{"_id": c.ContentId, "content.type": TypeTopic}).One(&topic)
	return &topic
}
