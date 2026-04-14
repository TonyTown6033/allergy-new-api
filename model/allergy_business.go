package model

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

const (
	AllergyPaymentStatusPending   = "pending"
	AllergyPaymentStatusPaid      = "paid"
	AllergyPaymentStatusRefunded  = "refunded"
	AllergyPaymentStatusCancelled = "cancelled"

	AllergyOrderStatusPendingPayment  = "pending_payment"
	AllergyOrderStatusPaid            = "paid"
	AllergyOrderStatusKitPreparing    = "kit_preparing"
	AllergyOrderStatusKitShipped      = "kit_shipped"
	AllergyOrderStatusSampleReturning = "sample_returning"
	AllergyOrderStatusSampleReceived  = "sample_received"
	AllergyOrderStatusInTesting       = "in_testing"
	AllergyOrderStatusReportReady     = "report_ready"
	AllergyOrderStatusCompleted       = "completed"
	AllergyOrderStatusCancelled       = "cancelled"

	AllergyKitStatusPrepared       = "prepared"
	AllergyKitStatusShipped        = "shipped"
	AllergyKitStatusDelivered      = "delivered"
	AllergyKitStatusSampleSentBack = "sample_sent_back"
	AllergyKitStatusSampleReceived = "sample_received"

	AllergyReportStatusUploaded  = "uploaded"
	AllergyReportStatusPublished = "published"
	AllergyReportStatusRevoked   = "revoked"
)

type AllergyOrderFilter struct {
	OrderNo       string
	Email         string
	PaymentStatus string
	OrderStatus   string
}

func IsValidAllergyOrderStatus(status string) bool {
	switch strings.TrimSpace(status) {
	case AllergyOrderStatusPendingPayment,
		AllergyOrderStatusPaid,
		AllergyOrderStatusKitPreparing,
		AllergyOrderStatusKitShipped,
		AllergyOrderStatusSampleReturning,
		AllergyOrderStatusSampleReceived,
		AllergyOrderStatusInTesting,
		AllergyOrderStatusReportReady,
		AllergyOrderStatusCompleted,
		AllergyOrderStatusCancelled:
		return true
	default:
		return false
	}
}

func IsValidAllergyKitStatus(status string) bool {
	switch strings.TrimSpace(status) {
	case AllergyKitStatusPrepared,
		AllergyKitStatusShipped,
		AllergyKitStatusDelivered,
		AllergyKitStatusSampleSentBack,
		AllergyKitStatusSampleReceived:
		return true
	default:
		return false
	}
}

func CreateOrderTimelineEvent(orderID int64, eventType string, title string, desc string, visibleToUser bool, operatorUserID int, payloadJSON string, occurredAt time.Time) error {
	if orderID <= 0 {
		return errors.New("订单不存在")
	}
	if occurredAt.IsZero() {
		occurredAt = time.Now()
	}
	event := &OrderTimelineEvent{
		OrderID:          orderID,
		EventType:        eventType,
		EventTitle:       title,
		EventDesc:        desc,
		VisibleToUser:    visibleToUser,
		OperatorUserID:   operatorUserID,
		EventPayloadJSON: payloadJSON,
		OccurredAt:       occurredAt,
		CreatedAt:        time.Now(),
	}
	return DB.Create(event).Error
}

func createOrderTimelineEventTx(tx *gorm.DB, orderID int64, eventType string, title string, desc string, visibleToUser bool, operatorUserID int, payloadJSON string, occurredAt time.Time) error {
	if occurredAt.IsZero() {
		occurredAt = time.Now()
	}
	event := &OrderTimelineEvent{
		OrderID:          orderID,
		EventType:        eventType,
		EventTitle:       title,
		EventDesc:        desc,
		VisibleToUser:    visibleToUser,
		OperatorUserID:   operatorUserID,
		EventPayloadJSON: payloadJSON,
		OccurredAt:       occurredAt,
		CreatedAt:        time.Now(),
	}
	return tx.Create(event).Error
}

