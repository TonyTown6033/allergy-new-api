package controller

import (
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

type adminAllergyServiceProductRequest struct {
	ServiceCode string `json:"service_code"`
	Title       string `json:"title"`
	Description string `json:"description"`
	ImageURL    string `json:"image_url"`
	CTAText     string `json:"cta_text"`
	Tag         string `json:"tag"`
	PriceCents  int    `json:"price_cents"`
	SortOrder   int    `json:"sort_order"`
	Status      string `json:"status"`
}

func ListAdminAllergyServiceProducts(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	products, total, err := model.ListAdminAllergyServiceProducts(pageInfo)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	items := make([]gin.H, 0, len(products))
	for _, product := range products {
		items = append(items, buildAdminAllergyServiceProductItem(product))
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(items)
	common.ApiSuccess(c, pageInfo)
}

func GetAdminAllergyServiceProduct(c *gin.Context) {
	productID, ok := parseInt64Param(c, "id")
	if !ok {
		common.ApiErrorMsg(c, "检测项目参数错误")
		return
	}
	product, err := model.GetAllergyServiceProductByID(productID)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, buildAdminAllergyServiceProductDetail(product))
}

func CreateAdminAllergyServiceProduct(c *gin.Context) {
	var req adminAllergyServiceProductRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	product, err := model.CreateAllergyServiceProduct(model.AllergyServiceProductInput{
		ServiceCode: req.ServiceCode,
		Title:       req.Title,
		Description: req.Description,
		ImageURL:    req.ImageURL,
		CTAText:     req.CTAText,
		Tag:         req.Tag,
		PriceCents:  req.PriceCents,
		SortOrder:   req.SortOrder,
		Status:      req.Status,
	})
	if err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}
	common.ApiSuccess(c, buildAdminAllergyServiceProductDetail(product))
}

func UpdateAdminAllergyServiceProduct(c *gin.Context) {
	productID, ok := parseInt64Param(c, "id")
	if !ok {
		common.ApiErrorMsg(c, "检测项目参数错误")
		return
	}
	var req adminAllergyServiceProductRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	product, err := model.UpdateAllergyServiceProduct(productID, model.AllergyServiceProductInput{
		Title:       req.Title,
		Description: req.Description,
		ImageURL:    req.ImageURL,
		CTAText:     req.CTAText,
		Tag:         req.Tag,
		PriceCents:  req.PriceCents,
		SortOrder:   req.SortOrder,
		Status:      req.Status,
	})
	if err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}
	common.ApiSuccess(c, buildAdminAllergyServiceProductDetail(product))
}

func PublishAdminAllergyServiceProduct(c *gin.Context) {
	updateAdminAllergyServiceProductStatus(c, model.AllergyServiceProductStatusPublished)
}

func ArchiveAdminAllergyServiceProduct(c *gin.Context) {
	updateAdminAllergyServiceProductStatus(c, model.AllergyServiceProductStatusArchived)
}

func updateAdminAllergyServiceProductStatus(c *gin.Context, status string) {
	productID, ok := parseInt64Param(c, "id")
	if !ok {
		common.ApiErrorMsg(c, "检测项目参数错误")
		return
	}
	product, err := model.SetAllergyServiceProductStatus(productID, status)
	if err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}
	common.ApiSuccess(c, buildAdminAllergyServiceProductDetail(product))
}

func buildAdminAllergyServiceProductItem(product *model.AllergyServiceProduct) gin.H {
	return gin.H{
		"id":           product.ID,
		"service_code": product.ServiceCode,
		"title":        product.Title,
		"price_cents":  product.PriceCents,
		"currency":     product.Currency,
		"status":       product.Status,
		"sort_order":   product.SortOrder,
		"updated_at":   product.UpdatedAt.Format(time.RFC3339),
	}
}

func buildAdminAllergyServiceProductDetail(product *model.AllergyServiceProduct) gin.H {
	return gin.H{
		"id":           product.ID,
		"service_code": product.ServiceCode,
		"title":        product.Title,
		"description":  product.Description,
		"image_url":    product.ImageURL,
		"cta_text":     product.CTAText,
		"tag":          product.Tag,
		"price_cents":  product.PriceCents,
		"currency":     product.Currency,
		"sort_order":   product.SortOrder,
		"status":       product.Status,
		"created_at":   product.CreatedAt.Format(time.RFC3339),
		"updated_at":   product.UpdatedAt.Format(time.RFC3339),
	}
}
