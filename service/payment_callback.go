package service

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	"buy-ticket/domain"
	"buy-ticket/repository"
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
		if attempt, findErr := s.findPaymentAttemptForCallback(ctx, input.MerchantTradeNo); findErr == nil && attempt != nil {
			callbackPayload, _ := json.Marshal(input)
			if attempt.MarkFailed(input.RtnMsg, callbackPayload, time.Now()) {
				_ = s.PaymentAttemptRepo.Save(ctx, attempt)
			}
		}
		return nil
	}

	if input.TradeNo == "" || input.TradeAmt == "" {
		return ErrInvalidPaymentCallback
	}

	attempt, err := s.findPaymentAttemptForCallback(ctx, input.MerchantTradeNo)
	if err != nil {
		return err
	}

	if existingPayment, err := s.PaymentRepo.FindByPaymentNo(ctx, input.TradeNo); err == nil && existingPayment != nil {
		if attempt != nil && attempt.Status != domain.PaymentAttemptStatusSucceeded {
			callbackPayload, _ := json.Marshal(input)
			if attempt.MarkSucceeded(existingPayment.ID, input.TradeNo, callbackPayload, time.Now()) {
				_ = s.PaymentAttemptRepo.Save(ctx, attempt)
			}
		}
		return nil
	}

	var order *domain.Order
	if attempt != nil {
		order, err = s.OrderRepo.FindByID(ctx, attempt.OrderID)
	} else {
		order, err = s.OrderRepo.FindByOrderNo(ctx, input.MerchantTradeNo)
	}
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

	payment, err := s.PayOrder(ctx, PayOrderInput{
		OrderID:   order.ID,
		PaymentNo: input.TradeNo,
		Method:    normalizeECPayPaymentMethod(input.PaymentType),
		Amount:    amount,
		PaidAt:    paidAt,
	})
	if err != nil {
		return err
	}

	if attempt != nil {
		callbackPayload, _ := json.Marshal(input)
		if attempt.MarkSucceeded(payment.ID, input.TradeNo, callbackPayload, paidAt) {
			return s.PaymentAttemptRepo.Save(ctx, attempt)
		}
	}

	return err
}

func (s *BookingService) findPaymentAttemptForCallback(ctx context.Context, merchantTradeNo string) (*domain.PaymentAttempt, error) {
	if s.PaymentAttemptRepo == nil {
		return nil, nil
	}

	attempt, err := s.PaymentAttemptRepo.FindByMerchantTradeNo(ctx, merchantTradeNo)
	if err == nil {
		return attempt, nil
	}
	if errors.Is(err, repository.ErrPaymentAttemptNotFound) {
		return nil, nil
	}
	return nil, err
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
