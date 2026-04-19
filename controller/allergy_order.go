package controller

import (
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Calcium-Ion/go-epay/epay"
	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting/operation_setting"
	"github.com/QuantumNous/new-api/setting/system_setting"
	"github.com/gin-gonic/gin"
	"github.com/samber/lo"
	"gorm.io/gorm"
)

type createAllergyOrderRequest struct {
	ServiceCode     string         `json:"service_code"`
	RecipientName   string         `json:"recipient_name"`
	RecipientPhone  string         `json:"recipient_phone"`
	RecipientEmail  string         `json:"recipient_email"`
	ShippingAddress map[string]any `json:"shipping_address"`
}

type requestAllergyOrderPayRequest struct {
	PaymentMethod string `json:"payment_method"`
	SuccessURL    string `json:"success_url"`
	CancelURL     string `json:"cancel_url"`
}

type updateAdminAllergyOrderStatusRequest struct {
	OrderStatus string `json:"order_status"`
	Remark      string `json:"remark"`
}

type upsertAdminAllergyKitRequest struct {
	KitCode            string `json:"kit_code"`
	KitStatus          string `json:"kit_status"`
	OutboundCarrier    string `json:"outbound_carrier"`
	OutboundTrackingNo string `json:"outbound_tracking_no"`
	OutboundShippedAt  string `json:"outbound_shipped_at"`
}

type markAdminAllergySampleReceivedRequest struct {
	ReceivedAt string `json:"received_at"`
	Remark     string `json:"remark"`
}

type markAdminAllergySampleSentBackRequest struct {
	SentBackAt       string `json:"sent_back_at"`
	ReturnTrackingNo string `json:"return_tracking_no"`
	Remark           string `json:"remark"`
}

type startAdminAllergyTestingRequest struct {
	StartedAt string `json:"started_at"`
	Remark    string `json:"remark"`
}

type completeAdminAllergyOrderRequest struct {
	CompletedAt string `json:"completed_at"`
	Remark      string `json:"remark"`
}

type sendAdminAllergyReportEmailRequest struct {
	TargetEmail string `json:"target_email"`
}

func buildAllergyAvailablePaymentMethods() []gin.H {
	items := make([]gin.H, 0, len(operation_setting.PayMethods))
	for _, payMethod := range operation_setting.PayMethods {
		code := strings.TrimSpace(payMethod["type"])
		if code == "" {
			continue
		}
		label := strings.TrimSpace(payMethod["name"])
		if label == "" {
			label = code
		}
		items = append(items, gin.H{
			"code":  code,
			"label": label,
		})
	}
	return items
}

func CreateAllergyOrder(c *gin.Context) {
	var req createAllergyOrderRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	serviceProduct, err := model.GetPublishedAllergyServiceProductByCode(req.ServiceCode)
	if err != nil {
		common.ApiErrorMsg(c, "服务不存在")
		return
	}
	if strings.TrimSpace(req.RecipientName) == "" || strings.TrimSpace(req.RecipientPhone) == "" {
		common.ApiErrorMsg(c, "收件人信息不能为空")
		return
	}
	email, err := validateAllergyEmail(req.RecipientEmail)
	if err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}
	shippingJSONBytes, err := common.Marshal(req.ShippingAddress)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	order, err := model.CreateAllergyOrder(
		c.GetInt("id"),
		serviceProduct.ServiceCode,
		serviceProduct.Title,
		serviceProduct.PriceCents,
		req.RecipientName,
		req.RecipientPhone,
		email,
		string(shippingJSONBytes),
	)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{
		"order_id":       order.ID,
		"order_no":       order.OrderNo,
		"payment_status": order.PaymentStatus,
		"order_status":   order.OrderStatus,
	})
}

