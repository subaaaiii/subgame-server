package structs

type OrderRequest struct {
	ProductId uint `json:"product_id" binding:"required"`
	Qty       int  `json:"qty" binding:"required"`
}

type CallbackRequest struct {
	OrderID string `json:"order_id"`
	Status  string `json:"status"`
}

type UpdateStatusRequest struct {
	OrderID string `json:"order_id"`
	Status  string `json:"status"`
}
