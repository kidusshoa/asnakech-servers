package handlers

import (
	"strconv"

	"github.com/asnakech/asnakech-servers/internal/domain"
	"github.com/asnakech/asnakech-servers/internal/middleware"
	"github.com/asnakech/asnakech-servers/internal/response"
	"github.com/asnakech/asnakech-servers/internal/service"
	"github.com/gin-gonic/gin"
)

type CommunicationHandler struct {
	comms *service.CommunicationService
}

func NewCommunicationHandler(comms *service.CommunicationService) *CommunicationHandler {
	return &CommunicationHandler{comms: comms}
}

type announcementBody struct {
	Title  string `json:"title" binding:"required"`
	Body   string `json:"body"`
	Pinned bool   `json:"pinned"`
}

type threadBody struct {
	Title string `json:"title" binding:"required"`
	Body  string `json:"body" binding:"required"`
}

type postBody struct {
	Body     string  `json:"body" binding:"required"`
	ParentID *string `json:"parent_id"`
}

type dmStartBody struct {
	UserID string `json:"user_id" binding:"required"`
}

type messageBody struct {
	Body string `json:"body" binding:"required"`
}

type AnnouncementResponse struct {
	ID          string  `json:"id"`
	CourseID    string  `json:"course_id"`
	AuthorID    string  `json:"author_id"`
	Title       string  `json:"title"`
	Body        string  `json:"body"`
	Status      string  `json:"status"`
	Pinned      bool    `json:"pinned"`
	PublishedAt *string `json:"published_at,omitempty"`
	AuthorName  string  `json:"author_name,omitempty"`
	CreatedAt   string  `json:"created_at"`
	UpdatedAt   string  `json:"updated_at"`
}

type ThreadResponse struct {
	ID         string `json:"id"`
	CourseID   string `json:"course_id"`
	AuthorID   string `json:"author_id"`
	Title      string `json:"title"`
	Status     string `json:"status"`
	AuthorName string `json:"author_name,omitempty"`
	PostCount  int    `json:"post_count"`
	CreatedAt  string `json:"created_at"`
	UpdatedAt  string `json:"updated_at"`
}

type PostResponse struct {
	ID         string  `json:"id"`
	ThreadID   string  `json:"thread_id"`
	AuthorID   string  `json:"author_id"`
	ParentID   *string `json:"parent_id,omitempty"`
	Body       string  `json:"body"`
	AuthorName string  `json:"author_name,omitempty"`
	CreatedAt  string  `json:"created_at"`
	UpdatedAt  string  `json:"updated_at"`
}

type ConversationResponse struct {
	ID            string `json:"id"`
	OtherUserID   string `json:"other_user_id"`
	OtherUserName string `json:"other_user_name,omitempty"`
	LastMessage   string `json:"last_message,omitempty"`
	UnreadCount   int    `json:"unread_count"`
	UpdatedAt     string `json:"updated_at"`
}

type DMMessageResponse struct {
	ID        string  `json:"id"`
	SenderID  string  `json:"sender_id"`
	Body      string  `json:"body"`
	ReadAt    *string `json:"read_at,omitempty"`
	CreatedAt string  `json:"created_at"`
}

type NotificationResponse struct {
	ID        string            `json:"id"`
	EventType string            `json:"event_type"`
	Subject   string            `json:"subject"`
	Body      string            `json:"body"`
	Payload   map[string]string `json:"payload,omitempty"`
	ReadAt    *string           `json:"read_at,omitempty"`
	CreatedAt string            `json:"created_at"`
}

// CreateAnnouncement godoc
// @Summary      Create course announcement
// @Tags         communication
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Course ID"
// @Param        body body announcementBody true "Announcement"
// @Success      201 {object} handlers.AnnouncementEnvelope
// @Router       /api/v1/courses/{id}/announcements [post]
func (h *CommunicationHandler) CreateAnnouncement(c *gin.Context) {
	var req announcementBody
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, bindError(err))
		return
	}
	a, err := h.comms.CreateAnnouncement(c.Request.Context(), c.GetString(middleware.ContextUserID), c.Param("id"), req.Title, req.Body, req.Pinned, isPlatformAdmin(c))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Created(c, toAnnouncementResponse(a))
}