func ListAllergyOrders(c *gin.Context) {
	orders, err := model.ListAllergyOrdersByUser(c.GetInt("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	items := make([]gin.H, 0, len(orders))
	for _, order := range orders {
		items = append(items, gin.H{
			"order_id":        order.ID,
			"order_no":        order.OrderNo,
			"service_name":    order.ServiceNameSnapshot,
			"payment_status":  order.PaymentStatus,
			"order_status":    order.OrderStatus,
			"created_at":      order.CreatedAt.Format(time.RFC3339),
			"report_ready_at": formatOptionalTime(order.ReportReadyAt),
		})
	}
	common.ApiSuccess(c, items)
}

func GetAllergyOrderDetail(c *gin.Context) {
	orderID, ok := parseInt64Param(c, "id")
	if !ok {
		common.ApiErrorMsg(c, "订单参数错误")
		return
	}
	order, err := model.GetAllergyOrderForUser(c.GetInt("id"), orderID)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	kit, err := model.GetSampleKitByOrderID(order.ID)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	shippingAddress := decodeJSONStringToMap(order.ShippingAddressJSON)
	response := gin.H{
		"order_id":                  order.ID,
		"order_no":                  order.OrderNo,
		"service_name":              order.ServiceNameSnapshot,
		"service_price_cents":       order.ServicePriceCents,
		"currency":                  order.Currency,
		"payment_status":            order.PaymentStatus,
		"order_status":              order.OrderStatus,
		"recipient_name":            order.RecipientName,
		"recipient_phone":           order.RecipientPhone,
		"recipient_email":           order.RecipientEmail,
		"shipping_address":          shippingAddress,
		"available_payment_methods": buildAllergyAvailablePaymentMethods(),
		"sample_kit":                nil,
	}
	if kit != nil {
		response["sample_kit"] = gin.H{
			"kit_code":             kit.KitCode,
			"kit_status":           kit.Status,
			"outbound_tracking_no": kit.TrackingNumber,
		}
	}
	common.ApiSuccess(c, response)
}

func CancelAllergyOrder(c *gin.Context) {
	orderID, ok := parseInt64Param(c, "id")
	if !ok {
		common.ApiErrorMsg(c, "订单参数错误")
		return
	}
	order, err := model.CancelAllergyOrder(c.GetInt("id"), orderID)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{
		"order_id":       order.ID,
		"payment_status": order.PaymentStatus,
		"order_status":   order.OrderStatus,
		"cancelled_at":   formatOptionalTime(order.CancelledAt),
	})
}

func RequestAllergyOrderEpay(c *gin.Context) {
	orderID, ok := parseInt64Param(c, "id")
	if !ok {
		common.ApiErrorMsg(c, "订单参数错误")
		return
	}
	var req requestAllergyOrderPayRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	if !operation_setting.ContainsPayMethod(req.PaymentMethod) {
		common.ApiErrorMsg(c, "支付方式不存在")
		return
	}
	order, err := model.GetAllergyOrderForUser(c.GetInt("id"), orderID)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if order.PaymentStatus != model.AllergyPaymentStatusPending {
		common.ApiErrorMsg(c, "订单当前状态不可支付")
		return
	}
	client := GetEpayClient()
	if client == nil {
		common.ApiErrorMsg(c, "当前管理员未配置支付信息")
		return
	}
	callbackAddress := service.GetCallbackAddress()
	notifyURL, err := url.Parse(strings.TrimRight(callbackAddress, "/") + "/api/orders/epay/notify")
	if err != nil {
		common.ApiErrorMsg(c, "回调地址配置错误")
		return
	}
	returnTarget, err := validateAllergyRedirectTarget(req.SuccessURL, fmt.Sprintf("%s/orders/%d", strings.TrimRight(system_setting.ServerAddress, "/"), order.ID))
	if err != nil {
		common.ApiErrorMsg(c, "支付返回地址错误")
		return
	}
	cancelTarget, err := validateAllergyRedirectTarget(req.CancelURL, returnTarget)
	if err != nil {
		common.ApiErrorMsg(c, "支付取消地址错误")
		return
	}
	returnURL, err := buildAllergyEpayReturnURL(callbackAddress, returnTarget, cancelTarget)
	if err != nil {
		common.ApiErrorMsg(c, "支付回跳地址错误")
		return
	}
	tradeNo := model.GenerateAllergyPaymentTradeNo(order.ID)
	order, err = model.SetAllergyOrderPaymentRequest(order.ID, c.GetInt("id"), req.PaymentMethod, tradeNo)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	uri, params, err := client.Purchase(&epay.PurchaseArgs{
		Type:           req.PaymentMethod,
		ServiceTradeNo: tradeNo,
		Name:           fmt.Sprintf("ALLERGY:%s", order.ServiceNameSnapshot),
		Money:          fmt.Sprintf("%.2f", float64(order.ServicePriceCents)/100),
		Device:         epay.PC,
		NotifyUrl:      notifyURL,
		ReturnUrl:      returnURL,
	})
	if err != nil {
		common.ApiErrorMsg(c, "拉起支付失败")
		return
	}
	common.ApiSuccess(c, gin.H{
		"payment_method": order.PaymentMethod,
		"trade_no":       tradeNo,
		"redirect_url":   uri,
		"form_data":      params,
		"payment_status": order.PaymentStatus,
	})
}

func AllergyOrderEpayNotify(c *gin.Context) {
	params, ok := parseEpayRequestParams(c)
	if !ok {
		_, _ = c.Writer.Write([]byte("fail"))
		return
	}
	client := GetEpayClient()
	if client == nil {
		_, _ = c.Writer.Write([]byte("fail"))
		return
	}
	verifyInfo, err := client.Verify(params)
	if err != nil || !verifyInfo.VerifyStatus || verifyInfo.TradeStatus != epay.StatusTradeSuccess {
		_, _ = c.Writer.Write([]byte("fail"))
		return
	}

	LockOrder(verifyInfo.ServiceTradeNo)
	defer UnlockOrder(verifyInfo.ServiceTradeNo)
	if _, err := model.CompleteAllergyOrderPayment(verifyInfo.ServiceTradeNo, verifyInfo.TradeNo, common.GetJsonString(verifyInfo)); err != nil {
		_, _ = c.Writer.Write([]byte("fail"))
		return
	}
	_, _ = c.Writer.Write([]byte("success"))
}

func AllergyOrderEpayReturn(c *gin.Context) {
	successTarget := normalizeAllergyRedirectTarget(c.Query("success"), strings.TrimRight(system_setting.ServerAddress, "/")+"/orders")
	cancelTarget := normalizeAllergyRedirectTarget(c.Query("cancel"), successTarget)
	redirectTarget := cancelTarget
	params, ok := parseEpayRequestParams(c)
	if ok {
		client := GetEpayClient()
		if client != nil {
			if verifyInfo, err := client.Verify(params); err == nil && verifyInfo.VerifyStatus && verifyInfo.TradeStatus == epay.StatusTradeSuccess {
				LockOrder(verifyInfo.ServiceTradeNo)
				_, _ = model.CompleteAllergyOrderPayment(verifyInfo.ServiceTradeNo, verifyInfo.TradeNo, common.GetJsonString(verifyInfo))
				UnlockOrder(verifyInfo.ServiceTradeNo)
				redirectTarget = successTarget
			}
		}
	}
	c.Redirect(http.StatusFound, redirectTarget)
}

func GetAllergyOrderPayStatus(c *gin.Context) {
	orderID, ok := parseInt64Param(c, "id")
	if !ok {
		common.ApiErrorMsg(c, "订单参数错误")
		return
	}
	order, err := model.GetAllergyOrderForUser(c.GetInt("id"), orderID)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{
		"order_id":       order.ID,
		"payment_status": order.PaymentStatus,
		"order_status":   order.OrderStatus,
		"paid_at":        formatOptionalTime(order.PaidAt),
	})
}

