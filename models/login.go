// Copyright 2014 The Gogs Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package models

import (
	"crypto/tls"
	"errors"
	"fmt"
	"net/smtp"
	"strings"
	"time"

	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"

	"github.com/MessageDream/salvation/modules/auth/ldap"
	"github.com/MessageDream/salvation/modules/log"
	"github.com/MessageDream/salvation/modules/setting"

)

type LoginType int

const (
	NOTYPE LoginType = iota
	PLAIN
	LDAP
	SMTP
)

var (
	ErrAuthenticationAlreadyExist = errors.New("Authentication already exist")
	ErrAuthenticationNotExist     = errors.New("Authentication does not exist")
	ErrAuthenticationUserUsed     = errors.New("Authentication has been used by some users")
)

var LoginTypes = map[LoginType]string{
	LDAP: "LDAP",
	SMTP: "SMTP",
}


type LDAPConfig struct {
	ldap.Ldapsource
}


type SMTPConfig struct {
	Auth string
	Host string
	Port int
	TLS  bool
}


type LoginSource struct {
	Id_               bson.ObjectId `bson:"_id"`
	Id                int64
	Type              LoginType
	Name              string
	IsActived         bool
	LDAPCfg           LDAPConfig
	SMTPCfg			  SMTPConfig
	AllowAutoRegister bool
	Created           time.Time
	Updated           time.Time
}

func (source *LoginSource) TypeString() string {
	return LoginTypes[source.Type]
}

func (source *LoginSource) LDAP() LDAPConfig {
	return source.Cfg.(LDAPConfig)
}

func (source *LoginSource) SMTP() SMTPConfig {
	return source.Cfg.(SMTPConfig)
}

func CreateSource(db *mgo.Database,source *LoginSource) error {
	source.Id_=bson.NewObjectId()
	c:=db.C(LOGIN_SOURCE)
	err:=c.Insert(source)
	return err
}

func GetAuths(db *mgo.Database) ([]*LoginSource, error) {
	var auths = make([]*LoginSource, 0, 5)
	c:=db.C(LOGIN_SOURCE)
	err:=c.Find(nil).All(&auths)
	if err==mgo.ErrNotFound {
		return nil, ErrAuthenticationNotExist
	}else if err != nil {
		return nil, err
	}
	return auths, err
}

func GetLoginSourceById(db *mgo.Database,id int64) (*LoginSource, error) {
	source := new(LoginSource)
	c:=db.C(LOGIN_SOURCE)
	err :=c.Find(bson.M{"id",id}).One(source)

	if err==mgo.ErrNotFound {
		return nil, ErrAuthenticationNotExist
	}else if err != nil {
		return nil, err
	}
	return source, nil
}

func UpdateSource(db *mgo.Database,source *LoginSource) error {
	c:=db.C(LOGIN_SOURCE)
	err :=c.UpdateId(source.Id_,source)
	return err
}

func DelLoginSource(db *mgo.Database,source *LoginSource) error {
	c:=db.C(USERS)
	cnt, err := c.Find(bson.M{"loginsource":source.Id}).Count()
	if err != nil {
		return err
	}
	if cnt > 0 {
		return ErrAuthenticationUserUsed
	}
	c:=db.C(LOGIN_SOURCE)
	err=c.RemoveId(source.Id_)
	return err
}

// UserSignIn validates user name and password.
func UserSignIn(db *mgo.Database,uname, passwd string) (*User, error) {
	var u *User
	c:=db.C(USERS)
	var err error
	if strings.Contains(uname, "@") {
		err=c.Find(bson.M{"email":uname}).One(u)
	} else {
		err=c.Find(bson.M{"lowername":uname}).One(u)
	}

	if err != nil && err!=mgo.ErrNotFound{
		return nil, err
	}

	if u.LoginType == NOTYPE {
		if u.Index>0 {
			u.LoginType = PLAIN
		} else {
			return nil, ErrUserNotExist
		}
	}

	// For plain login, user must exist to reach this line.
	// Now verify password.
	if u.LoginType == PLAIN {
		newUser := &User{Password: passwd, Salt: u.Salt}
		newUser.EncodePasswd()
		if u.Password != newUser.Password {
			return nil, ErrUserNotExist
		}
		return u, nil
	} else {
		c:=db.C(LOGIN_SOURCE)
		if err==mgo.ErrNotFound {
			var sources []LoginSource

			err=c.Find(bson.M{"isactive":true,"allowautoregister":true}).All(&sources)
			if err != nil {
				return nil, err
			}

			for _, source := range sources {
				if source.Type == LDAP {
					u, err := LoginUserLdapSource(db,nil, uname, passwd,
						source.Id, source.Cfg.(*LDAPConfig), true)
					if err == nil {
						return u, nil
					}
					log.Warn("Fail to login(%s) by LDAP(%s): %v", uname, source.Name, err)
				} else if source.Type == SMTP {
					u, err := LoginUserSMTPSource(db,nil, uname, passwd,
						source.Id, source.Cfg.(*SMTPConfig), true)
					if err == nil {
						return u, nil
					}
					log.Warn("Fail to login(%s) by SMTP(%s): %v", uname, source.Name, err)
				}
			}

			return nil, ErrUserNotExist
		}

		var source LoginSource
		err=c.Find(bson.M{"id":u.LoginSource}).One(&source)

		if err==mgo.ErrNotFound {
			return nil, ErrAuthenticationNotExist
		}else if err != nil {
			return nil, err
		}else if !source.IsActived {
			return nil, ErrLoginSourceNotActived
		}

		switch u.LoginType {
		case LDAP:
			return LoginUserLdapSource(db,u, u.LoginName, passwd,
				source.Id, source.Cfg.(*LDAPConfig), false)
		case SMTP:
			return LoginUserSMTPSource(db,u, u.LoginName, passwd,
				source.Id, source.Cfg.(*SMTPConfig), false)
		}
		return nil, ErrUnsupportedLoginType
	}
}

