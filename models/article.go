package models

import (
	mgo"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

// 文章分类
type ArticleCategory struct {
	Id_  bson.ObjectId `bson:"_id"`
	Name string
}

// 文章
type Article struct {
	Content
	Id_            bson.ObjectId `bson:"_id"`
	CategoryId     bson.ObjectId
	OriginalSource string
	OriginalUrl    string
}

// 主题所属类型
func (a *Article) Category(db *mgo.Database) *ArticleCategory {
	c := db.C(ARTICLE_CATEGORIES)
	category := ArticleCategory{}
	c.Find(bson.M{"_id": a.CategoryId}).One(&category)

	return &category
}
