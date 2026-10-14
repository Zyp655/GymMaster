package inventory

import (
	"gymmaster-api/api/inventory"
	"gymmaster-api/internal/service"
)

type ControllerV1 struct {
	stockService service.StockService
}

func NewV1(stockService service.StockService) inventory.IInventoryV1 {
	return &ControllerV1{stockService: stockService}
}
