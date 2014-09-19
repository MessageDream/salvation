// Copyright 2014 The Gogs Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package models

import (
	"errors"
	"time"

	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

type OauthType int

const (
	GITHUB OauthType = iota + 1
	GOOGLE
	TWITTER
	QQ
	WEIBO
	BITBUCKET
	FACEBOOK
)

var (
	ErrOauth2RecordNotExist = errors.New("OAuth2 record does not exist")
	ErrOauth2NotAssociated  = errors.New("OAuth2 is not associated with user")
)

type Oauth2 struct {
	Id_             bson.ObjectId `bson:"_id"`
	Uid                bson.ObjectId
	User              *User
	Type              int        // twitter,github,google...
	Identity          string    // id..
	Token             string
	Created           time.Time
	Updated           time.Time
	HasRecentActivity bool
}

func BindUserOauth2(db *mgo.Database,userId, oauthId bson.ObjectId) error {
	c:=db.C(OAUTH_2)
	return c.UpdateId(oauthId,&Oauth2{Uid: userId})
}

func AddOauth2(db *mgo.Database,oa *Oauth2) error {
	oa.Id_=bson.NewObjectId()
	c:=db.C(OAUTH_2)
	return c.Insert(oa)
}

func GetOauth2(db *mgo.Database,identity string) (oa *Oauth2, err error) {
	oa = new(Oauth2)
	c:=db.C(OAUTH_2)
	err=c.Find(bson.M{"identity",identity}).One(oa)

	if err==mgo.ErrNotFound {
		return nil, ErrOauth2RecordNotExist
	}else if err != nil {
		return
	} else if oa.Uid == -1 {
		return oa, ErrOauth2NotAssociated
	}
	oa.User, err = GetUserById(oa.Uid)
	return oa, err
}

func GetOauth2ById(db *mgo.Database,id bson.ObjectId) (oa *Oauth2, err error) {
	oa = new(Oauth2)
	c:=db.C(OAUTH_2)

	err=c.FindId(id).One(oa)
	if err==mgo.ErrNotFound {
		return nil, ErrOauth2RecordNotExist
	}
	return
}

// UpdateOauth2 updates given OAuth2.
func UpdateOauth2(db *mgo.Database,oa *Oauth2) error {
	c:=db.C(OAUTH_2)
	err :=c.UpdateId(oa.Id_,oa)
	return err
}

// GetOauthByUserId returns list of oauthes that are releated to given user.
func GetOauthByUserId(db *mgo.Database,uid bson.ObjectId) ([]*Oauth2, error) {
	socials := make([]*Oauth2, 0, 5)
	c:=db.C(OAUTH_2)
	err := c.Find(bson.M{"uid":uid}).All(socials)
	if err==mgo.ErrNotFound {
		return nil, ErrOauth2RecordNotExist
	}else if err != nil {
		return socials,err
	}

	for _, social := range socials {
		social.HasRecentActivity = social.Updated.Add(7 * 24 * time.Hour).After(time.Now())
	}
	return socials, err
}

// DeleteOauth2ById deletes a oauth2 by ID.
func DeleteOauth2ById(db *mgo.Database,id bson.ObjectId) error {
	c:=db.C(OAUTH_2)
	return c.RemoveId(id)
}

// CleanUnbindOauth deletes all unbind OAuthes.
func CleanUnbindOauth(db *mgo.Database) error {
	c:=db.C(OAUTH_2)
	_,err:=c.RemoveAll(bson.M{"uid":bson.Undefined})
	return err
}
