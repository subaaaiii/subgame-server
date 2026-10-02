package helpers

const (
	StatusPending    = "PENDING"
	StatusPaid       = "PAID"
	StatusFailed     = "FAILED"
	StatusExpired    = "EXPIRED"
	StatusDelivering = "DELIVERING"
	StatusDelivered  = "DELIVERED"
	StatusConfirmed  = "CONFIRMED"
)

func GenerateLogTitle(status string) string {
	switch status {
	case StatusPaid:
		return "Payment successful. Order is being processed."
	case StatusFailed:
		return "Payment failed."
	case StatusExpired:
		return "Payment expired."
	case StatusDelivering:
		return "Delivery started."
	case StatusDelivered:
		return "Seller delivered the product. Awaiting buyer's confirmation"
	case StatusConfirmed:
		return "Receipt of the product has been confirmed"
	default:
		return "Order updated."
	}
}