func GenerateAllergyOrderNo() string {
	return fmt.Sprintf("AO%s%s", time.Now().Format("20060102150405"), strings.ToUpper(common.GetRandomString(4)))
}

func GenerateAllergyPaymentTradeNo(orderID int64) string {
	return fmt.Sprintf("AO_PAY_%d_%d_%s", orderID, time.Now().Unix(), strings.ToUpper(common.GetRandomString(4)))
}

func CreateAllergyOrder(userID int, serviceCode string, serviceName string, priceCents int, recipientName string, recipientPhone string, recipientEmail string, shippingAddressJSON string) (*AllergyOrder, error) {
	order := &AllergyOrder{
		OrderNo:             GenerateAllergyOrderNo(),
		UserID:              userID,
		ServiceCode:         serviceCode,
		ServiceNameSnapshot: serviceName,
		ServicePriceCents:   priceCents,
		Currency:            "CNY",
		PaymentStatus:       AllergyPaymentStatusPending,
		OrderStatus:         AllergyOrderStatusPendingPayment,
		RecipientName:       strings.TrimSpace(recipientName),
		RecipientPhone:      strings.TrimSpace(recipientPhone),
		RecipientEmail:      NormalizeEmail(recipientEmail),
		ShippingAddressJSON: shippingAddressJSON,
	}
	err := DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(order).Error; err != nil {
			return err
		}
		return createOrderTimelineEventTx(tx, order.ID, "order_created", "订单已创建", "请完成支付以开始检测流程", true, userID, "", time.Now())
	})
	if err != nil {
		return nil, err
	}
	return order, nil
}

func GetAllergyOrderByID(orderID int64) (*AllergyOrder, error) {
	var order AllergyOrder
	err := DB.First(&order, orderID).Error
	if err != nil {
		return nil, err
	}
	return &order, nil
}

func GetAllergyOrderForUser(userID int, orderID int64) (*AllergyOrder, error) {
	var order AllergyOrder
	err := DB.Where("id = ? AND user_id = ?", orderID, userID).First(&order).Error
	if err != nil {
		return nil, err
	}
	return &order, nil
}

func ListAllergyOrdersByUser(userID int) ([]*AllergyOrder, error) {
	var orders []*AllergyOrder
	err := DB.Where("user_id = ?", userID).Order("id desc").Find(&orders).Error
	return orders, err
}

func ListAdminAllergyOrders(filter AllergyOrderFilter, pageInfo *common.PageInfo) ([]*AllergyOrder, int64, error) {
	var orders []*AllergyOrder
	var total int64
	query := DB.Model(&AllergyOrder{})
	if filter.OrderNo != "" {
		query = query.Where("order_no LIKE ?", "%"+strings.TrimSpace(filter.OrderNo)+"%")
	}
	if filter.Email != "" {
		query = query.Where("recipient_email LIKE ?", "%"+NormalizeEmail(filter.Email)+"%")
	}
	if filter.PaymentStatus != "" {
		query = query.Where("payment_status = ?", strings.TrimSpace(filter.PaymentStatus))
	}
	if filter.OrderStatus != "" {
		query = query.Where("order_status = ?", strings.TrimSpace(filter.OrderStatus))
	}
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := query.Order("id desc").Limit(pageInfo.GetPageSize()).Offset(pageInfo.GetStartIdx()).Find(&orders).Error; err != nil {
		return nil, 0, err
	}
	return orders, total, nil
}

func SetAllergyOrderPaymentRequest(orderID int64, userID int, paymentMethod string, tradeNo string) (*AllergyOrder, error) {
	order, err := GetAllergyOrderForUser(userID, orderID)
	if err != nil {
		return nil, err
	}
	if order.PaymentStatus != AllergyPaymentStatusPending {
		return nil, errors.New("订单支付状态不可再次拉起支付")
	}
	updates := map[string]any{
		"payment_method": paymentMethod,
		"payment_ref":    tradeNo,
		"updated_at":     time.Now(),
	}
	if err := DB.Model(order).Updates(updates).Error; err != nil {
		return nil, err
	}
	order.PaymentMethod = paymentMethod
	order.PaymentRef = tradeNo
	return order, nil
}

