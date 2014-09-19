package models

import (

	"html/template"
	"time"

	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

// 通用的内容
type Content struct {
	Id_          bson.ObjectId // 同外层Id_
	Type         int
	Title        string
	Markdown     string
	Html         template.HTML
	CommentCount int
	Hits         int // 点击数量
	CreatedAt    time.Time
	CreatedBy    bson.ObjectId
	UpdatedAt    time.Time
	UpdatedBy    string
}

func (c *Content) Creater(db *mgo.Database) *User {
	c_ := db.C(USERS)
	user := User{}
	c_.Find(bson.M{"_id": c.CreatedBy}).One(&user)

	return &user
}

func (c *Content) Updater(db *mgo.Database) *User {
	if c.UpdatedBy == "" {
		return nil
	}

	c_ := db.C(USERS)
	user := User{}
	c_.Find(bson.M{"_id": bson.ObjectIdHex(c.UpdatedBy)}).One(&user)

	return &user
}

func (c *Content) Comments(db *mgo.Database) *[]Comment {
	c_ := db.C(COMMENTS)
	var comments []Comment

	c_.Find(bson.M{"contentid": c.Id_}).All(&comments)

	return &comments
}

// 只能收藏未收藏过的主题
func (c *Content) CanCollect(username string, db *mgo.Database) bool {
	var user User
	c_ := db.C(USERS)
	err := c_.Find(bson.M{"username": username}).One(&user)
	if err != nil {
		return false
	}
	has := false
	for _, v := range user.TopicsCollected {
		if v.TopicId == c.Id_.Hex() {
			has = true
		}
	}
	return !has
}

// 是否有权编辑主题
func (c *Content) CanEdit(username string, db *mgo.Database) bool {
	var user User
	c_ := db.C(USERS)
	err := c_.Find(bson.M{"username": username}).One(&user)
	if err != nil {
		return false
	}

	if user.IsSuperuser {
		return true
	}

	return c.CreatedBy == user.Id_
}

func (c *Content) CanDelete(username string, db *mgo.Database) bool {
	var user User
	c_ := db.C(USERS)
	err := c_.Find(bson.M{"username": username}).One(&user)
	if err != nil {
		return false
	}

	return user.IsSuperuser
}

// 主题
type Topic struct {
	Content
	Id_             bson.ObjectId `bson:"_id"`
	NodeId          bson.ObjectId
	LatestReplierId string
	LatestRepliedAt time.Time
}

// 主题所属节点
func (t *Topic) Node(db *mgo.Database) *Node {
c := db.C(NODES)
node := Node{}
c.Find(bson.M{"_id": t.NodeId}).One(&node)

return &node
}

// 主题链接
func (t *Topic) Link(id bson.ObjectId) string {
	return "http://golangtc.com/t/" + id.Hex()

}

//格式化日期
func (t *Topic) Format(tm time.Time) string {
	return tm.Format(time.RFC822)
}

// 主题的最近的一个回复
func (t *Topic) LatestReplier(db *mgo.Database) *User {
	if t.LatestReplierId == "" {
		return nil
	}

	c := db.C(USERS)
	user := User{}

	err := c.Find(bson.M{"_id": bson.ObjectIdHex(t.LatestReplierId)}).One(&user)

	if err != nil {
		return nil
	}

	return &user
}

