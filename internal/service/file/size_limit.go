package file

import "fmt"

const (
	defaultMaxImageUploadBytes int64 = 10 * 1024 * 1024  // 10 MiB
	defaultMaxAudioUploadBytes int64 = 50 * 1024 * 1024  // 50 MiB
	defaultMaxVideoUploadBytes int64 = 100 * 1024 * 1024 // 100 MiB
	defaultMaxFileUploadBytes  int64 = 100 * 1024 * 1024 // 100 MiB
)

type UploadLimits struct {
	ImageBytes int64
	AudioBytes int64
	VideoBytes int64
	FileBytes  int64
}

func DefaultUploadLimits() UploadLimits {
	return UploadLimits{
		ImageBytes: defaultMaxImageUploadBytes,
		AudioBytes: defaultMaxAudioUploadBytes,
		VideoBytes: defaultMaxVideoUploadBytes,
		FileBytes:  defaultMaxFileUploadBytes,
	}
}

func normalizeUploadLimits(limits UploadLimits) UploadLimits {
	defaults := DefaultUploadLimits()
	if limits.ImageBytes <= 0 {
		limits.ImageBytes = defaults.ImageBytes
	}
	if limits.AudioBytes <= 0 {
		limits.AudioBytes = defaults.AudioBytes
	}
	if limits.VideoBytes <= 0 {
		limits.VideoBytes = defaults.VideoBytes
	}
	if limits.FileBytes <= 0 {
		limits.FileBytes = defaults.FileBytes
	}

	return limits
}

func maxUploadSizeForType(fileType string, limits UploadLimits) int64 {
	switch fileType {
	case "image":
		return limits.ImageBytes
	case "audio":
		return limits.AudioBytes
	case "video":
		return limits.VideoBytes
	default:
		return limits.FileBytes
	}
}

func validateUploadSize(fileType string, size int64, limits UploadLimits) error {
	maxSize := maxUploadSizeForType(fileType, limits)
	if size <= maxSize {
		return nil
	}

	return FileTooLargeError{
		FileType: fileType,
		Size:     size,
		MaxSize:  maxSize,
	}
}

type FileTooLargeError struct {
	FileType string
	Size     int64
	MaxSize  int64
}

func (e FileTooLargeError) Error() string {
	return fmt.Sprintf("file too large for type %q: got %d bytes, max allowed is %d bytes", e.FileType, e.Size, e.MaxSize)
}

func (FileTooLargeError) Is(target error) bool {
	_, ok := target.(FileTooLargeError)
	return ok
}