func GetAllergyOrderTimeline(c *gin.Context) {
	orderID, ok := parseInt64Param(c, "id")
	if !ok {
		common.ApiErrorMsg(c, "订单参数错误")
		return
	}
	events, err := model.GetAllergyOrderTimelineForUser(c.GetInt("id"), orderID)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	items := make([]gin.H, 0, len(events))
	for _, event := range events {
		items = append(items, gin.H{
			"event_type":  event.EventType,
			"title":       event.EventTitle,
			"description": event.EventDesc,
			"occurred_at": event.OccurredAt.Format(time.RFC3339),
		})
	}
	common.ApiSuccess(c, items)
}

func GetAllergyOrderReport(c *gin.Context) {
	orderID, ok := parseInt64Param(c, "id")
	if !ok {
		common.ApiErrorMsg(c, "订单参数错误")
		return
	}
	order, err := model.GetAllergyOrderForUser(c.GetInt("id"), orderID)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	report, err := model.GetCurrentAllergyReportForOrder(order.ID)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if report == nil {
		common.ApiSuccess(c, nil)
		return
	}
	common.ApiSuccess(c, gin.H{
		"report_id":     report.ID,
		"report_title":  report.ReportTitle,
		"report_status": report.Status,
		"published_at":  formatOptionalTime(report.PublishedAt),
		"preview_url":   fmt.Sprintf("/api/reports/%d/preview", report.ID),
		"download_url":  fmt.Sprintf("/api/reports/%d/download", report.ID),
	})
}

func PreviewAllergyReport(c *gin.Context) {
	serveAllergyReportFile(c, "inline")
}

func DownloadAllergyReport(c *gin.Context) {
	serveAllergyReportFile(c, "attachment")
}

func PreviewAdminAllergyReport(c *gin.Context) {
	serveAdminAllergyReportFile(c, "inline")
}

func DownloadAdminAllergyReport(c *gin.Context) {
	serveAdminAllergyReportFile(c, "attachment")
}

func ListAdminAllergyOrders(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	orders, total, err := model.ListAdminAllergyOrders(model.AllergyOrderFilter{
		OrderNo:       c.Query("order_no"),
		Email:         c.Query("email"),
		PaymentStatus: c.Query("payment_status"),
		OrderStatus:   c.Query("order_status"),
	}, pageInfo)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	items := make([]gin.H, 0, len(orders))
	for _, order := range orders {
		items = append(items, gin.H{
			"order_id":        order.ID,
			"order_no":        order.OrderNo,
			"service_name":    order.ServiceNameSnapshot,
			"recipient_name":  order.RecipientName,
			"recipient_email": order.RecipientEmail,
			"payment_status":  order.PaymentStatus,
			"order_status":    order.OrderStatus,
			"paid_at":         formatOptionalTime(order.PaidAt),
			"created_at":      order.CreatedAt.Format(time.RFC3339),
		})
	}
	pageInfo.SetTotal(int(total))
	pageInfo.SetItems(items)
	common.ApiSuccess(c, pageInfo)
}

