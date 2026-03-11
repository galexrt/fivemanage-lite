package file

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"strings"

	"github.com/fivemanage/lite/api"
	"github.com/fivemanage/lite/internal/crypt"
	"github.com/fivemanage/lite/internal/database"
	filequery "github.com/fivemanage/lite/internal/database/query/file"
	"github.com/fivemanage/lite/internal/http/httputil"
	"github.com/fivemanage/lite/pkg/storage"
	"github.com/sirupsen/logrus"
	"github.com/uptrace/bun"
)

type Service struct {
	db           *bun.DB
	storage      storage.StorageLayer
	uploadLimits UploadLimits
}

func NewService(db *bun.DB, storageLayer storage.StorageLayer, uploadLimits UploadLimits) *Service {
	return &Service{
		db:           db,
		storage:      storageLayer,
		uploadLimits: normalizeUploadLimits(uploadLimits),
	}
}

// this is used in the public api only
func (s *Service) CreateFile(
	ctx context.Context,
	organizationID string,
	file multipart.File,
	fileHeader *multipart.FileHeader,
) (string, error) {
	var err error
	var key string

	primaryKey, err := crypt.GeneratePrimaryKey()
	if err != nil {
		return "", UploadStorageError{
			ErrorMsg: err.Error(),
		}
	}

	mimeType, ext, fileType, err := httputil.GetMimeDetails(fileHeader, file)
	if err != nil {
		return "", UploadStorageError{
			ErrorMsg: errors.New("failed to get mime type").Error(),
		}
	}

	if err := validateUploadSize(fileType, fileHeader.Size, s.uploadLimits); err != nil {
		return "", err
	}

	key, err = generateFileKey(organizationID, ext)
	if err != nil {
		return "", UploadStorageError{
			ErrorMsg: errors.New("failed to generate file key").Error(),
		}
	}

	tx, err := filequery.Create(ctx, s.db, &database.Asset{
		ID:             primaryKey,
		Type:           fileType,
		Size:           fileHeader.Size,
		OrganizationID: organizationID,
		Key:            key,
	})
	if err != nil {
		return "", err
	}

	// you'd think we didn't need to do this, but the since we read the file before this step to get mime type and shit
	// we need to reset the file pointer and read it again
	buffer, err := s.encode(file, fileHeader)
	if err != nil {
		logrus.WithError(err).WithField("organization_id", organizationID).Error("FileService.CreateStorageFile")
		if err := tx.Rollback(); err != nil {
			logrus.WithError(err).WithField("organization_id", organizationID).Error("FileService.CreateStorageFile")
			return "", err
		}

		return "", UploadStorageError{
			ErrorMsg: err.Error(),
		}
	}

	err = s.storage.UploadFile(ctx, buffer, key, mimeType)
	if err != nil {
		if err := tx.Rollback(); err != nil {
			return "", err
		}

		return "", err
	}

	// this is a bit tricky, but if this fails....then...oh well
	err = tx.Commit()
	if err != nil {
		return "", err
	}

	return key, nil
}

// uhhh, this is used in the dashboard, not the public api
func (s *Service) CreateStorageFile(
	ctx context.Context,
	organizationID string,
	file multipart.File,
	fileHeader *multipart.FileHeader,
) error {
	var err error

	primaryKey, err := crypt.GeneratePrimaryKey()
	if err != nil {
		return UploadStorageError{
			ErrorMsg: err.Error(),
		}
	}

	mimeType, _, fileType, err := httputil.GetMimeDetails(fileHeader, file)
	if err != nil {
		return UploadStorageError{
			ErrorMsg: errors.New("failed to get mime type").Error(),
		}
	}
	if err := validateUploadSize(fileType, fileHeader.Size, s.uploadLimits); err != nil {
		return err
	}

	// fileHeader.Filename has the extension most of the time
	// should it become an issue, we can look for it and check if its empty;
	key := generateWebKey(organizationID, fileHeader.Filename)

	tx, err := filequery.Create(ctx, s.db, &database.Asset{
		ID:             primaryKey,
		OrganizationID: organizationID,
		Type:           fileType,
		Size:           fileHeader.Size,
		Key:            key,
	})
	if err != nil {
		logrus.WithError(err).WithField("organization_id", organizationID).Error("FileService.CreateStorageFile")
		return err
	}

	buffer, err := s.encode(file, fileHeader)
	if err != nil {
		logrus.WithError(err).WithField("organization_id", organizationID).Error("FileService.CreateStorageFile")
		if err := tx.Rollback(); err != nil {
			logrus.WithError(err).WithField("organization_id", organizationID).Error("FileService.CreateStorageFile")
			return err
		}

		return UploadStorageError{
			ErrorMsg: err.Error(),
		}
	}

	err = s.storage.UploadFile(ctx, buffer, key, mimeType)
	if err != nil {
		logrus.WithError(err).WithField("organization_id", organizationID).Error("FileService.CreateStorageFile")

		if err := tx.Rollback(); err != nil {
			logrus.WithError(err).WithField("organization_id", organizationID).Error("FileService.CreateStorageFile")
			return err
		}

		return UploadStorageError{
			ErrorMsg: err.Error(),
		}
	}

	err = tx.Commit()
	if err != nil {
		return err
	}

	return nil
}