// Query if name/passwd can login against the LDAP direcotry pool
// Create a local user if success
// Return the same LoginUserPlain semantic
func LoginUserLdapSource(db *mgo.Database,u *User, name, passwd string, sourceId int64, cfg *LDAPConfig, autoRegister bool) (*User, error) {
	mail, logged := cfg.Ldapsource.SearchEntry(name, passwd)
	if !logged {
		// user not in LDAP, do nothing
		return nil, ErrUserNotExist
	}
	if !autoRegister {
		return u, nil
	}

	// fake a local user creation
	u = &User{
		LowerName:   strings.ToLower(name),
		UserName:        strings.ToLower(name),
		LoginType:   LDAP,
		LoginSource: sourceId,
		IsActive:    true,
		Password:      passwd,
		Email:       mail,
	}

	err := CreateUser(db,u)
	return u, err
}

type loginAuth struct {
	username, password string
}

func LoginAuth(username, password string) smtp.Auth {
	return &loginAuth{username, password}
}

func (a *loginAuth) Start(server *smtp.ServerInfo) (string, []byte, error) {
	return "LOGIN", []byte(a.username), nil
}

func (a *loginAuth) Next(fromServer []byte, more bool) ([]byte, error) {
	if more {
		switch string(fromServer) {
		case "Username:":
			return []byte(a.username), nil
		case "Password:":
			return []byte(a.password), nil
		}
	}
	return nil, nil
}

var (
	SMTP_PLAIN = "PLAIN"
	SMTP_LOGIN = "LOGIN"
	SMTPAuths  = []string{SMTP_PLAIN, SMTP_LOGIN}
)

func SmtpAuth(host string, port int, a smtp.Auth, useTls bool) error {
	c, err := smtp.Dial(fmt.Sprintf("%s:%d", host, port))
	if err != nil {
		return err
	}
	defer c.Close()

	if err = c.Hello(setting.AppName); err != nil {
		return err
	}

	if useTls {
		if ok, _ := c.Extension("STARTTLS"); ok {
			config := &tls.Config{ServerName: host}
			if err = c.StartTLS(config); err != nil {
				return err
			}
		} else {
			return errors.New("SMTP server unsupports TLS")
		}
	}

	if ok, _ := c.Extension("AUTH"); ok {
		if err = c.Auth(a); err != nil {
			return err
		}
		return nil
	}
	return ErrUnsupportedLoginType
}

// Query if name/passwd can login against the LDAP direcotry pool
// Create a local user if success
// Return the same LoginUserPlain semantic
func LoginUserSMTPSource(db *mgo.Database,u *User, name, passwd string, sourceId int64, cfg *SMTPConfig, autoRegister bool) (*User, error) {
	var auth smtp.Auth
	if cfg.Auth == SMTP_PLAIN {
		auth = smtp.PlainAuth("", name, passwd, cfg.Host)
	} else if cfg.Auth == SMTP_LOGIN {
		auth = LoginAuth(name, passwd)
	} else {
		return nil, errors.New("Unsupported SMTP auth type")
	}

	if err := SmtpAuth(cfg.Host, cfg.Port, auth, cfg.TLS); err != nil {
		if strings.Contains(err.Error(), "Username and Password not accepted") {
			return nil, ErrUserNotExist
		}
		return nil, err
	}

	if !autoRegister {
		return u, nil
	}

	var loginName = name
	idx := strings.Index(name, "@")
	if idx > -1 {
		loginName = name[:idx]
	}
	// fake a local user creation
	u = &User{
		LowerName:   strings.ToLower(loginName),
		UserName:        strings.ToLower(loginName),
		LoginType:   SMTP,
		LoginSource: sourceId,
		IsActive:    true,
		Password:      passwd,
		Email:       name,
	}
	err := CreateUser(db,u)
	return u, err
}
