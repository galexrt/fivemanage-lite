package dataset

import (
	"net/http"

	"github.com/fivemanage/lite/api"
	"github.com/fivemanage/lite/internal/http/appctx"
	"github.com/fivemanage/lite/internal/http/httputil"
	"github.com/fivemanage/lite/internal/http/validator"
	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
)

// createDatasetHandler godoc
// @Summary      Create dataset
// @Description  Create a new dataset in the current organization
// @Tags         dataset
// @Accept       json
// @Produce      json
// @Param        organizationId  path      string  true  "Organization ID"
// @Param        data            body      api.Dataset  true  "Dataset"
// @Success      200             {object}  httputil.ResponseData{data=api.Dataset}
// @Failure      500             {object}  httputil.ErrorResponseData
// @Router       /dash/{organizationId}/dataset [post]
func (r *handler) createDatasetHandler(c echo.Context) error {
	cc := c.(*appctx.Context)
	ctx := cc.Request().Context()

	var data api.Dataset
	if err := validator.BindAndValidate(cc, &data); err != nil {
		logrus.WithError(err).Error("failed to bind and validate token")
		return cc.JSON(500, httputil.ErrorResponse(err.Error()))
	}

	organizationID := cc.Param("organizationId")

	dataset, err := r.datasetService.Create(ctx, organizationID, data)
	if err != nil {
		logrus.WithError(err).Error("failed to create dataset")
		return cc.JSON(500, httputil.ErrorResponse(err.Error()))
	}

	return cc.JSON(http.StatusOK, httputil.Response(dataset))
}

// listDatasetsHandler godoc
// @Summary      List datasets
// @Description  List all datasets in the current organization
// @Tags         dataset
// @Produce      json
// @Param        organizationId  path      string  true  "Organization ID"
// @Success      200             {object}  httputil.ResponseData{data=[]api.Dataset}
// @Failure      500             {object}  httputil.ErrorResponseData
// @Router       /dash/{organizationId}/dataset [get]
func (r *handler) listDatasetsHandler(c echo.Context) error {
	cc := c.(*appctx.Context)
	ctx := cc.Request().Context()

	organizationID := cc.Param("organizationId")

	datasets, err := r.datasetService.List(ctx, organizationID)
	if err != nil {
		logrus.WithError(err).Error("failed to list datasets")
		return cc.JSON(500, httputil.ErrorResponse(err.Error()))
	}

	return cc.JSON(http.StatusOK, httputil.Response(datasets))
}

// listDatasetFieldsHandler godoc
// @Summary      List dataset fields
// @Description  List all fields of a dataset
// @Tags         dataset
// @Produce      json
// @Param        organizationId  path      string  true  "Organization ID"
// @Param        datasetId       path      string  true  "Dataset ID"
// @Success      200             {object}  httputil.ResponseData{data=[]string}
// @Failure      500             {object}  httputil.ErrorResponseData
// @Router       /dash/{organizationId}/dataset/{datasetId}/fields [get]
func (r *handler) listDatasetFieldsHandler(c echo.Context) error {
	cc := c.(*appctx.Context)
	ctx := cc.Request().Context()

	organizationID := cc.Param("organizationId")
	datasetID := cc.Param("datasetId")

	fields, err := r.datasetService.ListFields(ctx, organizationID, datasetID)
	if err != nil {
		logrus.WithError(err).Error("failed to list dataset fields")
		return cc.JSON(500, httputil.ErrorResponse(err.Error()))
	}

	return cc.JSON(http.StatusOK, httputil.Response(fields))
}

// queryDatasetLogsHandler godoc
// @Summary      Query dataset logs
// @Description  Query logs for a dataset with filters and cursor
// @Tags         dataset
// @Accept       json
// @Produce      json
// @Param        organizationId  path      string               true  "Organization ID"
// @Param        datasetId       path      string               true  "Dataset ID"
// @Param        data            body      api.ListLogsSchema  true  "Query Logs Schema"
// @Success      200             {object}  httputil.ResponseData{data=api.DatasetLogsResponse}
// @Failure      500             {object}  httputil.ErrorResponseData
// @Router       /dash/{organizationId}/dataset/{datasetId}/logs [post]
func (r *handler) queryDatasetLogsHandler(c echo.Context) error {
	cc := c.(*appctx.Context)
	ctx := cc.Request().Context()

	organizationID := cc.Param("organizationId")
	datasetID := cc.Param("datasetId")

	var data api.ListLogsSchema
	if err := validator.BindAndValidate(cc, &data); err != nil {
		logrus.WithError(err).Error("failed to bind and validate token")
		return cc.JSON(500, httputil.ErrorResponse(err.Error()))
	}

	logs, err := r.datasetService.QueryLogs(ctx, organizationID, datasetID, data.FromDate, data.ToDate, data.Filter, data.Cursor)
	if err != nil {
		logrus.WithError(err).Error("failed to query dataset logs")
		return cc.JSON(500, httputil.ErrorResponse(err.Error()))
	}

	var response api.DatasetLogsResponse
	response.Data = logs
	if response.Data == nil {
		response.Data = make([]api.DatasetLog, 0)
	}

	response.Meta.TotalRowCount = len(logs)

	return cc.JSON(http.StatusOK, httputil.Response(response))
}

// getDatasetLogHandler godoc
// @Summary      Get dataset log
// @Description  Get a specific log entry by ID
// @Tags         dataset
// @Produce      json
// @Param        organizationId  path      string  true  "Organization ID"
// @Param        datasetId       path      string  true  "Dataset ID"
// @Param        logId           path      string  true  "Log ID"
// @Success      200             {object}  httputil.ResponseData{data=api.DatasetLog}
// @Failure      500             {object}  httputil.ErrorResponseData
// @Router       /dash/{organizationId}/dataset/{datasetId}/logs/{logId} [get]
func (r *handler) getDatasetLogHandler(c echo.Context) error {
	cc := c.(*appctx.Context)
	ctx := cc.Request().Context()

	organizationID := cc.Param("organizationId")
	datasetID := cc.Param("datasetId")
	logID := cc.Param("logId")

	log, err := r.datasetService.GetLog(ctx, organizationID, datasetID, logID)
	if err != nil {
		logrus.WithError(err).Error("failed to get dataset log")
		return cc.JSON(500, httputil.ErrorResponse(err.Error()))
	}

	return cc.JSON(http.StatusOK, httputil.Response(log))
}
