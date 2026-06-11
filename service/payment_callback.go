package service

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"

	"buy-ticket/domain"
)

var ErrInvalidPaymentCallback = errors.New("invalid payment callback")

type HandleECPayCallbackInput struct {
	MerchantID           string
	MerchantTradeNo      string
	RtnCode              string
	RtnMsg               string
	TradeNo              string
	TradeAmt             string
	PaymentDate          string
	PaymentType          string
	PaymentTypeChargeFee string
	TradeDate            string
	SimulatePaid         string
	CustomField1         string
	CustomField2         string
	CustomField3         string
	CustomField4         string
	CheckMacValue        string
	ReturnStatus         string
}

func (s *BookingService) HandleECPayCallback(ctx context.Context, input HandleECPayCallbackInput) error {
	if input.MerchantTradeNo == "" || input.RtnCode == "" {
		return ErrInvalidPaymentCallback
	}

	if err := s.VerifyMockPaymentCallback(VerifyMockPaymentCallbackInput{
		MerchantID:           input.MerchantID,
		MerchantTradeNo:      input.MerchantTradeNo,
		RtnCode:              input.RtnCode,
		RtnMsg:               input.RtnMsg,
		TradeNo:              input.TradeNo,
		TradeAmt:             input.TradeAmt,
		PaymentDate:          input.PaymentDate,
		PaymentType:          input.PaymentType,
		PaymentTypeChargeFee: input.PaymentTypeChargeFee,
		TradeDate:            input.TradeDate,
		SimulatePaid:         input.SimulatePaid,
		CustomField1:         input.CustomField1,
		CustomField2:         input.CustomField2,
		CustomField3:         input.CustomField3,
		CustomField4:         input.CustomField4,
		CheckMacValue:        input.CheckMacValue,
		ReturnStatus:         input.ReturnStatus,
	}); err != nil {
		return err
	}

	if input.RtnCode != "1" {
		return nil
	}

	if input.TradeNo == "" || input.TradeAmt == "" {
		return ErrInvalidPaymentCallback
	}

	if existingPayment, err := s.PaymentRepo.FindByPaymentNo(ctx, input.TradeNo); err == nil && existingPayment != nil {
		return nil
	}

	order, err := s.OrderRepo.FindByOrderNo(ctx, input.MerchantTradeNo)
	if err != nil {
		return err
	}

	if order.Status == domain.OrderStatusPaid {
		return nil
	}

	amount, err := strconv.ParseInt(input.TradeAmt, 10, 64)
	if err != nil {
		return ErrInvalidPaymentCallback
	}

	paidAt, err := parseECPayPaymentTime(input.PaymentDate)
	if err != nil {
		return err
	}

	_, err = s.PayOrder(ctx, PayOrderInput{
		OrderID:   order.ID,
		PaymentNo: input.TradeNo,
		Method:    normalizeECPayPaymentMethod(input.PaymentType),
		Amount:    amount,
		PaidAt:    paidAt,
	})
	return err
}

func parseECPayPaymentTime(value string) (time.Time, error) {
	if strings.TrimSpace(value) == "" {
		return time.Now(), nil
	}

	paidAt, err := time.ParseInLocation("2006/01/02 15:04:05", value, time.Local)
	if err != nil {
		return time.Time{}, ErrInvalidPaymentCallback
	}

	return paidAt, nil
}

func normalizeECPayPaymentMethod(value string) string {
	method := strings.TrimSpace(strings.ToLower(value))
	method = strings.ReplaceAll(method, " ", "_")
	if method == "" {
		return "ecpay"
	}
	return "ecpay_" + method
}