// ListAnnouncements godoc
// @Summary      List course announcements
// @Tags         communication
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Course ID"
// @Success      200 {object} handlers.AnnouncementListEnvelope
// @Router       /api/v1/courses/{id}/announcements [get]
func (h *CommunicationHandler) ListAnnouncements(c *gin.Context) {
	items, err := h.comms.ListAnnouncements(c.Request.Context(), c.GetString(middleware.ContextUserID), c.Param("id"), isPlatformAdmin(c))
	if err != nil {
		response.Fail(c, err)
		return
	}
	out := make([]AnnouncementResponse, 0, len(items))
	for i := range items {
		out = append(out, toAnnouncementResponse(&items[i]))
	}
	response.JSON(c, 200, out, response.Meta{RequestID: c.GetString("request_id"), Total: int64(len(out))})
}

// GetAnnouncement godoc
// @Summary      Get announcement
// @Tags         communication
// @Produce      json
// @Security     BearerAuth
// @Param        announcementId path string true "Announcement ID"
// @Success      200 {object} handlers.AnnouncementEnvelope
// @Router       /api/v1/announcements/{announcementId} [get]
func (h *CommunicationHandler) GetAnnouncement(c *gin.Context) {
	a, err := h.comms.GetAnnouncement(c.Request.Context(), c.GetString(middleware.ContextUserID), c.Param("announcementId"), isPlatformAdmin(c))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, toAnnouncementResponse(a))
}

// UpdateAnnouncement godoc
// @Summary      Update draft announcement
// @Tags         communication
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        announcementId path string true "Announcement ID"
// @Param        body body announcementBody true "Announcement"
// @Success      200 {object} handlers.AnnouncementEnvelope
// @Router       /api/v1/announcements/{announcementId} [patch]
func (h *CommunicationHandler) UpdateAnnouncement(c *gin.Context) {
	var req announcementBody
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, bindError(err))
		return
	}
	a, err := h.comms.UpdateAnnouncement(c.Request.Context(), c.GetString(middleware.ContextUserID), c.Param("announcementId"), req.Title, req.Body, req.Pinned, isPlatformAdmin(c))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, toAnnouncementResponse(a))
}

// PublishAnnouncement godoc
// @Summary      Publish announcement
// @Tags         communication
// @Produce      json
// @Security     BearerAuth
// @Param        announcementId path string true "Announcement ID"
// @Success      200 {object} handlers.AnnouncementEnvelope
// @Router       /api/v1/announcements/{announcementId}/publish [post]
func (h *CommunicationHandler) PublishAnnouncement(c *gin.Context) {
	a, err := h.comms.PublishAnnouncement(c.Request.Context(), c.GetString(middleware.ContextUserID), c.Param("announcementId"), isPlatformAdmin(c))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, toAnnouncementResponse(a))
}

// DeleteAnnouncement godoc
// @Summary      Delete announcement
// @Tags         communication
// @Produce      json
// @Security     BearerAuth
// @Param        announcementId path string true "Announcement ID"
// @Success      204
// @Router       /api/v1/announcements/{announcementId} [delete]
func (h *CommunicationHandler) DeleteAnnouncement(c *gin.Context) {
	if err := h.comms.DeleteAnnouncement(c.Request.Context(), c.GetString(middleware.ContextUserID), c.Param("announcementId"), isPlatformAdmin(c)); err != nil {
		response.Fail(c, err)
		return
	}
	response.NoContent(c)
}

