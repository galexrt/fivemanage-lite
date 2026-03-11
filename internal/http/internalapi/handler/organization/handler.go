package organization

import (
	"net/http"

	"github.com/fivemanage/lite/api"
	"github.com/fivemanage/lite/internal/http/appctx"
	"github.com/fivemanage/lite/internal/http/httputil"
	"github.com/fivemanage/lite/internal/http/validator"
	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
)

// createOrganizationHandler godoc
// @Summary      Create organization
// @Description  Create a new organization
// @Tags         organization
// @Accept       json
// @Produce      json
// @Param        data  body      api.CreateOrganizationRequest  true  "Create Organization Request"
// @Success      200   {object}  httputil.ResponseData{data=api.Organization}
// @Failure      500   {object}  httputil.ErrorResponseData
// @Router       /dash/organization [post]
func (r *handler) createOrganizationHandler(c echo.Context) error {
	cc := c.(*appctx.Context)
	ctx := cc.Request().Context()

	var data api.CreateOrganizationRequest
	if err := validator.BindAndValidate(cc, &data); err != nil {
		logrus.WithError(err).Error("failed to bind and validate token")
		return cc.JSON(500, httputil.ErrorResponse(err.Error()))
	}

	user := cc.User()

	organization, err := r.organizationService.CreateOrganization(ctx, &data, user.ID)
	if err != nil {
		logrus.WithError(err).Error("failed to create organization")
		return cc.JSON(500, httputil.ErrorResponse(err.Error()))
	}

	return cc.JSON(http.StatusOK, httputil.Response(organization))
}

// listOrganizationsHandler godoc
// @Summary      List organizations
// @Description  List all organizations
// @Tags         organization
// @Produce      json
// @Success      200  {object}  httputil.ResponseData{data=[]api.Organization}
// @Failure      500  {object}  httputil.ErrorResponseData
// @Router       /dash/organization [get]
func (r *handler) listOrganizationsHandler(c echo.Context) error {
	cc := c.(*appctx.Context)
	ctx := cc.Request().Context()

	organizations, err := r.organizationService.ListOrganizations(ctx)
	if err != nil {
		logrus.WithError(err).Error("failed to list organizations")
		return cc.JSON(500, httputil.ErrorResponse(err.Error()))
	}

	return cc.JSON(http.StatusOK, httputil.Response(organizations))
}

// getOrganizationHandler godoc
// @Summary      Get organization
// @Description  Get organization by ID
// @Tags         organization
// @Produce      json
// @Param        id   path      string  true  "Organization ID"
// @Success      200  {object}  httputil.ResponseData{data=api.Organization}
// @Failure      500  {object}  httputil.ErrorResponseData
// @Router       /dash/organization/{id} [get]
func (r *handler) getOrganizationHandler(c echo.Context) error {
	cc := c.(*appctx.Context)
	ctx := cc.Request().Context()

	id := cc.Param("id")

	organization, err := r.organizationService.FindOrganizationByID(ctx, id)
	if err != nil {
		logrus.WithError(err).Error("failed to find organization")
		return cc.JSON(500, httputil.ErrorResponse(err.Error()))
	}

	return cc.JSON(http.StatusOK, httputil.Response(organization))
}

// getOrganizationStatsHandler godoc
// @Summary      Get organization stats
// @Description  Get statistics for an organization
// @Tags         organization
// @Produce      json
// @Param        id   path      string  true  "Organization ID"
// @Success      200  {object}  httputil.ResponseData{data=api.OrganizationStats}
// @Failure      500  {object}  httputil.ErrorResponseData
// @Router       /dash/organization/{id}/stats [get]
func (r *handler) getOrganizationStatsHandler(c echo.Context) error {
	cc := c.(*appctx.Context)
	ctx := cc.Request().Context()

	id := cc.Param("id")

	stats, err := r.organizationService.GetStats(ctx, id)
	if err != nil {
		logrus.WithError(err).Error("failed to get organization stats")
		return cc.JSON(500, httputil.ErrorResponse(err.Error()))
	}

	return cc.JSON(http.StatusOK, httputil.Response(stats))
}
