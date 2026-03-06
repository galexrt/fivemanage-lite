package internalapi

import (
	"errors"
	"net/http"

	"github.com/fivemanage/lite/internal/http/httputil"
	"github.com/fivemanage/lite/internal/service/file"
	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
)

func registerStorageApi(group *echo.Group, fileService *file.Service) {
	h := &storageHandler{fileService: fileService}
	group.GET("/storage/:organizationId", h.listStorageFiles)
	group.GET("/storage/:organizationId/file/:fileId", h.getStorageFile)
	group.POST("/storage/:organizationId/upload", h.uploadStorageFile)
}

type storageHandler struct {
	fileService *file.Service
}

// listStorageFiles godoc
// @Summary      List storage files
// @Description  List files in storage for an organization
// @Tags         storage
// @Produce      json
// @Param        organizationId  path      string  true   "Organization ID"
// @Param        search          query     string  false  "Search query"
// @Param        type            query     string  false  "File type"
// @Param        page            query     int     false  "Page number"
// @Param        pageSize        query     int     false  "Page size"
// @Success      200             {object}  httputil.ResponseData{data=api.AssetResponse}
// @Failure      500             {object}  httputil.ErrorResponseData
// @Router       /dash/storage/{organizationId} [get]
func (h *storageHandler) listStorageFiles(c echo.Context) error {
	ctx := c.Request().Context()

	organizationID := c.Param("organizationId")
	search := c.QueryParam("search")
	fileType := c.QueryParam("type")

	page, _ := httputil.QueryInt(c, "page", 0)
	pageSize, _ := httputil.QueryInt(c, "pageSize", 20)

	assetData, err := h.fileService.ListStorageFiles(ctx, organizationID, search, fileType, page, pageSize)
	if err != nil {
		if errors.Is(err, file.ListStorageError{}) {
			return c.JSON(http.StatusInternalServerError,
				httputil.ErrorResponse("Failed to list storage files"),
			)
		}

		return c.JSON(http.StatusInternalServerError, httputil.ErrorResponse(err.Error()))
	}

	return c.JSON(200, httputil.Response(assetData))
}

// getStorageFile godoc
// @Summary      Get storage file
// @Description  Get a specific storage file by ID
// @Tags         storage
// @Produce      json
// @Param        organizationId  path      string  true  "Organization ID"
// @Param        fileId          path      string  true  "File ID"
// @Success      200             {object}  httputil.ResponseData{data=api.Asset}
// @Failure      500             {object}  httputil.ErrorResponseData
// @Router       /dash/storage/{organizationId}/file/{fileId} [get]
func (h *storageHandler) getStorageFile(c echo.Context) error {
	ctx := c.Request().Context()

	organizationID := c.Param("organizationId")
	fileID := c.Param("fileId")

	assetData, err := h.fileService.GetStorageFile(ctx, organizationID, fileID)
	if err != nil {
		if errors.Is(err, file.GetFileError{}) {
			return c.JSON(http.StatusInternalServerError,
				httputil.ErrorResponse("Failed to get storage file"),
			)
		}

		return c.JSON(http.StatusInternalServerError, httputil.ErrorResponse(err.Error()))
	}

	return c.JSON(200, httputil.Response(assetData))
}

// uploadStorageFile godoc
// @Summary      Upload storage file
// @Description  Upload a file to storage for an organization
// @Tags         storage
// @Accept       multipart/form-data
// @Produce      json
// @Param        organizationId  path      string  true  "Organization ID"
// @Param        file            formData  file    true  "File to upload"
// @Success      200             {object}  httputil.ResponseData{data=string}
// @Failure      500             {object}  echo.HTTPError
// @Router       /dash/storage/{organizationId}/upload [post]
func (h *storageHandler) uploadStorageFile(c echo.Context) error {
	ctx := c.Request().Context()

	organizationID := c.Param("organizationId")

	formFile, header, err := httputil.File(c.Request(), "file")
	if err != nil {
		logrus.WithField("organizationId", organizationID).Error("failed to get file from request")

		return echo.NewHTTPError(http.StatusInternalServerError,
			httputil.ErrorResponse("Failed to get file from request"),
		)
	}

	err = h.fileService.CreateStorageFile(
		ctx,
		organizationID,
		formFile,
		header,
	)
	if err != nil {
		logrus.WithError(err).WithField("organizationId", organizationID).Error("failed to create file")
		if errors.Is(err, file.FileTooLargeError{}) {
			return echo.NewHTTPError(http.StatusRequestEntityTooLarge, httputil.ErrorResponse(err.Error()))
		}

		if errors.Is(err, file.UploadStorageError{}) {
			return echo.NewHTTPError(http.StatusInternalServerError,
				httputil.ErrorResponse("Failed to upload storage file"),
			)
		}

		return echo.NewHTTPError(http.StatusInternalServerError, httputil.ErrorResponse(err.Error()))
	}

	return c.JSON(http.StatusOK, httputil.Response("File uploaded successfully"))
}