// CreateThread godoc
// @Summary      Create discussion thread
// @Tags         communication
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Course ID"
// @Param        body body threadBody true "Thread"
// @Success      201 {object} handlers.ThreadEnvelope
// @Router       /api/v1/courses/{id}/threads [post]
func (h *CommunicationHandler) CreateThread(c *gin.Context) {
	var req threadBody
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, bindError(err))
		return
	}
	t, err := h.comms.CreateThread(c.Request.Context(), c.GetString(middleware.ContextUserID), c.Param("id"), req.Title, req.Body, isPlatformAdmin(c))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Created(c, toThreadResponse(t))
}

// ListThreads godoc
// @Summary      List discussion threads
// @Tags         communication
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Course ID"
// @Success      200 {object} handlers.ThreadListEnvelope
// @Router       /api/v1/courses/{id}/threads [get]
func (h *CommunicationHandler) ListThreads(c *gin.Context) {
	items, err := h.comms.ListThreads(c.Request.Context(), c.GetString(middleware.ContextUserID), c.Param("id"), isPlatformAdmin(c))
	if err != nil {
		response.Fail(c, err)
		return
	}
	out := make([]ThreadResponse, 0, len(items))
	for i := range items {
		out = append(out, toThreadResponse(&items[i]))
	}
	response.JSON(c, 200, out, response.Meta{RequestID: c.GetString("request_id"), Total: int64(len(out))})
}

// GetThread godoc
// @Summary      Get discussion thread
// @Tags         communication
// @Produce      json
// @Security     BearerAuth
// @Param        threadId path string true "Thread ID"
// @Success      200 {object} handlers.ThreadEnvelope
// @Router       /api/v1/threads/{threadId} [get]
func (h *CommunicationHandler) GetThread(c *gin.Context) {
	t, err := h.comms.GetThread(c.Request.Context(), c.GetString(middleware.ContextUserID), c.Param("threadId"), isPlatformAdmin(c))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, toThreadResponse(t))
}

// LockThread godoc
// @Summary      Lock discussion thread
// @Tags         communication
// @Produce      json
// @Security     BearerAuth
// @Param        threadId path string true "Thread ID"
// @Success      200 {object} handlers.ThreadEnvelope
// @Router       /api/v1/threads/{threadId}/lock [post]
func (h *CommunicationHandler) LockThread(c *gin.Context) {
	t, err := h.comms.LockThread(c.Request.Context(), c.GetString(middleware.ContextUserID), c.Param("threadId"), isPlatformAdmin(c))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, toThreadResponse(t))
}

// CreatePost godoc
// @Summary      Reply in discussion thread
// @Tags         communication
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        threadId path string true "Thread ID"
// @Param        body body postBody true "Post"
// @Success      201 {object} handlers.PostEnvelope
// @Router       /api/v1/threads/{threadId}/posts [post]
func (h *CommunicationHandler) CreatePost(c *gin.Context) {
	var req postBody
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, bindError(err))
		return
	}
	p, err := h.comms.CreatePost(c.Request.Context(), c.GetString(middleware.ContextUserID), c.Param("threadId"), req.Body, req.ParentID, isPlatformAdmin(c))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Created(c, toPostResponse(p))
}

// ListPosts godoc
// @Summary      List thread posts
// @Tags         communication
// @Produce      json
// @Security     BearerAuth
// @Param        threadId path string true "Thread ID"
// @Success      200 {object} handlers.PostListEnvelope
// @Router       /api/v1/threads/{threadId}/posts [get]
func (h *CommunicationHandler) ListPosts(c *gin.Context) {
	items, err := h.comms.ListPosts(c.Request.Context(), c.GetString(middleware.ContextUserID), c.Param("threadId"), isPlatformAdmin(c))
	if err != nil {
		response.Fail(c, err)
		return
	}
	out := make([]PostResponse, 0, len(items))
	for i := range items {
		out = append(out, toPostResponse(&items[i]))
	}
	response.JSON(c, 200, out, response.Meta{RequestID: c.GetString("request_id"), Total: int64(len(out))})
}

