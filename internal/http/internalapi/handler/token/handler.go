package token

import (
	"context"
	"net/http"
	"time"

	"github.com/fivemanage/lite/api"
	"github.com/fivemanage/lite/internal/http/appctx"
	"github.com/fivemanage/lite/internal/http/httputil"
	"github.com/fivemanage/lite/internal/http/validator"
	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
)

// createTokenHandler godoc
// @Summary      Create token
// @Description  Create a new API token for the current organization
// @Tags         token
// @Accept       json
// @Produce      json
// @Param        organizationId  path      string  true  "Organization ID"
// @Param        data            body      api.CreateTokenRequest  true  "Create Token Request"
// @Success      200             {object}  httputil.ResponseData{data=api.CreateTokenResponse}
// @Failure      500             {object}  httputil.ErrorResponseData
// @Router       /dash/{organizationId}/token [post]
func (r *handler) createTokenHandler(c echo.Context) error {
	cc := c.(*appctx.Context)
	ctx, cancel := context.WithTimeout(cc.Request().Context(), 5*time.Second)
	defer cancel()

	organizationID := cc.Param("organizationId")

	var data api.CreateTokenRequest
	if err := validator.BindAndValidate(cc, &data); err != nil {
		logrus.WithError(err).Error("failed to bind and validate token")
		return cc.JSON(500, httputil.ErrorResponse(err.Error()))
	}

	data.OrganizationID = organizationID
	apiToken, err := r.tokenService.CreateToken(ctx, &data)
	if err != nil {
		return cc.JSON(500, httputil.ErrorResponse(err.Error()))
	}

	// we return the apiToken as a one-time thing
	tokenResponse := &api.CreateTokenResponse{
		Token: apiToken,
	}

	return cc.JSON(http.StatusOK, httputil.Response(tokenResponse))
}

// listTokensHandler godoc
// @Summary      List tokens
// @Description  List all API tokens for the current organization
// @Tags         token
// @Produce      json
// @Param        organizationId  path      string  true  "Organization ID"
// @Success      200             {object}  httputil.ResponseData{data=[]api.ListTokensResponse}
// @Failure      500             {object}  httputil.ErrorResponseData
// @Router       /dash/{organizationId}/token [get]
func (r *handler) listTokensHandler(c echo.Context) error {
	cc := c.(*appctx.Context)
	ctx := cc.Request().Context()

	organizationID := cc.Param("organizationId")

	tokens, err := r.tokenService.ListTokens(ctx, organizationID)
	if err != nil {
		return cc.JSON(500, httputil.ErrorResponse(err.Error()))
	}

	return cc.JSON(http.StatusOK, httputil.Response(tokens))
}

// deleteTokenHandler godoc
// @Summary      Delete token
// @Description  Delete an API token by ID
// @Tags         token
// @Produce      json
// @Param        organizationId  path      string  true  "Organization ID"
// @Param        id              path      string  true  "Token ID"
// @Success      200             {object}  httputil.ResponseData{data=nil}
// @Failure      500             {object}  httputil.ErrorResponseData
// @Router       /dash/{organizationId}/token/{id} [delete]
func (r *handler) deleteTokenHandler(c echo.Context) error {
	cc := c.(*appctx.Context)
	ctx := cc.Request().Context()

	tokenID := cc.Param("id")

	err := r.tokenService.DeleteToken(ctx, tokenID)
	if err != nil {
		return cc.JSON(500, httputil.ErrorResponse(err.Error()))
	}

	return cc.JSON(http.StatusOK, httputil.Response(nil))
}
