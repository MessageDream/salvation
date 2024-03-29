// Copyright 2014 The Gogs Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package admin

import (
	"strings"

	"github.com/Unknwon/com"

	"github.com/MessageDream/salvation/models"
	"github.com/MessageDream/salvation/modules/auth"
	"github.com/MessageDream/salvation/modules/base"
	"github.com/MessageDream/salvation/modules/log"
	"github.com/MessageDream/salvation/modules/middleware"
)

const (
	USERS     base.TplName = "admin/user/list"
	USER_NEW  base.TplName = "admin/user/new"
	USER_EDIT base.TplName = "admin/user/edit"
)

func pagination(ctx *middleware.Context, count int64, pageNum int) int {
	p := com.StrTo(ctx.Query("p")).MustInt()
	if p < 1 {
		p = 1
	}
	curCount := int64((p-1)*pageNum + pageNum)
	if curCount > count {
		p = int(count) / pageNum
	} else if count > curCount {
		ctx.Data["NextPageNum"] = p + 1
	}
	if p > 1 {
		ctx.Data["LastPageNum"] = p - 1
	}
	return p
}

func Users(ctx *middleware.Context) {
	ctx.Data["Title"] = ctx.Tr("admin.users")
	ctx.Data["PageIsAdmin"] = true
	ctx.Data["PageIsAdminUsers"] = true

	pageNum := 50
	p := pagination(ctx, models.CountUsers(), pageNum)

	var err error
	ctx.Data["Users"], err = models.GetUsers(pageNum, (p-1)*pageNum)
	if err != nil {
		ctx.Handle(500, "GetUsers", err)
		return
	}
	ctx.HTML(200, USERS)
}

func NewUser(ctx *middleware.Context) {
	ctx.Data["Title"] = ctx.Tr("admin.users.new_account")
	ctx.Data["PageIsAdmin"] = true
	ctx.Data["PageIsAdminUsers"] = true

	auths, err := models.GetAuths()
	if err != nil {
		ctx.Handle(500, "GetAuths", err)
		return
	}
	ctx.Data["LoginSources"] = auths
	ctx.HTML(200, USER_NEW)
}

func NewUserPost(ctx *middleware.Context, form auth.RegisterForm) {
	ctx.Data["Title"] = ctx.Tr("admin.users.new_account")
	ctx.Data["PageIsAdmin"] = true
	ctx.Data["PageIsAdminUsers"] = true

	if ctx.HasError() {
		ctx.HTML(200, USER_NEW)
		return
	}

	if form.Password != form.Retype {
		ctx.Data["Err_Password"] = true
		ctx.RenderWithErr(ctx.Tr("form.password_not_match"), USER_NEW, &form)
		return
	}

	u := &models.User{
		Name:      form.UserName,
		Email:     form.Email,
		Passwd:    form.Password,
		IsActive:  true,
		LoginType: models.PLAIN,
	}

	if len(form.LoginType) > 0 {
		// NOTE: need rewrite.
		fields := strings.Split(form.LoginType, "-")
		tp, _ := com.StrTo(fields[0]).Int()
		u.LoginType = models.LoginType(tp)
		u.LoginSource, _ = com.StrTo(fields[1]).Int64()
		u.LoginName = form.LoginName
	}

	if err := models.CreateUser(u); err != nil {
		switch err {
		case models.ErrUserAlreadyExist:
			ctx.Data["Err_UserName"] = true
			ctx.RenderWithErr(ctx.Tr("form.username_been_taken"), USER_NEW, &form)
		case models.ErrEmailAlreadyUsed:
			ctx.Data["Err_Email"] = true
			ctx.RenderWithErr(ctx.Tr("form.email_been_used"), USER_NEW, &form)
		case models.ErrUserNameIllegal:
			ctx.Data["Err_UserName"] = true
			ctx.RenderWithErr(ctx.Tr("form.illegal_username"), USER_NEW, &form)
		default:
			ctx.Handle(500, "CreateUser", err)
		}
		return
	}
	log.Trace("Account created by admin(%s): %s", ctx.User.Name, u.Name)
	ctx.Redirect("/admin/users")
}

func EditUser(ctx *middleware.Context) {
	ctx.Data["Title"] = ctx.Tr("admin.users.edit_account")
	ctx.Data["PageIsAdmin"] = true
	ctx.Data["PageIsAdminUsers"] = true

	uid := com.StrTo(ctx.Params(":userid")).MustInt64()
	if uid == 0 {
		ctx.Handle(404, "EditUser", nil)
		return
	}

	u, err := models.GetUserById(uid)
	if err != nil {
		ctx.Handle(500, "GetUserById", err)
		return
	}

	ctx.Data["User"] = u
	auths, err := models.GetAuths()
	if err != nil {
		ctx.Handle(500, "GetAuths", err)
		return
	}
	ctx.Data["LoginSources"] = auths
	ctx.HTML(200, USER_EDIT)
}

func EditUserPost(ctx *middleware.Context, form auth.AdminEditUserForm) {
	ctx.Data["Title"] = ctx.Tr("admin.users.edit_account")
	ctx.Data["PageIsAdmin"] = true
	ctx.Data["PageIsAdminUsers"] = true

	uid := com.StrTo(ctx.Params(":userid")).MustInt64()
	if uid == 0 {
		ctx.Handle(404, "EditUser", nil)
		return
	}

	u, err := models.GetUserById(uid)
	if err != nil {
		ctx.Handle(500, "GetUserById", err)
		return
	}

	if ctx.HasError() {
		ctx.HTML(200, USER_EDIT)
		return
	}

	// NOTE: need password length check?
	if len(form.Passwd) > 0 {
		u.Passwd = form.Passwd
		u.Salt = models.GetUserSalt()
		u.EncodePasswd()
	}

	u.Email = form.Email
	u.Website = form.Website
	u.Location = form.Location
	if len(form.Avatar) == 0 {
		form.Avatar = form.Email
	}
	u.Avatar = base.EncodeMd5(form.Avatar)
	u.AvatarEmail = form.Avatar
	u.IsActive = form.Active
	u.IsAdmin = form.Admin
	if err := models.UpdateUser(u); err != nil {
		ctx.Handle(500, "UpdateUser", err)
		return
	}
	log.Trace("Account profile updated by admin(%s): %s", ctx.User.Name, u.Name)

	ctx.Data["User"] = u
	ctx.Flash.Success(ctx.Tr("admin.users.update_profile_success"))
	ctx.Redirect("/admin/users/" + ctx.Params(":userid"))
}

func DeleteUser(ctx *middleware.Context) {
	uid := com.StrTo(ctx.Params(":userid")).MustInt64()
	if uid == 0 {
		ctx.Handle(404, "DeleteUser", nil)
		return
	}

	u, err := models.GetUserById(uid)
	if err != nil {
		ctx.Handle(500, "GetUserById", err)
		return
	}

	if err = models.DeleteUser(u); err != nil {
		switch err {
		case models.ErrUserOwnRepos:
			ctx.Flash.Error(ctx.Tr("admin.users.still_own_repo"))
			ctx.Redirect("/admin/users/" + ctx.Params(":userid"))
		default:
			ctx.Handle(500, "DeleteUser", err)
		}
		return
	}
	log.Trace("Account deleted by admin(%s): %s", ctx.User.Name, u.Name)
	ctx.Redirect("/admin/users")
}