func GetAdminAllergyOrderDetail(c *gin.Context) {
	orderID, ok := parseInt64Param(c, "id")
	if !ok {
		common.ApiErrorMsg(c, "订单参数错误")
		return
	}
	order, err := model.GetAllergyOrderByID(orderID)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	kit, err := model.GetSampleKitByOrderID(order.ID)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	submission, err := model.GetLabSubmissionByOrderID(order.ID)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	reports, err := model.ListAllergyReportsForOrder(order.ID)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	timeline, err := model.GetAllergyOrderTimeline(order.ID)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	var currentReport *model.LabReport
	for _, item := range reports {
		if item.IsCurrent {
			currentReport = item
			break
		}
	}
	if currentReport == nil && len(reports) > 0 {
		currentReport = reports[0]
	}
	response := gin.H{
		"order_id":                      order.ID,
		"order_no":                      order.OrderNo,
		"service_code":                  order.ServiceCode,
		"service_name":                  order.ServiceNameSnapshot,
		"service_price_cents":           order.ServicePriceCents,
		"currency":                      order.Currency,
		"payment_status":                order.PaymentStatus,
		"order_status":                  order.OrderStatus,
		"payment_method":                order.PaymentMethod,
		"payment_ref":                   order.PaymentRef,
		"payment_provider_order_no":     order.PaymentProviderOrderNo,
		"payment_callback_payload_json": order.PaymentCallbackPayloadJSON,
		"paid_at":                       formatOptionalTime(order.PaidAt),
		"report_ready_at":               formatOptionalTime(order.ReportReadyAt),
		"completed_at":                  formatOptionalTime(order.CompletedAt),
		"cancelled_at":                  formatOptionalTime(order.CancelledAt),
		"admin_remark":                  order.AdminRemark,
		"recipient_email":               order.RecipientEmail,
		"recipient_name":                order.RecipientName,
		"recipient_phone":               order.RecipientPhone,
		"shipping_address":              decodeJSONStringToMap(order.ShippingAddressJSON),
		"created_at":                    order.CreatedAt.Format(time.RFC3339),
		"updated_at":                    order.UpdatedAt.Format(time.RFC3339),
		"sample_kit":                    nil,
		"lab_submission":                nil,
		"current_report":                nil,
		"reports":                       buildAdminAllergyReportItems(reports),
		"timeline":                      buildAdminAllergyTimelineItems(timeline),
	}
	if kit != nil {
		response["sample_kit"] = gin.H{
			"kit_id":               kit.ID,
			"kit_code":             kit.KitCode,
			"kit_status":           kit.Status,
			"outbound_carrier":     kit.TrackingCompany,
			"outbound_tracking_no": kit.TrackingNumber,
			"return_tracking_no":   kit.ReturnTrackingNo,
			"outbound_shipped_at":  formatOptionalTime(kit.ShippedAt),
			"delivered_at":         formatOptionalTime(kit.DeliveredAt),
			"sample_received_at":   formatOptionalTime(kit.SampleReceivedAt),
			"remark":               kit.Remark,
			"created_at":           kit.CreatedAt.Format(time.RFC3339),
			"updated_at":           kit.UpdatedAt.Format(time.RFC3339),
		}
	}
	if submission != nil {
		response["lab_submission"] = gin.H{
			"submission_id":        submission.ID,
			"lab_name":             submission.LabName,
			"submission_no":        submission.SubmissionNo,
			"status":               submission.Status,
			"external_sample_code": submission.ExternalSampleCode,
			"tracking_number":      submission.TrackingNumber,
			"received_at":          formatOptionalTime(submission.ReceivedAt),
			"submitted_at":         formatOptionalTime(submission.SubmittedAt),
			"testing_started_at":   formatOptionalTime(submission.TestingStartedAt),
			"completed_at":         formatOptionalTime(submission.CompletedAt),
			"raw_payload_json":     submission.RawPayloadJSON,
			"remark":               submission.Remark,
			"created_at":           submission.CreatedAt.Format(time.RFC3339),
			"updated_at":           submission.UpdatedAt.Format(time.RFC3339),
		}
	}
	if currentReport != nil {
		response["current_report"] = buildAdminAllergyReportItem(currentReport)
	}
	common.ApiSuccess(c, response)
}

func UpdateAdminAllergyOrderStatus(c *gin.Context) {
	orderID, ok := parseInt64Param(c, "id")
	if !ok {
		common.ApiErrorMsg(c, "订单参数错误")
		return
	}
	var req updateAdminAllergyOrderStatusRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	if !model.IsValidAllergyOrderStatus(req.OrderStatus) {
		common.ApiErrorMsg(c, "订单状态不合法")
		return
	}
	if err := updateAdminAllergyOrderStatus(orderID, req.OrderStatus, req.Remark, c.GetInt("id")); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"order_status": req.OrderStatus})
}