// UpdatePost godoc
// @Summary      Edit discussion post
// @Tags         communication
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        postId path string true "Post ID"
// @Param        body body postBody true "Post"
// @Success      200 {object} handlers.PostEnvelope
// @Router       /api/v1/posts/{postId} [patch]
func (h *CommunicationHandler) UpdatePost(c *gin.Context) {
	var req postBody
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, bindError(err))
		return
	}
	p, err := h.comms.UpdatePost(c.Request.Context(), c.GetString(middleware.ContextUserID), c.Param("postId"), req.Body, isPlatformAdmin(c))
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.OK(c, toPostResponse(p))
}

// DeletePost godoc
// @Summary      Delete discussion post
// @Tags         communication
// @Produce      json
// @Security     BearerAuth
// @Param        postId path string true "Post ID"
// @Success      204
// @Router       /api/v1/posts/{postId} [delete]
func (h *CommunicationHandler) DeletePost(c *gin.Context) {
	if err := h.comms.DeletePost(c.Request.Context(), c.GetString(middleware.ContextUserID), c.Param("postId"), isPlatformAdmin(c)); err != nil {
		response.Fail(c, err)
		return
	}
	response.NoContent(c)
}

// StartConversation godoc
// @Summary      Start or get DM conversation
// @Tags         communication
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        body body dmStartBody true "Recipient"
// @Success      201 {object} handlers.ConversationEnvelope
// @Router       /api/v1/conversations [post]
func (h *CommunicationHandler) StartConversation(c *gin.Context) {
	var req dmStartBody
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, bindError(err))
		return
	}
	conv, err := h.comms.StartConversation(c.Request.Context(), c.GetString(middleware.ContextUserID), req.UserID)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Created(c, toConversationResponse(conv))
}

// ListConversations godoc
// @Summary      List my DM conversations
// @Tags         communication
// @Produce      json
// @Security     BearerAuth
// @Success      200 {object} handlers.ConversationListEnvelope
// @Router       /api/v1/me/conversations [get]
func (h *CommunicationHandler) ListConversations(c *gin.Context) {
	items, err := h.comms.ListConversations(c.Request.Context(), c.GetString(middleware.ContextUserID))
	if err != nil {
		response.Fail(c, err)
		return
	}
	out := make([]ConversationResponse, 0, len(items))
	for i := range items {
		out = append(out, toConversationResponse(&items[i]))
	}
	response.JSON(c, 200, out, response.Meta{RequestID: c.GetString("request_id"), Total: int64(len(out))})
}

// SendMessage godoc
// @Summary      Send DM message
// @Tags         communication
// @Accept       json
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Conversation ID"
// @Param        body body messageBody true "Message"
// @Success      201 {object} handlers.DMMessageEnvelope
// @Router       /api/v1/conversations/{id}/messages [post]
func (h *CommunicationHandler) SendMessage(c *gin.Context) {
	var req messageBody
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, bindError(err))
		return
	}
	m, err := h.comms.SendMessage(c.Request.Context(), c.GetString(middleware.ContextUserID), c.Param("id"), req.Body)
	if err != nil {
		response.Fail(c, err)
		return
	}
	response.Created(c, toDMMessageResponse(m))
}

// ListMessages godoc
// @Summary      List DM messages
// @Tags         communication
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Conversation ID"
// @Param        page query int false "Page"
// @Param        per_page query int false "Per page"
// @Success      200 {object} handlers.DMMessageListEnvelope
// @Router       /api/v1/conversations/{id}/messages [get]
func (h *CommunicationHandler) ListMessages(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "50"))
	items, total, err := h.comms.ListMessages(c.Request.Context(), c.GetString(middleware.ContextUserID), c.Param("id"), page, perPage)
	if err != nil {
		response.Fail(c, err)
		return
	}
	out := make([]DMMessageResponse, 0, len(items))
	for i := range items {
		out = append(out, toDMMessageResponse(&items[i]))
	}
	response.JSON(c, 200, out, response.Meta{RequestID: c.GetString("request_id"), Page: page, PerPage: perPage, Total: int64(total)})
}

