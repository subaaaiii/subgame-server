package structs

type UpdateProductStatusRequest struct {
	Ids    []uint `json:"ids" binding:"required"`
	Status string `json:"status" binding:"required"`
}