func UpsertAdminAllergyOrderKit(c *gin.Context) {
	orderID, ok := parseInt64Param(c, "id")
	if !ok {
		common.ApiErrorMsg(c, "订单参数错误")
		return
	}
	var req upsertAdminAllergyKitRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	if !model.IsValidAllergyKitStatus(req.KitStatus) {
		common.ApiErrorMsg(c, "采样盒状态不合法")
		return
	}
	shippedAt, err := parseOptionalRFC3339(req.OutboundShippedAt)
	if err != nil {
		common.ApiErrorMsg(c, "寄出时间格式错误")
		return
	}
	kit, err := model.UpsertSampleKitForOrder(orderID, req.KitCode, req.KitStatus, req.OutboundCarrier, req.OutboundTrackingNo, shippedAt, c.GetInt("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{
		"kit_code":             kit.KitCode,
		"kit_status":           kit.Status,
		"outbound_tracking_no": kit.TrackingNumber,
	})
}

func MarkAdminAllergySampleReceived(c *gin.Context) {
	orderID, ok := parseInt64Param(c, "id")
	if !ok {
		common.ApiErrorMsg(c, "订单参数错误")
		return
	}
	var req markAdminAllergySampleReceivedRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	receivedAt := time.Now()
	if strings.TrimSpace(req.ReceivedAt) != "" {
		parsed, err := time.Parse(time.RFC3339, req.ReceivedAt)
		if err != nil {
			common.ApiErrorMsg(c, "签收时间格式错误")
			return
		}
		receivedAt = parsed
	}
	if err := model.MarkAllergySampleReceived(orderID, receivedAt, req.Remark, c.GetInt("id")); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"order_status": model.AllergyOrderStatusSampleReceived})
}

func MarkAdminAllergySampleSentBack(c *gin.Context) {
	orderID, ok := parseInt64Param(c, "id")
	if !ok {
		common.ApiErrorMsg(c, "订单参数错误")
		return
	}
	var req markAdminAllergySampleSentBackRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	sentBackAt := time.Now()
	if strings.TrimSpace(req.SentBackAt) != "" {
		parsed, err := time.Parse(time.RFC3339, req.SentBackAt)
		if err != nil {
			common.ApiErrorMsg(c, "回寄时间格式错误")
			return
		}
		sentBackAt = parsed
	}
	if err := model.MarkAllergySampleSentBack(orderID, sentBackAt, req.ReturnTrackingNo, req.Remark, c.GetInt("id")); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"order_status": model.AllergyOrderStatusSampleReturning})
}

func StartAdminAllergyTesting(c *gin.Context) {
	orderID, ok := parseInt64Param(c, "id")
	if !ok {
		common.ApiErrorMsg(c, "订单参数错误")
		return
	}
	var req startAdminAllergyTestingRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	startedAt := time.Now()
	if strings.TrimSpace(req.StartedAt) != "" {
		parsed, err := time.Parse(time.RFC3339, req.StartedAt)
		if err != nil {
			common.ApiErrorMsg(c, "开始检测时间格式错误")
			return
		}
		startedAt = parsed
	}
	if err := model.StartAllergyOrderTesting(orderID, startedAt, req.Remark, c.GetInt("id")); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"order_status": model.AllergyOrderStatusInTesting})
}

func UploadAdminAllergyOrderReport(c *gin.Context) {
	orderID, ok := parseInt64Param(c, "id")
	if !ok {
		common.ApiErrorMsg(c, "订单参数错误")
		return
	}
	fileHeader, err := c.FormFile("file")
	if err != nil {
		common.ApiErrorMsg(c, "请上传 PDF 文件")
		return
	}
	if !strings.HasSuffix(strings.ToLower(fileHeader.Filename), ".pdf") {
		common.ApiErrorMsg(c, "仅支持 PDF 报告")
		return
	}
	filePath, fileName, fileSize, mimeType, err := saveAllergyReportUpload(fileHeader)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	reportTitle := strings.TrimSpace(c.PostForm("report_title"))
	if reportTitle == "" {
		reportTitle = "过敏原检测报告"
	}
	report, err := model.CreateAllergyLabReport(orderID, reportTitle, fileName, filePath, mimeType, fileSize, c.GetInt("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{
		"report_id":     report.ID,
		"report_status": report.Status,
	})
}

func PublishAdminAllergyReport(c *gin.Context) {
	reportID, ok := parseInt64Param(c, "id")
	if !ok {
		common.ApiErrorMsg(c, "报告参数错误")
		return
	}
	report, err := model.PublishAllergyLabReport(reportID, c.GetInt("id"))
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{
		"report_id":     report.ID,
		"report_status": model.AllergyReportStatusPublished,
	})
}