// MarkConversationRead godoc
// @Summary      Mark DM conversation read
// @Tags         communication
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Conversation ID"
// @Success      204
// @Router       /api/v1/conversations/{id}/read [post]
func (h *CommunicationHandler) MarkConversationRead(c *gin.Context) {
	if err := h.comms.MarkConversationRead(c.Request.Context(), c.GetString(middleware.ContextUserID), c.Param("id")); err != nil {
		response.Fail(c, err)
		return
	}
	response.NoContent(c)
}

// ListNotifications godoc
// @Summary      List in-app notifications
// @Tags         communication
// @Produce      json
// @Security     BearerAuth
// @Param        unread_only query bool false "Unread only"
// @Param        page query int false "Page"
// @Param        per_page query int false "Per page"
// @Success      200 {object} handlers.NotificationListEnvelope
// @Router       /api/v1/me/notifications [get]
func (h *CommunicationHandler) ListNotifications(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	perPage, _ := strconv.Atoi(c.DefaultQuery("per_page", "20"))
	unreadOnly := c.Query("unread_only") == "true"
	items, total, err := h.comms.ListNotifications(c.Request.Context(), c.GetString(middleware.ContextUserID), domain.NotificationListFilter{
		UnreadOnly: unreadOnly,
		Page:       page,
		PerPage:    perPage,
	})
	if err != nil {
		response.Fail(c, err)
		return
	}
	out := make([]NotificationResponse, 0, len(items))
	for i := range items {
		out = append(out, toNotificationResponse(&items[i]))
	}
	response.JSON(c, 200, out, response.Meta{RequestID: c.GetString("request_id"), Page: page, PerPage: perPage, Total: int64(total)})
}

// MarkNotificationRead godoc
// @Summary      Mark notification read
// @Tags         communication
// @Produce      json
// @Security     BearerAuth
// @Param        id path string true "Notification ID"
// @Success      204
// @Router       /api/v1/notifications/{id}/read [post]
func (h *CommunicationHandler) MarkNotificationRead(c *gin.Context) {
	if err := h.comms.MarkNotificationRead(c.Request.Context(), c.GetString(middleware.ContextUserID), c.Param("id")); err != nil {
		response.Fail(c, err)
		return
	}
	response.NoContent(c)
}

// MarkAllNotificationsRead godoc
// @Summary      Mark all notifications read
// @Tags         communication
// @Produce      json
// @Security     BearerAuth
// @Success      204
// @Router       /api/v1/me/notifications/read-all [post]
func (h *CommunicationHandler) MarkAllNotificationsRead(c *gin.Context) {
	if err := h.comms.MarkAllNotificationsRead(c.Request.Context(), c.GetString(middleware.ContextUserID)); err != nil {
		response.Fail(c, err)
		return
	}
	response.NoContent(c)
}

func toAnnouncementResponse(a *domain.Announcement) AnnouncementResponse {
	resp := AnnouncementResponse{
		ID: a.ID, CourseID: a.CourseID, AuthorID: a.AuthorID,
		Title: a.Title, Body: a.Body, Status: string(a.Status),
		Pinned: a.Pinned, AuthorName: a.AuthorName,
		CreatedAt: a.CreatedAt.Format(timeRFC3339), UpdatedAt: a.UpdatedAt.Format(timeRFC3339),
	}
	if a.PublishedAt != nil {
		s := a.PublishedAt.Format(timeRFC3339)
		resp.PublishedAt = &s
	}
	return resp
}

func toThreadResponse(t *domain.DiscussionThread) ThreadResponse {
	return ThreadResponse{
		ID: t.ID, CourseID: t.CourseID, AuthorID: t.AuthorID,
		Title: t.Title, Status: string(t.Status), AuthorName: t.AuthorName,
		PostCount: t.PostCount,
		CreatedAt: t.CreatedAt.Format(timeRFC3339), UpdatedAt: t.UpdatedAt.Format(timeRFC3339),
	}
}