func CompleteAllergyOrderPayment(tradeNo string, providerOrderNo string, payloadJSON string) (*AllergyOrder, error) {
	var order AllergyOrder
	err := DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("payment_ref = ?", strings.TrimSpace(tradeNo)).First(&order).Error; err != nil {
			return err
		}
		if order.PaymentStatus == AllergyPaymentStatusPaid {
			return nil
		}
		now := time.Now()
		if err := tx.Model(&order).Updates(map[string]any{
			"payment_status":                AllergyPaymentStatusPaid,
			"order_status":                  AllergyOrderStatusPaid,
			"paid_at":                       &now,
			"payment_provider_order_no":     providerOrderNo,
			"payment_callback_payload_json": payloadJSON,
			"updated_at":                    now,
		}).Error; err != nil {
			return err
		}
		return createOrderTimelineEventTx(tx, order.ID, "payment_completed", "订单已支付", "我们已开始准备采样盒", true, 0, payloadJSON, now)
	})
	if err != nil {
		return nil, err
	}
	return &order, nil
}

func GetAllergyOrderTimelineForUser(userID int, orderID int64) ([]*OrderTimelineEvent, error) {
	if _, err := GetAllergyOrderForUser(userID, orderID); err != nil {
		return nil, err
	}
	var events []*OrderTimelineEvent
	err := DB.Where("order_id = ? AND visible_to_user = ?", orderID, true).Order("occurred_at asc, id asc").Find(&events).Error
	return events, err
}

func GetAllergyOrderTimeline(orderID int64) ([]*OrderTimelineEvent, error) {
	var events []*OrderTimelineEvent
	err := DB.Where("order_id = ?", orderID).Order("occurred_at asc, id asc").Find(&events).Error
	return events, err
}

func GetSampleKitByOrderID(orderID int64) (*SampleKit, error) {
	var kit SampleKit
	err := DB.Where("order_id = ?", orderID).First(&kit).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &kit, nil
}

func GetLabSubmissionByOrderID(orderID int64) (*LabSubmission, error) {
	var submission LabSubmission
	err := DB.Where("order_id = ?", orderID).First(&submission).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &submission, nil
}

