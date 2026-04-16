package model

import (
	"errors"
	"fmt"
	"strings"

	"github.com/QuantumNous/new-api/common"
)

const (
	AllergyServiceProductStatusDraft     = "draft"
	AllergyServiceProductStatusPublished = "published"
	AllergyServiceProductStatusArchived  = "archived"
	AllergyServiceProductCurrencyCNY     = "CNY"
)

type AllergyServiceProductInput struct {
	ServiceCode string
	Title       string
	Description string
	ImageURL    string
	CTAText     string
	Tag         string
	PriceCents  int
	SortOrder   int
	Status      string
}

func IsValidAllergyServiceProductStatus(status string) bool {
	switch strings.TrimSpace(status) {
	case AllergyServiceProductStatusDraft,
		AllergyServiceProductStatusPublished,
		AllergyServiceProductStatusArchived:
		return true
	default:
		return false
	}
}

func NormalizeAllergyServiceCode(code string) (string, error) {
	code = strings.TrimSpace(code)
	if len(code) < 3 || len(code) > 64 {
		return "", errors.New("服务编码长度必须为 3-64 位")
	}
	for _, r := range code {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			continue
		}
		return "", errors.New("服务编码仅允许小写字母、数字和连字符")
	}
	if code[0] == '-' || code[len(code)-1] == '-' {
		return "", errors.New("服务编码不能以连字符开头或结尾")
	}
	return code, nil
}

func defaultAllergyServiceProducts() []*AllergyServiceProduct {
	return []*AllergyServiceProduct{
		{
			ServiceCode: "allergy-test-basic",
			Title:       "埃勒吉居家过敏原检测服务",
			Description: "通过一滴指尖血，精准检测100+种过敏原。APP根据结果生成个性化回避建议与营养补充方案。",
			ImageURL:    "https://picsum.photos/600/400?random=5",
			CTAText:     "立即购买",
			Tag:         "最受欢迎",
			PriceCents:  19900,
			Currency:    AllergyServiceProductCurrencyCNY,
			SortOrder:   10,
			Status:      AllergyServiceProductStatusPublished,
		},
	}
}

func EnsureDefaultAllergyServiceProducts() error {
	for _, seed := range defaultAllergyServiceProducts() {
		var product AllergyServiceProduct
		err := DB.Where("service_code = ?", seed.ServiceCode).Attrs(seed).FirstOrCreate(&product).Error
		if err != nil {
			return err
		}
	}
	return nil
}

func ListPublishedAllergyServiceProducts() ([]*AllergyServiceProduct, error) {
	var products []*AllergyServiceProduct
	err := DB.Where("status = ?", AllergyServiceProductStatusPublished).
		Order("sort_order asc, id desc").
		Find(&products).Error
	return products, err
}

func ListAdminAllergyServiceProducts(pageInfo *common.PageInfo) ([]*AllergyServiceProduct, int64, error) {
	var products []*AllergyServiceProduct
	var total int64
	query := DB.Model(&AllergyServiceProduct{})
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := query.Order("sort_order asc, id desc").Limit(pageInfo.GetPageSize()).Offset(pageInfo.GetStartIdx()).Find(&products).Error; err != nil {
		return nil, 0, err
	}
	return products, total, nil
}

func GetAllergyServiceProductByID(id int64) (*AllergyServiceProduct, error) {
	var product AllergyServiceProduct
	err := DB.First(&product, id).Error
	if err != nil {
		return nil, err
	}
	return &product, nil
}

func GetPublishedAllergyServiceProductByCode(code string) (*AllergyServiceProduct, error) {
	code, err := NormalizeAllergyServiceCode(code)
	if err != nil {
		return nil, err
	}
	var product AllergyServiceProduct
	err = DB.Where("service_code = ? AND status = ?", code, AllergyServiceProductStatusPublished).First(&product).Error
	if err != nil {
		return nil, err
	}
	return &product, nil
}

func CreateAllergyServiceProduct(input AllergyServiceProductInput) (*AllergyServiceProduct, error) {
	code, err := NormalizeAllergyServiceCode(input.ServiceCode)
	if err != nil {
		return nil, err
	}
	product := &AllergyServiceProduct{
		ServiceCode: code,
		Title:       strings.TrimSpace(input.Title),
		Description: strings.TrimSpace(input.Description),
		ImageURL:    strings.TrimSpace(input.ImageURL),
		CTAText:     strings.TrimSpace(input.CTAText),
		Tag:         strings.TrimSpace(input.Tag),
		PriceCents:  input.PriceCents,
		Currency:    AllergyServiceProductCurrencyCNY,
		SortOrder:   input.SortOrder,
		Status:      strings.TrimSpace(input.Status),
	}
	if err := normalizeAndValidateAllergyServiceProduct(product); err != nil {
		return nil, err
	}
	if err := DB.Create(product).Error; err != nil {
		return nil, err
	}
	return product, nil
}

func UpdateAllergyServiceProduct(id int64, input AllergyServiceProductInput) (*AllergyServiceProduct, error) {
	product, err := GetAllergyServiceProductByID(id)
	if err != nil {
		return nil, err
	}
	product.Title = strings.TrimSpace(input.Title)
	product.Description = strings.TrimSpace(input.Description)
	product.ImageURL = strings.TrimSpace(input.ImageURL)
	product.CTAText = strings.TrimSpace(input.CTAText)
	product.Tag = strings.TrimSpace(input.Tag)
	product.PriceCents = input.PriceCents
	product.Currency = AllergyServiceProductCurrencyCNY
	product.SortOrder = input.SortOrder
	if strings.TrimSpace(input.Status) != "" {
		product.Status = strings.TrimSpace(input.Status)
	}
	if err := normalizeAndValidateAllergyServiceProduct(product); err != nil {
		return nil, err
	}
	if err := DB.Save(product).Error; err != nil {
		return nil, err
	}
	return product, nil
}

func SetAllergyServiceProductStatus(id int64, status string) (*AllergyServiceProduct, error) {
	product, err := GetAllergyServiceProductByID(id)
	if err != nil {
		return nil, err
	}
	product.Status = strings.TrimSpace(status)
	if err := normalizeAndValidateAllergyServiceProduct(product); err != nil {
		return nil, err
	}
	if err := DB.Save(product).Error; err != nil {
		return nil, err
	}
	return product, nil
}

func normalizeAndValidateAllergyServiceProduct(product *AllergyServiceProduct) error {
	if product == nil {
		return errors.New("检测项目不能为空")
	}
	if _, err := NormalizeAllergyServiceCode(product.ServiceCode); err != nil {
		return err
	}
	product.ServiceCode = strings.TrimSpace(product.ServiceCode)
	product.Title = strings.TrimSpace(product.Title)
	product.Description = strings.TrimSpace(product.Description)
	product.ImageURL = strings.TrimSpace(product.ImageURL)
	product.CTAText = strings.TrimSpace(product.CTAText)
	product.Tag = strings.TrimSpace(product.Tag)
	product.Currency = AllergyServiceProductCurrencyCNY
	product.Status = strings.TrimSpace(product.Status)
	if product.Status == "" {
		product.Status = AllergyServiceProductStatusDraft
	}
	if product.CTAText == "" {
		product.CTAText = "立即购买"
	}
	if product.Title == "" {
		return errors.New("项目标题不能为空")
	}
	if product.Description == "" {
		return errors.New("项目详情不能为空")
	}
	if product.PriceCents <= 0 {
		return errors.New("项目价格必须大于 0")
	}
	if !IsValidAllergyServiceProductStatus(product.Status) {
		return fmt.Errorf("项目状态无效: %s", product.Status)
	}
	return nil
}
