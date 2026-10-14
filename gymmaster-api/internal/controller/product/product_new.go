package product

import (
	"gymmaster-api/api/product"
	"gymmaster-api/internal/service"
)

type ControllerV1 struct {
	productService service.ProductService
}

func NewV1(productService service.ProductService) product.IProductV1 {
	return &ControllerV1{productService: productService}
}