func UpsertSampleKitForOrder(orderID int64, kitCode string, kitStatus string, carrier string, trackingNo string, shippedAt *time.Time, operatorUserID int) (*SampleKit, error) {
	now := time.Now()
	var saved SampleKit
	err := DB.Transaction(func(tx *gorm.DB) error {
		var order AllergyOrder
		if err := tx.First(&order, orderID).Error; err != nil {
			return err
		}
		if order.PaymentStatus != AllergyPaymentStatusPaid {
			return errors.New("订单未支付，不能进入履约流程")
		}

		err := tx.Where("order_id = ?", orderID).First(&saved).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			saved = SampleKit{OrderID: orderID}
			err = nil
		}
		if err != nil {
			return err
		}
		if kitCode != "" {
			saved.KitCode = strings.TrimSpace(kitCode)
		}
		saved.Status = strings.TrimSpace(kitStatus)
		saved.TrackingCompany = strings.TrimSpace(carrier)
		saved.TrackingNumber = strings.TrimSpace(trackingNo)
		saved.ShippedAt = shippedAt

		if saved.ID == 0 {
			if err := tx.Create(&saved).Error; err != nil {
				return err
			}
		} else {
			if err := tx.Model(&saved).Updates(map[string]any{
				"kit_code":         saved.KitCode,
				"status":           saved.Status,
				"tracking_company": saved.TrackingCompany,
				"tracking_number":  saved.TrackingNumber,
				"shipped_at":       saved.ShippedAt,
				"updated_at":       now,
			}).Error; err != nil {
				return err
			}
		}

		orderStatus := order.OrderStatus
		eventType := ""
		eventTitle := ""
		eventDesc := ""
		switch saved.Status {
		case AllergyKitStatusPrepared:
			orderStatus = AllergyOrderStatusKitPreparing
			eventType = AllergyOrderStatusKitPreparing
			eventTitle = "采样盒准备中"
			eventDesc = "我们正在准备采样盒"
		case AllergyKitStatusShipped:
			orderStatus = AllergyOrderStatusKitShipped
			eventType = "kit_shipped"
			eventTitle = "采样盒已寄出"
			eventDesc = strings.TrimSpace(saved.TrackingCompany + " " + saved.TrackingNumber)
		case AllergyKitStatusDelivered:
			orderStatus = AllergyOrderStatusKitShipped
		}
		if err := tx.Model(&order).Updates(map[string]any{
			"order_status": orderStatus,
			"updated_at":   now,
		}).Error; err != nil {
			return err
		}
		if eventType != "" {
			return createOrderTimelineEventTx(tx, orderID, eventType, eventTitle, eventDesc, true, operatorUserID, "", now)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &saved, nil
}

func MarkAllergySampleSentBack(orderID int64, sentBackAt time.Time, returnTrackingNo string, remark string, operatorUserID int) error {
	now := time.Now()
	returnTrackingNo = strings.TrimSpace(returnTrackingNo)
	remark = strings.TrimSpace(remark)

	return DB.Transaction(func(tx *gorm.DB) error {
		var order AllergyOrder
		if err := tx.First(&order, orderID).Error; err != nil {
			return err
		}
		if order.PaymentStatus != AllergyPaymentStatusPaid {
			return errors.New("订单未支付，不能进入履约流程")
		}
		if order.OrderStatus != AllergyOrderStatusKitShipped && order.OrderStatus != AllergyOrderStatusSampleReturning {
			return errors.New("订单当前状态不能标记为样本回寄")
		}

		var kit SampleKit
		if err := tx.Where("order_id = ?", orderID).First(&kit).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New("采样盒尚未发货")
			}
			return err
		}
		kit.Status = AllergyKitStatusSampleSentBack
		kit.ReturnTrackingNo = returnTrackingNo
		kit.Remark = remark
		if err := tx.Model(&kit).Updates(map[string]any{
			"status":             kit.Status,
			"return_tracking_no": kit.ReturnTrackingNo,
			"remark":             kit.Remark,
			"updated_at":         now,
		}).Error; err != nil {
			return err
		}

		var submission LabSubmission
		err := tx.Where("order_id = ?", orderID).First(&submission).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			submission = LabSubmission{
				OrderID:     orderID,
				SampleKitID: kit.ID,
			}
			err = nil
		}
		if err != nil {
			return err
		}
		submission.SampleKitID = kit.ID
		submission.Status = "returning"
		submission.SubmittedAt = &sentBackAt
		submission.TrackingNumber = returnTrackingNo
		submission.Remark = remark
		if submission.ID == 0 {
			if err := tx.Create(&submission).Error; err != nil {
				return err
			}
		} else {
			if err := tx.Model(&submission).Updates(map[string]any{
				"sample_kit_id":   submission.SampleKitID,
				"status":          submission.Status,
				"submitted_at":    submission.SubmittedAt,
				"tracking_number": submission.TrackingNumber,
				"remark":          submission.Remark,
				"updated_at":      now,
			}).Error; err != nil {
				return err
			}
		}

		if err := tx.Model(&order).Updates(map[string]any{
			"order_status": AllergyOrderStatusSampleReturning,
			"admin_remark": remark,
			"updated_at":   now,
		}).Error; err != nil {
			return err
		}

		eventDesc := "用户已寄回样本，等待实验室签收"
		if returnTrackingNo != "" {
			eventDesc = returnTrackingNo
		} else if remark != "" {
			eventDesc = remark
		}
		payloadJSON := ""
		if returnTrackingNo != "" {
			payloadJSON = common.GetJsonString(map[string]any{
				"return_tracking_no": returnTrackingNo,
			})
		}
		return createOrderTimelineEventTx(tx, orderID, "sample_sent_back", "样本回寄中", eventDesc, true, operatorUserID, payloadJSON, sentBackAt)
	})
}