func (s *Service) ListStorageFiles(
	ctx context.Context,
	organizationID string,
	search string,
	fileType string,
	page int,
	pageSize int,
) (*api.AssetResponse, error) {
	var err error

	files, err := filequery.FindStorageFiles(ctx, s.db, organizationID, search, fileType, page, pageSize)
	if err != nil {
		storageError := &ListStorageError{
			ErrorMsg: err.Error(),
		}
		logrus.WithError(storageError).
			WithField("organization_id", organizationID).
			Error("FileService.ListStorageFiles")
		return nil, storageError
	}

	assets := make([]*api.Asset, len(files))
	for i, file := range files {
		assets[i] = &api.Asset{
			ID:        file.ID,
			Type:      file.Type,
			Key:       file.Key,
			Size:      file.Size,
			CreatedAt: file.CreatedAt,
		}
	}

	totalCount, err := filequery.FindTotalStorageCount(ctx, s.db, organizationID)
	if err != nil {
		storageError := &ListStorageError{
			ErrorMsg: err.Error(),
		}

		logrus.WithError(storageError).
			WithField("organization_id", organizationID).Error("FileService.ListStorageFiles")

		return nil, storageError
	}

	response := &api.AssetResponse{
		StorageFiles: assets,
		TotalCount:   totalCount,
	}

	return response, nil
}

func (s *Service) GetStorageFile(
	ctx context.Context,
	organizationID string,
	fileID string,
) (*api.Asset, error) {
	file, err := filequery.FindFileByID(ctx, s.db, organizationID, fileID)
	if err != nil {
		storageError := &GetFileError{
			ErrorMsg: err.Error(),
		}
		logrus.WithError(storageError).
			WithField("organization_id", organizationID).
			Error("FileService.GetStorageFile")
		return nil, storageError
	}

	return &api.Asset{
		ID:        file.ID,
		Type:      file.Type,
		Key:       file.Key,
		Size:      file.Size,
		CreatedAt: file.CreatedAt,
	}, nil
}

func (s *Service) ProxyFile(
	ctx context.Context,
	organizationID string,
	fileName string,
) (io.ReadCloser, string, int64, error) {
	if strings.TrimSpace(organizationID) == "" || strings.TrimSpace(fileName) == "" {
		return nil, "", 0, GetFileError{
			ErrorMsg: "organization id and file name are required",
		}
	}

	if strings.Contains(fileName, "/") {
		return nil, "", 0, GetFileError{
			ErrorMsg: "file name must not contain path separators",
		}
	}

	key := fmt.Sprintf("%s/%s", organizationID, fileName)
	reader, contentType, contentLength, err := s.storage.GetFile(ctx, key)
	if err != nil {
		return nil, "", 0, err
	}

	return reader, contentType, contentLength, nil
}

func (s *Service) encode(file multipart.File, header *multipart.FileHeader) (*bytes.Reader, error) {
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}

	buf := make([]byte, header.Size)
	_, err := file.Read(buf)
	if err != nil {
		return nil, err
	}

	reader := bytes.NewReader(buf)
	return reader, nil
}
