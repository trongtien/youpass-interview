package sharelink

import (
	"errors"
	"log"
	"net/http"
	"service-share-link/internal/platform/http_request"
	"service-share-link/internal/platform/ratelimit"
	"service-share-link/internal/share-link/constants"
	"service-share-link/internal/share-link/model"
	"service-share-link/internal/share-link/service"
	"service-share-link/internal/share-link/validator"
	"strconv"
	"time"
)

const (
	HEADER_USER_ID = "X-User-Id"
)

type Handler struct {
	service service.ServiceShareLink
}

func NewHandler(service service.ServiceShareLink) *Handler {
	return &Handler{
		service: service,
	}
}

var (
	publicRateLimit = ratelimit.Rule{Name: "public", Limit: 120, Window: time.Minute, Key: ratelimit.ClientIP}
	viewRateLimit   = ratelimit.Rule{Name: "view", Limit: 10, Window: time.Minute, Key: viewRateLimitKey}
	ownerRateLimit  = ratelimit.Rule{Name: "owner", Limit: 30, Window: time.Minute, Key: ownerRateLimitKey}
)

func RegisterRouterShareLink(mux *http.ServeMux, service service.ServiceShareLink, limiter *ratelimit.Limiter) http.Handler {
	handler := NewHandler(service)
	mux.HandleFunc("GET /v1/public/s/{code}", limiter.Wrap(publicRateLimit, handler.httpGetLink))
	mux.HandleFunc("POST /v1/public/s/{code}/view", limiter.Wrap(publicRateLimit, limiter.Wrap(viewRateLimit, handler.httpRecordView)))

	mux.HandleFunc("POST /v1/s", limiter.Wrap(ownerRateLimit, handler.httpCreateShare))
	mux.HandleFunc("PATCH /v1/s/{code}", limiter.Wrap(ownerRateLimit, handler.httpChangeShare))
	mux.HandleFunc("DELETE /v1/s/{code}", limiter.Wrap(ownerRateLimit, handler.httpDelete))

	return mux
}

func viewRateLimitKey(r *http.Request) string {
	return ratelimit.ClientIP(r) + "|" + r.PathValue("code")
}

func ownerRateLimitKey(r *http.Request) string {
	if ownerId, err := ownerIdFromRequest(r); err == nil {
		return "user:" + strconv.FormatInt(int64(ownerId), 10)
	}

	return "ip:" + ratelimit.ClientIP(r)
}

