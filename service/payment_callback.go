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
	MerchantTradeNo string
	RtnCode         string
	TradeNo         string
	TradeAmt        string
	PaymentDate     string
	PaymentType     string
}

func (s *BookingService) HandleECPayCallback(ctx context.Context, input HandleECPayCallbackInput) error {
	if input.MerchantTradeNo == "" || input.RtnCode == "" {
		return ErrInvalidPaymentCallback
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
