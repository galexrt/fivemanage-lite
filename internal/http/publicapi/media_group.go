package publicapi

import (
	"errors"
	"net/http"

	"github.com/fivemanage/lite/internal/auth"
	"github.com/fivemanage/lite/internal/http/httputil"
	"github.com/fivemanage/lite/internal/http/middleware"
	"github.com/fivemanage/lite/internal/service/file"
	"github.com/fivemanage/lite/internal/service/token"
	"github.com/fivemanage/lite/pkg/cache"
	"github.com/labstack/echo/v4"
	"github.com/spf13/viper"
)

// DRY they said
func registerMediaApi(group *echo.Group, fileService *file.Service, tokenService *token.Service, cache *cache.Cache) {
	h := &mediaHandler{fileService: fileService}
	group.POST("/image", h.uploadImage, middleware.TokenAuth(tokenService, cache), middleware.ValidateMime("image", middleware.WhitelistedImageMIME))
	group.POST("/video", h.uploadVideo, middleware.TokenAuth(tokenService, cache), middleware.ValidateMime("video", middleware.WhitelistedVideoMIME))
	group.POST("/audio", h.uploadAudio, middleware.TokenAuth(tokenService, cache), middleware.ValidateMime("audio", middleware.WhitelistedAudioMIME))
	group.POST("/file", h.uploadFile, middleware.TokenAuth(tokenService, cache), middleware.ValidateMime("file", nil))
}

type mediaHandler struct {
	fileService *file.Service
}

// uploadImage godoc
// @Summary      Upload image
// @Description  Upload an image file
// @Tags         public
// @Accept       multipart/form-data
// @Produce      json
// @Param        image  formData  file  true  "Image file"
// @Success      200   {object}  httputil.ResponseData{data={url=string}}
// @Failure      401    {object}  httputil.ErrorResponseData
// @Failure      500    {object}  httputil.ErrorResponseData
// @Router       /image [post]
func (h *mediaHandler) uploadImage(c echo.Context) error {
	return h.handleUpload(c, "image")
}

// uploadVideo godoc
// @Summary      Upload video
// @Description  Upload a video file
// @Tags         public
// @Accept       multipart/form-data
// @Produce      json
// @Param        video  formData  file  true  "Video file"
// @Success      200   {object}  httputil.ResponseData{data={url=string}}
// @Failure      401    {object}  httputil.ErrorResponseData
// @Failure      500    {object}  httputil.ErrorResponseData
// @Router       /video [post]
func (h *mediaHandler) uploadVideo(c echo.Context) error {
	return h.handleUpload(c, "video")
}

// uploadAudio godoc
// @Summary      Upload audio
// @Description  Upload an audio file
// @Tags         public
// @Accept       multipart/form-data
// @Produce      json
// @Param        audio  formData  file  true  "Audio file"
// @Success      200   {object}  httputil.ResponseData{data={url=string}}
// @Failure      401    {object}  httputil.ErrorResponseData
// @Failure      500    {object}  httputil.ErrorResponseData
// @Router       /audio [post]
func (h *mediaHandler) uploadAudio(c echo.Context) error {
	return h.handleUpload(c, "audio")
}

// uploadFile godoc
// @Summary      Upload file
// @Description  Upload a general file
// @Tags         public
// @Accept       multipart/form-data
// @Produce      json
// @Param        file  formData  file  true  "File"
// @Success      200   {object}  httputil.ResponseData{data={url=string}}
// @Failure      401   {object}  httputil.ErrorResponseData
// @Failure      500   {object}  httputil.ErrorResponseData
// @Router       /file [post]
func (h *mediaHandler) uploadFile(c echo.Context) error {
	return h.handleUpload(c, "file")
}

func (h *mediaHandler) handleUpload(c echo.Context, fileType string) error {
	var err error
	ctx := c.Request().Context()

	orgId, err := auth.CurrentOrgId(c)
	if err != nil {
		// we should probably have a better error here
		return echo.NewHTTPError(http.StatusUnauthorized, httputil.ErrorResponse("Unauthorized: "+err.Error()))
	}

	f, header, err := httputil.File(c.Request(), fileType)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, httputil.ErrorResponse(err.Error()))
	}

	var key string
	key, err = h.fileService.CreateFile(ctx, orgId, f, header)
	if err != nil {
		if errors.Is(err, file.FileTooLargeError{}) {
			return c.JSON(http.StatusRequestEntityTooLarge, httputil.ErrorResponse(err.Error()))
		}

		return c.JSON(http.StatusInternalServerError, httputil.ErrorResponse(err.Error()))
	}

	publicUrl := viper.GetString("public-url")

	return c.JSON(http.StatusOK, httputil.Response(struct {
		URL string `json:"url"`
	}{
		URL: publicUrl + "/" + key,
	}))
}