func (h *Handler) httpCreateShare(w http.ResponseWriter, r *http.Request) {
	ownerId, err := ownerIdFromRequest(r)
	if err != nil {
		http_request.WriteError(w, http.StatusUnauthorized, "unauthorized", err.Error())
		return
	}

	var req validator.CreateShareRequest
	if err := http_request.DecodeJSON(w, r, &req); err != nil {
		writeDecodeError(w, err)
		return
	}

	if err := validator.ValidateCreateShare(&req); err != nil {
		http_request.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	link, created, err := h.service.CreateLinkShare(r.Context(), ownerId, req.ResourceId)
	switch {
	case err == nil:
	case errors.Is(err, constants.ErrResourceSharedByOther):
		http_request.WriteError(w, http.StatusConflict, "resource_already_shared", "resource is already shared by another user")
		return
	default:
		log.Printf("[Handler] Create share link owner=%d resource=%d: %v", ownerId, req.ResourceId, err)
		http_request.WriteError(w, http.StatusInternalServerError, "internal_error", "could not create share link")
		return
	}

	status := http.StatusCreated
	if !created {
		status = http.StatusOK
	}
	http_request.WriteJSON(w, status, model.NewShareLinkResponse(link))
}

func (h *Handler) httpGetLink(w http.ResponseWriter, r *http.Request) {
	code := model.Code(r.PathValue("code"))
	if err := validator.ValidateCode(code); err != nil {
		http_request.WriteError(w, http.StatusNotFound, "not_found", "share link not found")
		return
	}

	link, err := h.service.GetLinkShare(r.Context(), code)
	if err != nil {
		writePublicError(w, err, "get", code)
		return
	}

	http_request.WriteJSON(w, http.StatusOK, model.NewShareLinkResponse(link))
}

func (h *Handler) httpRecordView(w http.ResponseWriter, r *http.Request) {
	code := model.Code(r.PathValue("code"))
	if err := validator.ValidateCode(code); err != nil {
		http_request.WriteError(w, http.StatusNotFound, "not_found", "share link not found")
		return
	}

	if err := h.service.RecordView(r.Context(), code); err != nil {
		writePublicError(w, err, "record view for", code)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) httpChangeShare(w http.ResponseWriter, r *http.Request) {
	ownerId, err := ownerIdFromRequest(r)
	if err != nil {
		http_request.WriteError(w, http.StatusUnauthorized, "unauthorized", err.Error())
		return
	}

	code := model.Code(r.PathValue("code"))
	if err := validator.ValidateCode(code); err != nil {
		http_request.WriteError(w, http.StatusNotFound, "not_found", "share link not found")
		return
	}

	var req validator.ChangeStatusRequest
	if err := http_request.DecodeJSON(w, r, &req); err != nil {
		writeDecodeError(w, err)
		return
	}

	if err := validator.ValidateChangeStatus(&req); err != nil {
		http_request.WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}

	link, err := h.service.ChangeStatusLinkShare(r.Context(), ownerId, code, *req.Enabled)
	if err != nil {
		writeOwnerActionError(w, err, "change status", code)
		return
	}

	http_request.WriteJSON(w, http.StatusOK, model.NewShareLinkResponse(link))
}

func (h *Handler) httpDelete(w http.ResponseWriter, r *http.Request) {
	ownerId, err := ownerIdFromRequest(r)
	if err != nil {
		http_request.WriteError(w, http.StatusUnauthorized, "unauthorized", err.Error())
		return
	}

	code := model.Code(r.PathValue("code"))
	if err := validator.ValidateCode(code); err != nil {
		http_request.WriteError(w, http.StatusNotFound, "not_found", "share link not found")
		return
	}

	if err := h.service.DeleteLinkShare(r.Context(), ownerId, code); err != nil {
		writeOwnerActionError(w, err, "delete", code)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func writeDecodeError(w http.ResponseWriter, err error) {
	var maxBytesErr *http.MaxBytesError
	if errors.As(err, &maxBytesErr) {
		http_request.WriteError(w, http.StatusRequestEntityTooLarge, "body_too_large", "request body is too large")
		return
	}
	http_request.WriteError(w, http.StatusBadRequest, "invalid_body", err.Error())
}

func writePublicError(w http.ResponseWriter, err error, action string, code model.Code) {
	switch {
	case errors.Is(err, constants.ErrorRecordNotFound):
		http_request.WriteError(w, http.StatusNotFound, "not_found", "share link not found")
	case errors.Is(err, constants.ErrLinkDeleted):
		http_request.WriteError(w, http.StatusGone, "link_deleted", err.Error())
	case errors.Is(err, constants.ErrLinkDisable):
		http_request.WriteError(w, http.StatusForbidden, "link_disabled", err.Error())
	default:
		log.Printf("[Handler] %s share link code=%s: %v", action, code, err)
		http_request.WriteError(w, http.StatusInternalServerError, "internal_error", "could not "+action+" share link")
	}
}

func writeOwnerActionError(w http.ResponseWriter, err error, action string, code model.Code) {
	switch {
	case errors.Is(err, constants.ErrorRecordNotFound):
		http_request.WriteError(w, http.StatusNotFound, "not_found", "share link not found")
	case errors.Is(err, constants.ErrNotOwner):
		http_request.WriteError(w, http.StatusForbidden, "forbidden", "share link belongs to another user")
	case errors.Is(err, constants.ErrorRecordDeleteNotChange):
		http_request.WriteError(w, http.StatusGone, "link_deleted", "share link is deleted and cannot be changed")
	default:
		log.Printf("[Handler] %s share link code=%s: %v", action, code, err)
		http_request.WriteError(w, http.StatusInternalServerError, "internal_error", "could not "+action+" share link")
	}
}

func ownerIdFromRequest(r *http.Request) (model.OwnerId, error) {
	id, err := strconv.ParseInt(r.Header.Get(HEADER_USER_ID), 10, 64)
	if err != nil || id <= 0 {
		return 0, errors.New("missing or invalid " + HEADER_USER_ID + " header")
	}

	return model.OwnerId(id), nil
}
