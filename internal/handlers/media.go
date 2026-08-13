package handlers

import (
	"strconv"
	"time"

	"github.com/asnakech/asnakech-servers/internal/domain"
	"github.com/asnakech/asnakech-servers/internal/middleware"
	"github.com/asnakech/asnakech-servers/internal/response"
	"github.com/asnakech/asnakech-servers/internal/service"
	"github.com/gin-gonic/gin"
)

type MediaHandler struct {
	media *service.MediaService
}

func NewMediaHandler(media *service.MediaService) *MediaHandler {
	return &MediaHandler{media: media}
}

type createMediaUploadRequest struct {
	Purpose          string  `json:"purpose" example:"lesson_media"`
	ContentType      string  `json:"content_type" binding:"required" example:"video/mp4"`
	OriginalFilename string  `json:"original_filename" example:"intro.mp4"`
	CourseID         *string `json:"course_id"`
	SizeBytes        int64   `json:"size_bytes" example:"1048576"`
}

type completeMediaRequest struct {
	SizeBytes       *int64 `json:"size_bytes"`
	DurationSeconds *int   `json:"duration_seconds"`
	Width           *int   `json:"width"`
	Height          *int   `json:"height"`
}

type scanResultRequest struct {
	ScanStatus string `json:"scan_status" binding:"required" example:"clean"`
	Note       string `json:"note"`
}