func MarkAllergySampleReceived(orderID int64, receivedAt time.Time, remark string, operatorUserID int) error {
	now := time.Now()
	return DB.Transaction(func(tx *gorm.DB) error {
		var order AllergyOrder
		if err := tx.First(&order, orderID).Error; err != nil {
			return err
		}

		var kit SampleKit
		err := tx.Where("order_id = ?", orderID).First(&kit).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			kit = SampleKit{OrderID: orderID}
			err = nil
		}
		if err != nil {
			return err
		}
		kit.Status = AllergyKitStatusSampleReceived
		kit.SampleReceivedAt = &receivedAt
		kit.Remark = strings.TrimSpace(remark)
		if kit.ID == 0 {
			if err := tx.Create(&kit).Error; err != nil {
				return err
			}
		} else {
			if err := tx.Model(&kit).Updates(map[string]any{
				"status":             kit.Status,
				"sample_received_at": kit.SampleReceivedAt,
				"remark":             kit.Remark,
				"updated_at":         now,
			}).Error; err != nil {
				return err
			}
		}

		var submission LabSubmission
		err = tx.Where("order_id = ?", orderID).First(&submission).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			submission = LabSubmission{OrderID: orderID}
			err = nil
		}
		if err != nil {
			return err
		}
		submission.SampleKitID = kit.ID
		submission.Status = "received"
		submission.ReceivedAt = &receivedAt
		submission.Remark = strings.TrimSpace(remark)
		if submission.ID == 0 {
			if err := tx.Create(&submission).Error; err != nil {
				return err
			}
		} else {
			if err := tx.Model(&submission).Updates(map[string]any{
				"sample_kit_id": submission.SampleKitID,
				"status":        submission.Status,
				"received_at":   submission.ReceivedAt,
				"remark":        submission.Remark,
				"updated_at":    now,
			}).Error; err != nil {
				return err
			}
		}

		if err := tx.Model(&order).Updates(map[string]any{
			"order_status": AllergyOrderStatusSampleReceived,
			"updated_at":   now,
		}).Error; err != nil {
			return err
		}

		return createOrderTimelineEventTx(tx, orderID, "sample_received", "样本已签收", "检测机构已收到样本", true, operatorUserID, "", receivedAt)
	})
}

func StartAllergyOrderTesting(orderID int64, startedAt time.Time, remark string, operatorUserID int) error {
	now := time.Now()
	remark = strings.TrimSpace(remark)

	return DB.Transaction(func(tx *gorm.DB) error {
		var order AllergyOrder
		if err := tx.First(&order, orderID).Error; err != nil {
			return err
		}
		if order.OrderStatus != AllergyOrderStatusSampleReceived && order.OrderStatus != AllergyOrderStatusInTesting {
			return errors.New("订单当前状态不能开始检测")
		}

		kit, err := GetSampleKitByOrderID(orderID)
		if err != nil {
			return err
		}

		var submission LabSubmission
		err = tx.Where("order_id = ?", orderID).First(&submission).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			submission = LabSubmission{
				OrderID: orderID,
			}
			err = nil
		}
		if err != nil {
			return err
		}
		if kit != nil {
			submission.SampleKitID = kit.ID
		}
		submission.Status = "testing"
		submission.TestingStartedAt = &startedAt
		submission.Remark = remark
		if submission.ID == 0 {
			if err := tx.Create(&submission).Error; err != nil {
				return err
			}
		} else {
			if err := tx.Model(&submission).Updates(map[string]any{
				"sample_kit_id":      submission.SampleKitID,
				"status":             submission.Status,
				"testing_started_at": submission.TestingStartedAt,
				"remark":             submission.Remark,
				"updated_at":         now,
			}).Error; err != nil {
				return err
			}
		}

		if err := tx.Model(&order).Updates(map[string]any{
			"order_status": AllergyOrderStatusInTesting,
			"admin_remark": remark,
			"updated_at":   now,
		}).Error; err != nil {
			return err
		}

		eventDesc := "检测机构正在分析样本"
		if remark != "" {
			eventDesc = remark
		}
		return createOrderTimelineEventTx(tx, orderID, "in_testing", "检测中", eventDesc, true, operatorUserID, "", startedAt)
	})
}