func toPostResponse(p *domain.DiscussionPost) PostResponse {
	return PostResponse{
		ID: p.ID, ThreadID: p.ThreadID, AuthorID: p.AuthorID, ParentID: p.ParentID,
		Body: p.Body, AuthorName: p.AuthorName,
		CreatedAt: p.CreatedAt.Format(timeRFC3339), UpdatedAt: p.UpdatedAt.Format(timeRFC3339),
	}
}

func toConversationResponse(c *domain.DMConversation) ConversationResponse {
	otherID := c.OtherUserID
	if otherID == "" {
		if c.UserAID != "" {
			otherID = c.UserBID
		}
	}
	return ConversationResponse{
		ID: c.ID, OtherUserID: otherID, OtherUserName: c.OtherUserName,
		LastMessage: c.LastMessage, UnreadCount: c.UnreadCount,
		UpdatedAt: c.UpdatedAt.Format(timeRFC3339),
	}
}

func toDMMessageResponse(m *domain.DMMessage) DMMessageResponse {
	resp := DMMessageResponse{
		ID: m.ID, SenderID: m.SenderID, Body: m.Body,
		CreatedAt: m.CreatedAt.Format(timeRFC3339),
	}
	if m.ReadAt != nil {
		s := m.ReadAt.Format(timeRFC3339)
		resp.ReadAt = &s
	}
	return resp
}

func toNotificationResponse(n *domain.Notification) NotificationResponse {
	resp := NotificationResponse{
		ID: n.ID, EventType: n.EventType, Subject: n.Subject, Body: n.Body,
		Payload: n.Payload, CreatedAt: n.CreatedAt.Format(timeRFC3339),
	}
	if n.ReadAt != nil {
		s := n.ReadAt.Format(timeRFC3339)
		resp.ReadAt = &s
	}
	return resp
}

const timeRFC3339 = "2006-01-02T15:04:05Z07:00"

// Swagger envelope types
type AnnouncementEnvelope struct {
	Success bool                 `json:"success" example:"true"`
	Data    AnnouncementResponse `json:"data"`
}

type AnnouncementListEnvelope struct {
	Success bool                   `json:"success" example:"true"`
	Data    []AnnouncementResponse `json:"data"`
	Meta    response.Meta          `json:"meta"`
}

type ThreadEnvelope struct {
	Success bool           `json:"success" example:"true"`
	Data    ThreadResponse `json:"data"`
}

type ThreadListEnvelope struct {
	Success bool             `json:"success" example:"true"`
	Data    []ThreadResponse `json:"data"`
	Meta    response.Meta    `json:"meta"`
}

type PostEnvelope struct {
	Success bool         `json:"success" example:"true"`
	Data    PostResponse `json:"data"`
}

type PostListEnvelope struct {
	Success bool           `json:"success" example:"true"`
	Data    []PostResponse `json:"data"`
	Meta    response.Meta  `json:"meta"`
}

type ConversationEnvelope struct {
	Success bool                 `json:"success" example:"true"`
	Data    ConversationResponse `json:"data"`
}

type ConversationListEnvelope struct {
	Success bool                   `json:"success" example:"true"`
	Data    []ConversationResponse `json:"data"`
	Meta    response.Meta          `json:"meta"`
}

type DMMessageEnvelope struct {
	Success bool              `json:"success" example:"true"`
	Data    DMMessageResponse `json:"data"`
}

type DMMessageListEnvelope struct {
	Success bool                `json:"success" example:"true"`
	Data    []DMMessageResponse `json:"data"`
	Meta    response.Meta       `json:"meta"`
}

type NotificationListEnvelope struct {
	Success bool                   `json:"success" example:"true"`
	Data    []NotificationResponse `json:"data"`
	Meta    response.Meta          `json:"meta"`
}