type MediaAssetResponse struct {
	ID               string    `json:"id"`
	OwnerID          string    `json:"owner_id"`
	CourseID         *string   `json:"course_id,omitempty"`
	Purpose          string    `json:"purpose"`
	Status           string    `json:"status"`
	ContentType      string    `json:"content_type"`
	OriginalFilename string    `json:"original_filename"`
	SizeBytes        int64     `json:"size_bytes"`
	StorageKey       string    `json:"storage_key"`
	PublicURL        string    `json:"public_url"`
	DurationSeconds  *int      `json:"duration_seconds,omitempty"`
	Width            *int      `json:"width,omitempty"`
	Height           *int      `json:"height,omitempty"`
	ScanStatus       string    `json:"scan_status"`
	ScanNote         string    `json:"scan_note,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type MediaUploadIntentResponse struct {
	Asset     MediaAssetResponse `json:"asset"`
	Method    string             `json:"method" example:"PUT"`
	UploadURL string             `json:"upload_url"`
	Headers   map[string]string  `json:"headers"`
	ExpiresAt time.Time          `json:"expires_at"`
}

type MediaUploadIntentEnvelope struct {
	Success bool                      `json:"success" example:"true"`
	Data    MediaUploadIntentResponse `json:"data"`
}

type MediaAssetEnvelope struct {
	Success bool               `json:"success" example:"true"`
	Data    MediaAssetResponse `json:"data"`
}

type MediaListEnvelope struct {
	Success bool                 `json:"success" example:"true"`
	Data    []MediaAssetResponse `json:"data"`
	Meta    response.Meta        `json:"meta"`
}

// CreateUpload godoc
// @Summary      Create presigned media upload
// @Tags         media
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body body createMediaUploadRequest true "Upload"
// @Success      201 {object} MediaUploadIntentEnvelope
// @Router       /api/v1/media/uploads [post]
func (h *MediaHandler) CreateUpload(c *gin.Context) {
	var req createMediaUploadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, bindError(err))
		return
	}
	intent, err := h.media.CreateUpload(c.Request.Context(), c.GetString(middleware.ContextUserID), domain.MediaCreateInput{
		Purpose:          domain.MediaPurpose(req.Purpose),
		ContentType:      req.ContentType,
		OriginalFilename: req.OriginalFilename,
		CourseID:         req.CourseID,
		SizeBytes:        req.SizeBytes,
	}, isPlatformAdmin(c))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Created(c, toMediaUploadIntentResponse(intent))
}

// CompleteUpload godoc
// @Summary      Confirm media upload + optional video metadata
// @Tags         media
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Media asset ID"
// @Param        body body completeMediaRequest false "Metadata"
// @Success      200 {object} MediaAssetEnvelope
// @Router       /api/v1/media/{id}/complete [post]
func (h *MediaHandler) CompleteUpload(c *gin.Context) {
	var req completeMediaRequest
	_ = c.ShouldBindJSON(&req)
	asset, err := h.media.Complete(
		c.Request.Context(),
		c.GetString(middleware.ContextUserID),
		c.Param("id"),
		domain.MediaCompleteInput{
			SizeBytes:       req.SizeBytes,
			DurationSeconds: req.DurationSeconds,
			Width:           req.Width,
			Height:          req.Height,
		},
		isPlatformAdmin(c),
	)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, toMediaAssetResponse(asset))
}

// GetMedia godoc
// @Summary      Get media asset
// @Tags         media
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Media asset ID"
// @Success      200 {object} MediaAssetEnvelope
// @Router       /api/v1/media/{id} [get]
func (h *MediaHandler) GetMedia(c *gin.Context) {
	asset, err := h.media.Get(c.Request.Context(), c.GetString(middleware.ContextUserID), c.Param("id"), isPlatformAdmin(c))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, toMediaAssetResponse(asset))
}

// ListMyMedia godoc
// @Summary      List my media assets
// @Tags         media
// @Produce      json
// @Security     BearerAuth
// @Param        page query int false "Page"
// @Param        per_page query int false "Per page"
// @Success      200 {object} MediaListEnvelope
// @Router       /api/v1/me/media [get]
func (h *MediaHandler) ListMyMedia(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "20"))
	items, total, err := h.media.ListMine(c.Request.Context(), c.GetString(middleware.ContextUserID), page, perPage)
	if err != nil {
		response.Fail(c, err)
		return
	}
	out := make([]MediaAssetResponse, 0, len(items))
	for i := range items {
		out = append(out, toMediaAssetResponse(&items[i]))
	}
	response.JSON(c, 200, out, response.Meta{
		RequestID: c.GetString("request_id"),
		Page:      page,
		PerPage:   perPage,
		Total:     total,
	})
}

// DeleteMedia godoc
// @Summary      Soft-delete media asset
// @Tags         media
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Media asset ID"
// @Success      200 {object} MessageResponse
// @Router       /api/v1/media/{id} [delete]
func (h *MediaHandler) DeleteMedia(c *gin.Context) {
	if err := h.media.Delete(c.Request.Context(), c.GetString(middleware.ContextUserID), c.Param("id"), isPlatformAdmin(c)); err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, MessageData{Message: "media asset deleted"})
}

// ApplyScanResult godoc
// @Summary      Apply virus-scan result (admin hook)
// @Tags         media
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Media asset ID"
// @Param        body body scanResultRequest true "Scan result"
// @Success      200 {object} MediaAssetEnvelope
// @Router       /api/v1/media/{id}/scan-result [post]
func (h *MediaHandler) ApplyScanResult(c *gin.Context) {
	var req scanResultRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, bindError(err))
		return
	}
	asset, err := h.media.ApplyScanResult(
		c.Request.Context(),
		c.GetString(middleware.ContextUserID),
		c.Param("id"),
		domain.MediaScanStatus(req.ScanStatus),
		req.Note,
		isPlatformAdmin(c),
	)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, toMediaAssetResponse(asset))
}

func toMediaAssetResponse(a *domain.MediaAsset) MediaAssetResponse {
	return MediaAssetResponse{
		ID:               a.ID,
		OwnerID:          a.OwnerID,
		CourseID:         a.CourseID,
		Purpose:          string(a.Purpose),
		Status:           string(a.Status),
		ContentType:      a.ContentType,
		OriginalFilename: a.OriginalFilename,
		SizeBytes:        a.SizeBytes,
		StorageKey:       a.StorageKey,
		PublicURL:        a.PublicURL,
		DurationSeconds:  a.DurationSeconds,
		Width:            a.Width,
		Height:           a.Height,
		ScanStatus:       string(a.ScanStatus),
		ScanNote:         a.ScanNote,
		CreatedAt:        a.CreatedAt,
		UpdatedAt:        a.UpdatedAt,
	}
}

func toMediaUploadIntentResponse(i *domain.MediaUploadIntent) MediaUploadIntentResponse {
	return MediaUploadIntentResponse{
		Asset:     toMediaAssetResponse(&i.Asset),
		Method:    i.Method,
		UploadURL: i.UploadURL,
		Headers:   i.Headers,
		ExpiresAt: i.ExpiresAt,
	}
}
