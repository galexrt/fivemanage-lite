package member

import (
	"net/http"

	"github.com/fivemanage/lite/internal/http/appctx"
	"github.com/fivemanage/lite/internal/http/httputil"
	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
)

// listMembersHandler godoc
// @Summary      List members
// @Description  List members of an organization
// @Tags         member
// @Produce      json
// @Param        id   path      string  true  "Organization ID"
// @Success      200  {object}  httputil.ResponseData{data=[]api.OrganizationMember}
// @Failure      500  {object}  httputil.ErrorResponseData
// @Router       /dash/organization/{id}/member [get]
func (r *handler) listMembersHandler(c echo.Context) error {
	cc := c.(*appctx.Context)
	ctx := cc.Request().Context()

	id := cc.Param("id")

	members, err := r.memberService.ListMembers(ctx, id)
	if err != nil {
		logrus.WithError(err).Error("failed to list members")
		return cc.JSON(500, httputil.ErrorResponse(err.Error()))
	}

	return cc.JSON(http.StatusOK, httputil.Response(members))
}