func SendAdminAllergyReportEmail(c *gin.Context) {
	reportID, ok := parseInt64Param(c, "id")
	if !ok {
		common.ApiErrorMsg(c, "报告参数错误")
		return
	}
	var req sendAdminAllergyReportEmailRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil && c.Request.ContentLength > 0 {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	report, err := model.GetLabReportByID(reportID)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if report.Status != model.AllergyReportStatusPublished {
		common.ApiErrorMsg(c, "报告尚未发布")
		return
	}
	order, err := model.GetAllergyOrderByID(report.OrderID)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	user, err := model.GetUserById(order.UserID, false)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	targetEmail := strings.TrimSpace(req.TargetEmail)
	if targetEmail == "" {
		targetEmail = user.Email
	}
	targetEmail, err = validateAllergyEmail(targetEmail)
	if err != nil {
		common.ApiErrorMsg(c, err.Error())
		return
	}
	allowedTargets := map[string]struct{}{
		model.NormalizeEmail(user.Email):           {},
		model.NormalizeEmail(order.RecipientEmail): {},
	}
	if _, ok := allowedTargets[targetEmail]; !ok {
		common.ApiErrorMsg(c, "目标邮箱不在允许范围内")
		return
	}
	if err := sendAllergyReportEmail(targetEmail, report); err != nil {
		if _, logErr := model.CreateReportDeliveryLog(report.ID, order.ID, targetEmail, "manual_resend", "failed", c.GetInt("id"), err.Error()); logErr != nil {
			common.ApiError(c, logErr)
			return
		}
		common.ApiError(c, err)
		return
	}
	logItem, err := model.CreateReportDeliveryLog(report.ID, order.ID, targetEmail, "manual_resend", "sent", c.GetInt("id"), "")
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{
		"report_id":       report.ID,
		"target_email":    targetEmail,
		"delivery_status": logItem.DeliveryStatus,
	})
}

func ListAdminAllergyReportDeliveryLogs(c *gin.Context) {
	reportID, ok := parseInt64Param(c, "id")
	if !ok {
		common.ApiErrorMsg(c, "报告参数错误")
		return
	}
	logs, err := model.ListReportDeliveryLogs(reportID)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	items := make([]gin.H, 0, len(logs))
	for _, item := range logs {
		items = append(items, gin.H{
			"delivery_channel": item.DeliveryChannel,
			"target":           item.RecipientEmail,
			"status":           item.DeliveryStatus,
			"sent_at":          formatOptionalTime(item.SentAt),
			"created_at":       item.CreatedAt.Format(time.RFC3339),
		})
	}
	common.ApiSuccess(c, items)
}

func CompleteAdminAllergyOrder(c *gin.Context) {
	orderID, ok := parseInt64Param(c, "id")
	if !ok {
		common.ApiErrorMsg(c, "订单参数错误")
		return
	}
	var req completeAdminAllergyOrderRequest
	if err := common.DecodeJson(c.Request.Body, &req); err != nil {
		common.ApiErrorMsg(c, "参数错误")
		return
	}
	completedAt := time.Now()
	if strings.TrimSpace(req.CompletedAt) != "" {
		parsed, err := time.Parse(time.RFC3339, req.CompletedAt)
		if err != nil {
			common.ApiErrorMsg(c, "完成时间格式错误")
			return
		}
		completedAt = parsed
	}
	if err := model.CompleteAllergyOrder(orderID, completedAt, req.Remark, c.GetInt("id")); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"order_status": model.AllergyOrderStatusCompleted})
}

func parseInt64Param(c *gin.Context, key string) (int64, bool) {
	value, err := strconv.ParseInt(strings.TrimSpace(c.Param(key)), 10, 64)
	if err != nil || value <= 0 {
		return 0, false
	}
	return value, true
}

func parseOptionalRFC3339(value string) (*time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

func decodeJSONStringToMap(raw string) map[string]any {
	if strings.TrimSpace(raw) == "" {
		return map[string]any{}
	}
	result := map[string]any{}
	if err := common.UnmarshalJsonStr(raw, &result); err != nil {
		return map[string]any{}
	}
	return result
}

func formatOptionalTime(value *time.Time) any {
	if value == nil || value.IsZero() {
		return nil
	}
	return value.Format(time.RFC3339)
}

func buildAdminAllergyTimelineItems(events []*model.OrderTimelineEvent) []gin.H {
	items := make([]gin.H, 0, len(events))
	for _, event := range events {
		items = append(items, gin.H{
			"event_id":           event.ID,
			"event_type":         event.EventType,
			"title":              event.EventTitle,
			"description":        event.EventDesc,
			"visible_to_user":    event.VisibleToUser,
			"operator_user_id":   event.OperatorUserID,
			"event_payload_json": event.EventPayloadJSON,
			"occurred_at":        event.OccurredAt.Format(time.RFC3339),
			"created_at":         event.CreatedAt.Format(time.RFC3339),
		})
	}
	return items
}

func buildAdminAllergyReportItems(reports []*model.LabReport) []gin.H {
	items := make([]gin.H, 0, len(reports))
	for _, report := range reports {
		items = append(items, buildAdminAllergyReportItem(report))
	}
	return items
}

func buildAdminAllergyReportItem(report *model.LabReport) gin.H {
	return gin.H{
		"report_id":          report.ID,
		"report_title":       report.ReportTitle,
		"report_status":      report.Status,
		"version":            report.Version,
		"is_current":         report.IsCurrent,
		"file_name":          report.FileName,
		"mime_type":          report.MimeType,
		"file_size_bytes":    report.FileSizeBytes,
		"uploaded_at":        formatOptionalTime(report.UploadedAt),
		"published_at":       formatOptionalTime(report.PublishedAt),
		"last_email_sent_at": formatOptionalTime(report.LastEmailSentAt),
		"email_sent_count":   report.EmailSentCount,
		"preview_url":        fmt.Sprintf("/api/admin/reports/%d/preview", report.ID),
		"download_url":       fmt.Sprintf("/api/admin/reports/%d/download", report.ID),
		"remark":             report.Remark,
		"created_at":         report.CreatedAt.Format(time.RFC3339),
		"updated_at":         report.UpdatedAt.Format(time.RFC3339),
	}
}

func validateAllergyRedirectTarget(raw string, fallback string) (string, error) {
	target := strings.TrimSpace(raw)
	if target == "" {
		return fallback, nil
	}
	parsed, err := url.Parse(target)
	if err != nil {
		return "", err
	}
	if !parsed.IsAbs() || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return "", fmt.Errorf("invalid redirect target")
	}
	return parsed.String(), nil
}