func CreateAllergyLabReport(orderID int64, reportTitle string, fileName string, filePath string, mimeType string, fileSize int64, operatorUserID int) (*LabReport, error) {
	now := time.Now()
	report := &LabReport{}
	err := DB.Transaction(func(tx *gorm.DB) error {
		var order AllergyOrder
		if err := tx.First(&order, orderID).Error; err != nil {
			return err
		}
		var maxVersion int
		if err := tx.Model(&LabReport{}).Where("order_id = ?", orderID).Select("COALESCE(MAX(version), 0)").Scan(&maxVersion).Error; err != nil {
			return err
		}
		kit, err := GetSampleKitByOrderID(orderID)
		if err != nil {
			return err
		}
		var submission LabSubmission
		_ = tx.Where("order_id = ?", orderID).First(&submission).Error

		report.OrderID = orderID
		if kit != nil {
			report.SampleKitID = kit.ID
		}
		report.LabSubmissionID = submission.ID
		report.Version = maxVersion + 1
		report.Status = AllergyReportStatusUploaded
		report.IsCurrent = false
		report.ReportTitle = strings.TrimSpace(reportTitle)
		report.PDFStorageType = "local"
		report.FileName = fileName
		report.FilePath = filePath
		report.MimeType = mimeType
		report.FileSizeBytes = fileSize
		report.UploadedAt = &now
		report.CreatedAt = now
		report.UpdatedAt = now
		if err := tx.Create(report).Error; err != nil {
			return err
		}
		return createOrderTimelineEventTx(tx, orderID, "report_uploaded", "检测报告已上传", "报告已上传，等待发布", false, operatorUserID, "", now)
	})
	if err != nil {
		return nil, err
	}
	return report, nil
}

func GetLabReportByID(reportID int64) (*LabReport, error) {
	var report LabReport
	if err := DB.First(&report, reportID).Error; err != nil {
		return nil, err
	}
	return &report, nil
}

func GetCurrentAllergyReportForOrder(orderID int64) (*LabReport, error) {
	var report LabReport
	err := DB.Where("order_id = ? AND is_current = ? AND status = ?", orderID, true, AllergyReportStatusPublished).
		Order("id desc").
		First(&report).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &report, nil
}

func ListAllergyReportsForOrder(orderID int64) ([]*LabReport, error) {
	var reports []*LabReport
	err := DB.Where("order_id = ?", orderID).Order("version desc, id desc").Find(&reports).Error
	return reports, err
}

func PublishAllergyLabReport(reportID int64, operatorUserID int) (*LabReport, error) {
	var report LabReport
	err := DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.First(&report, reportID).Error; err != nil {
			return err
		}
		now := time.Now()
		if err := tx.Model(&LabReport{}).Where("order_id = ?", report.OrderID).Updates(map[string]any{
			"is_current": false,
			"updated_at": now,
		}).Error; err != nil {
			return err
		}
		if err := tx.Model(&report).Updates(map[string]any{
			"status":       AllergyReportStatusPublished,
			"is_current":   true,
			"published_at": &now,
			"updated_at":   now,
		}).Error; err != nil {
			return err
		}
		if err := tx.Model(&AllergyOrder{}).Where("id = ?", report.OrderID).Updates(map[string]any{
			"order_status":    AllergyOrderStatusReportReady,
			"report_ready_at": &now,
			"updated_at":      now,
		}).Error; err != nil {
			return err
		}
		return createOrderTimelineEventTx(tx, report.OrderID, "report_published", "检测报告已发布", "您现在可以在线预览和下载报告", true, operatorUserID, "", now)
	})
	if err != nil {
		return nil, err
	}
	return &report, nil
}

