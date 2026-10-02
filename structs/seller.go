package structs

type SellerResponse struct {
	Id             uint                `json:"id"`
	IdentityNumber string              `json:"identity_number"`
	IdentityImage  string              `json:"identity_image"`
	Status         string              `json:"status"`
	UserId         uint                `json:"user_id"`
	User           UserProfileResponse `json:"user"`
}

type UpdateSellerStatusRequest struct {
	Ids    []uint `json:"ids" binding:"required"`
	Status string `json:"status" binding:"required"`
}
