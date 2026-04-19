package controller

import (
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
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

var allowedAllergyServiceProductImageExtensions = map[string]struct{}{
	".jpg":  {},
	".jpeg": {},
	".png":  {},
	".gif":  {},
	".webp": {},
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

func UploadAdminAllergyServiceProductImage(c *gin.Context) {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		common.ApiErrorMsg(c, "请上传图片文件")
		return
	}
	if err := validateAllergyServiceProductImage(fileHeader); err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}
	imageURL, err := saveAllergyServiceProductImage(fileHeader)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{
		"image_url": imageURL,
	})
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

func ServeAllergyServiceProductImage(c *gin.Context) {
	fileName := strings.TrimSpace(c.Param("file_name"))
	if fileName == "" || fileName != filepath.Base(fileName) {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
	filePath := filepath.Join(allergyServiceProductImageStorageDir(), fileName)
	if _, err := os.Stat(filePath); err != nil {
		c.AbortWithStatus(http.StatusNotFound)
		return
	}
	c.File(filePath)
}

func allergyServiceProductImageStorageDir() string {
	if dir := strings.TrimSpace(os.Getenv("ALLERGY_PRODUCT_IMAGE_STORAGE_DIR")); dir != "" {
		return dir
	}
	return filepath.Join("storage", "allergy-product-images")
}

func validateAllergyServiceProductImage(fileHeader *multipart.FileHeader) error {
	if fileHeader == nil || fileHeader.Size <= 0 {
		return errors.New("请上传图片文件")
	}
	extension := strings.ToLower(filepath.Ext(strings.TrimSpace(fileHeader.Filename)))
	if _, ok := allowedAllergyServiceProductImageExtensions[extension]; !ok {
		return errors.New("仅支持图片文件")
	}
	file, err := fileHeader.Open()
	if err != nil {
		return err
	}
	defer file.Close()

	header := make([]byte, 512)
	n, err := file.Read(header)
	if err != nil && !errors.Is(err, io.EOF) {
		return err
	}
	if n == 0 {
		return errors.New("请上传图片文件")
	}
	if contentType := http.DetectContentType(header[:n]); !strings.HasPrefix(contentType, "image/") {
		return errors.New("仅支持图片文件")
	}
	return nil
}

func saveAllergyServiceProductImage(fileHeader *multipart.FileHeader) (string, error) {
	src, err := fileHeader.Open()
	if err != nil {
		return "", err
	}
	defer src.Close()

	dir := allergyServiceProductImageStorageDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}

	extension := strings.ToLower(filepath.Ext(strings.TrimSpace(fileHeader.Filename)))
	fileName := fmt.Sprintf("%d_%s%s", time.Now().UnixNano(), strings.ToLower(common.GetRandomString(6)), extension)
	filePath := filepath.Join(dir, fileName)

	dst, err := os.Create(filePath)
	if err != nil {
		return "", err
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return "", err
	}
	return fmt.Sprintf("/uploads/allergy-product-images/%s", fileName), nil
}
