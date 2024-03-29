// Copyright 2014 The Gogs Authors. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package admin

import (
	"github.com/MessageDream/salvation/models"
	"github.com/MessageDream/salvation/modules/base"
	"github.com/MessageDream/salvation/modules/middleware"
)

const (
	ORGS base.TplName = "admin/org/list"
)

func Organizations(ctx *middleware.Context) {
	ctx.Data["Title"] = ctx.Tr("admin.orgs")
	ctx.Data["PageIsAdmin"] = true
	ctx.Data["PageIsAdminOrganizations"] = true

	pageNum := 50
	p := pagination(ctx, models.CountOrganizations(), pageNum)

	var err error
	ctx.Data["Orgs"], err = models.GetOrganizations(pageNum, (p-1)*pageNum)
	if err != nil {
		ctx.Handle(500, "GetUsers", err)
		return
	}
	ctx.HTML(200, ORGS)
}