func GetAllergyReportForUser(userID int, reportID int64) (*LabReport, *AllergyOrder, error) {
	report, err := GetLabReportByID(reportID)
	if err != nil {
		return nil, nil, err
	}
	order, err := GetAllergyOrderForUser(userID, report.OrderID)
	if err != nil {
		return nil, nil, err
	}
	if report.Status != AllergyReportStatusPublished {
		return nil, nil, errors.New("报告尚未发布")
	}
	return report, order, nil
}

func CreateReportDeliveryLog(reportID int64, orderID int64, recipientEmail string, deliveryType string, status string, operatorUserID int, errorMessage string) (*ReportDeliveryLog, error) {
	now := time.Now()
	logItem := &ReportDeliveryLog{
		ReportID:          reportID,
		OrderID:           orderID,
		RecipientEmail:    NormalizeEmail(recipientEmail),
		DeliveryChannel:   "email",
		DeliveryType:      deliveryType,
		DeliveryStatus:    status,
		TriggeredByUserID: operatorUserID,
		ErrorMessage:      errorMessage,
		CreatedAt:         now,
	}
	if status == "sent" {
		logItem.SentAt = &now
	}
	if err := DB.Create(logItem).Error; err != nil {
		return nil, err
	}
	if status == "sent" {
		if err := DB.Transaction(func(tx *gorm.DB) error {
			if err := tx.Model(&LabReport{}).Where("id = ?", reportID).Updates(map[string]any{
				"last_email_sent_at":         &now,
				"email_sent_count":           gorm.Expr("email_sent_count + 1"),
				"last_sent_by_admin_user_id": operatorUserID,
				"updated_at":                 now,
			}).Error; err != nil {
				return err
			}
			return createOrderTimelineEventTx(tx, orderID, "report_email_sent", "检测报告已发送", "我们已将报告发送到您的邮箱", false, operatorUserID, "", now)
		}); err != nil {
			return nil, err
		}
	}
	return logItem, nil
}

func ListReportDeliveryLogs(reportID int64) ([]*ReportDeliveryLog, error) {
	var logs []*ReportDeliveryLog
	err := DB.Where("report_id = ?", reportID).Order("id desc").Find(&logs).Error
	return logs, err
}

func CompleteAllergyOrder(orderID int64, completedAt time.Time, remark string, operatorUserID int) error {
	now := time.Now()
	remark = strings.TrimSpace(remark)

	return DB.Transaction(func(tx *gorm.DB) error {
		var order AllergyOrder
		if err := tx.First(&order, orderID).Error; err != nil {
			return err
		}
		if order.OrderStatus != AllergyOrderStatusReportReady && order.OrderStatus != AllergyOrderStatusCompleted {
			return errors.New("订单当前状态不能标记为完成")
		}

		if err := tx.Model(&order).Updates(map[string]any{
			"order_status": AllergyOrderStatusCompleted,
			"completed_at": &completedAt,
			"admin_remark": remark,
			"updated_at":   now,
		}).Error; err != nil {
			return err
		}

		var submission LabSubmission
		err := tx.Where("order_id = ?", orderID).First(&submission).Error
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return err
		}
		if err == nil {
			if err := tx.Model(&submission).Updates(map[string]any{
				"status":       "completed",
				"completed_at": &completedAt,
				"remark":       remark,
				"updated_at":   now,
			}).Error; err != nil {
				return err
			}
		}

		eventDesc := "订单已完成"
		if remark != "" {
			eventDesc = remark
		}
		return createOrderTimelineEventTx(tx, orderID, "completed", "订单已完成", eventDesc, true, operatorUserID, "", completedAt)
	})
}
