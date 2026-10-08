package model

type ShareLinkResponse struct {
	Code       Code       `json:"code"`
	ResourceId ResourceId `json:"resource_id"`
	Status     Status     `json:"status"`
}

func NewShareLinkResponse(link *ShareLink) ShareLinkResponse {
	return ShareLinkResponse{
		Code:       link.Code,
		ResourceId: link.ResourceId,
		Status:     link.Status,
	}
}