func normalizeAllergyRedirectTarget(raw string, fallback string) string {
	target, err := validateAllergyRedirectTarget(raw, fallback)
	if err != nil {
		return fallback
	}
	return target
}

func buildAllergyEpayReturnURL(callbackAddress string, successTarget string, cancelTarget string) (*url.URL, error) {
	returnURL, err := url.Parse(strings.TrimRight(callbackAddress, "/") + "/api/orders/epay/return")
	if err != nil {
		return nil, err
	}
	query := returnURL.Query()
	query.Set("success", successTarget)
	query.Set("cancel", cancelTarget)
	returnURL.RawQuery = query.Encode()
	return returnURL, nil
}

func parseEpayRequestParams(c *gin.Context) (map[string]string, bool) {
	var params map[string]string
	filterParam := func(key string) bool {
		switch key {
		case "success", "cancel":
			return true
		default:
			return false
		}
	}
	if c.Request.Method == http.MethodPost {
		if err := c.Request.ParseForm(); err != nil {
			return nil, false
		}
		params = lo.Reduce(lo.Keys(c.Request.PostForm), func(r map[string]string, t string, i int) map[string]string {
			if filterParam(t) {
				return r
			}
			r[t] = c.Request.PostForm.Get(t)
			return r
		}, map[string]string{})
	} else {
		params = lo.Reduce(lo.Keys(c.Request.URL.Query()), func(r map[string]string, t string, i int) map[string]string {
			if filterParam(t) {
				return r
			}
			r[t] = c.Request.URL.Query().Get(t)
			return r
		}, map[string]string{})
	}
	return params, len(params) > 0
}

func serveAllergyReportFile(c *gin.Context, disposition string) {
	reportID, ok := parseInt64Param(c, "id")
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "报告参数错误"})
		return
	}
	report, _, err := model.GetAllergyReportForUser(c.GetInt("id"), reportID)
	if err != nil {
		c.JSON(http.StatusForbidden, gin.H{"success": false, "message": err.Error()})
		return
	}
	if strings.TrimSpace(report.FilePath) == "" {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "报告文件不存在"})
		return
	}
	if _, err := os.Stat(report.FilePath); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "报告文件不存在"})
		return
	}
	fileName := report.FileName
	if fileName == "" {
		fileName = "report.pdf"
	}
	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", fmt.Sprintf("%s; filename=\"%s\"", disposition, fileName))
	c.File(report.FilePath)
}

func serveAdminAllergyReportFile(c *gin.Context, disposition string) {
	reportID, ok := parseInt64Param(c, "id")
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"success": false, "message": "报告参数错误"})
		return
	}
	report, err := model.GetLabReportByID(reportID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "报告不存在"})
		return
	}
	if strings.TrimSpace(report.FilePath) == "" {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "报告文件不存在"})
		return
	}
	if _, err := os.Stat(report.FilePath); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "message": "报告文件不存在"})
		return
	}
	fileName := report.FileName
	if fileName == "" {
		fileName = "report.pdf"
	}
	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", fmt.Sprintf("%s; filename=\"%s\"", disposition, fileName))
	c.File(report.FilePath)
}

func updateAdminAllergyOrderStatus(orderID int64, orderStatus string, remark string, operatorUserID int) error {
	now := time.Now()
	remark = strings.TrimSpace(remark)
	return model.DB.Transaction(func(tx *gorm.DB) error {
		var order model.AllergyOrder
		if err := tx.First(&order, orderID).Error; err != nil {
			return err
		}

		updates := map[string]any{
			"order_status":    orderStatus,
			"admin_remark":    remark,
			"updated_at":      now,
			"completed_at":    order.CompletedAt,
			"cancelled_at":    order.CancelledAt,
			"report_ready_at": order.ReportReadyAt,
		}
		switch orderStatus {
		case model.AllergyOrderStatusReportReady:
			updates["report_ready_at"] = &now
		case model.AllergyOrderStatusCompleted:
			updates["completed_at"] = &now
		case model.AllergyOrderStatusCancelled:
			updates["cancelled_at"] = &now
		}
		if err := tx.Model(&order).Updates(updates).Error; err != nil {
			return err
		}

		eventTitle, eventDesc, visibleToUser := allergyOrderStatusTimelineMeta(orderStatus, remark)
		if eventTitle == "" {
			return nil
		}
		event := &model.OrderTimelineEvent{
			OrderID:        orderID,
			EventType:      orderStatus,
			EventTitle:     eventTitle,
			EventDesc:      eventDesc,
			VisibleToUser:  visibleToUser,
			OperatorUserID: operatorUserID,
			OccurredAt:     now,
			CreatedAt:      now,
		}
		return tx.Create(event).Error
	})
}

func allergyOrderStatusTimelineMeta(orderStatus string, remark string) (string, string, bool) {
	defaultDesc := remark
	switch orderStatus {
	case model.AllergyOrderStatusPendingPayment:
		if defaultDesc == "" {
			defaultDesc = "订单待支付"
		}
		return "订单待支付", defaultDesc, true
	case model.AllergyOrderStatusPaid:
		if defaultDesc == "" {
			defaultDesc = "订单支付已完成"
		}
		return "订单已支付", defaultDesc, true
	case model.AllergyOrderStatusKitPreparing:
		if defaultDesc == "" {
			defaultDesc = "我们正在准备采样盒"
		}
		return "采样盒准备中", defaultDesc, true
	case model.AllergyOrderStatusKitShipped:
		if defaultDesc == "" {
			defaultDesc = "采样盒已寄出，请留意物流信息"
		}
		return "采样盒已寄出", defaultDesc, true
	case model.AllergyOrderStatusSampleReturning:
		if defaultDesc == "" {
			defaultDesc = "样本回寄中"
		}
		return "样本回寄中", defaultDesc, true
	case model.AllergyOrderStatusSampleReceived:
		if defaultDesc == "" {
			defaultDesc = "检测机构已收到样本"
		}
		return "样本已签收", defaultDesc, true
	case model.AllergyOrderStatusInTesting:
		if defaultDesc == "" {
			defaultDesc = "检测机构正在分析样本"
		}
		return "检测中", defaultDesc, true
	case model.AllergyOrderStatusReportReady:
		if defaultDesc == "" {
			defaultDesc = "检测报告已生成，可在线查看"
		}
		return "检测报告已就绪", defaultDesc, true
	case model.AllergyOrderStatusCompleted:
		if defaultDesc == "" {
			defaultDesc = "订单已完成"
		}
		return "订单已完成", defaultDesc, true
	case model.AllergyOrderStatusCancelled:
		if defaultDesc == "" {
			defaultDesc = "订单已取消"
		}
		return "订单已取消", defaultDesc, true
	default:
		return "", "", false
	}
}

func allergyReportStorageDir() string {
	if dir := strings.TrimSpace(os.Getenv("ALLERGY_REPORT_STORAGE_DIR")); dir != "" {
		return dir
	}
	return filepath.Join("storage", "allergy-reports")
}

func saveAllergyReportUpload(fileHeader *multipart.FileHeader) (string, string, int64, string, error) {
	src, err := fileHeader.Open()
	if err != nil {
		return "", "", 0, "", err
	}
	defer src.Close()

	dir := allergyReportStorageDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", "", 0, "", err
	}
	fileName := fmt.Sprintf("%d_%s", time.Now().UnixNano(), filepath.Base(fileHeader.Filename))
	path := filepath.Join(dir, fileName)
	dst, err := os.Create(path)
	if err != nil {
		return "", "", 0, "", err
	}
	defer dst.Close()
	size, err := io.Copy(dst, src)
	if err != nil {
		return "", "", 0, "", err
	}
	return path, fileHeader.Filename, size, "application/pdf", nil
}

func sendAllergyReportEmail(targetEmail string, report *model.LabReport) error {
	subject := fmt.Sprintf("%s %s", common.SystemName, report.ReportTitle)
	content := fmt.Sprintf("<p>您好，附件为您的 %s。</p>", report.ReportTitle)
	if strings.TrimSpace(common.SMTPServer) == "" && strings.TrimSpace(common.SMTPAccount) == "" {
		common.SysLog(fmt.Sprintf("[allergy-report] simulate email to %s for report %d", common.MaskEmail(targetEmail), report.ID))
		return nil
	}
	return common.SendEmailWithAttachments(subject, targetEmail, content, []common.EmailAttachment{
		{
			FileName:    report.FileName,
			ContentType: "application/pdf",
			Path:        report.FilePath,
		},
	})
}
