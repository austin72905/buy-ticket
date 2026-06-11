package service

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strings"
)

var ErrInvalidPaymentSignature = errors.New("invalid payment signature")

type MockPaymentSignatureConfig struct {
	MerchantID string
	HashKey    string
	HashIV     string
}

type VerifyMockPaymentCallbackInput struct {
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

func (s *BookingService) VerifyMockPaymentCallback(input VerifyMockPaymentCallbackInput) error {
	if s.MockPaymentSignature.HashKey == "" || s.MockPaymentSignature.HashIV == "" {
		return nil
	}

	if strings.TrimSpace(input.CheckMacValue) == "" {
		return ErrInvalidPaymentSignature
	}
	if s.MockPaymentSignature.MerchantID != "" && input.MerchantID != s.MockPaymentSignature.MerchantID {
		return ErrInvalidPaymentSignature
	}

	expected := buildMockPaymentCheckMacValue(input, s.MockPaymentSignature)
	if !strings.EqualFold(expected, input.CheckMacValue) {
		return ErrInvalidPaymentSignature
	}

	return nil
}

func buildMockPaymentCheckMacValue(input VerifyMockPaymentCallbackInput, cfg MockPaymentSignatureConfig) string {
	params := map[string]string{
		"MerchantID":      input.MerchantID,
		"MerchantTradeNo": input.MerchantTradeNo,
		"RtnCode":         input.RtnCode,
		"RtnMsg":          input.RtnMsg,
		"TradeNo":         input.TradeNo,
		"TradeAmt":        input.TradeAmt,
		"PaymentDate":     input.PaymentDate,
		"PaymentType":     input.PaymentType,
		"TradeDate":       input.TradeDate,
		"SimulatePaid":    input.SimulatePaid,
		"ReturnStatus":    input.ReturnStatus,
	}

	if input.PaymentTypeChargeFee != "" {
		params["PaymentTypeChargeFee"] = input.PaymentTypeChargeFee
	}
	if input.CustomField1 != "" {
		params["CustomField1"] = input.CustomField1
	}
	if input.CustomField2 != "" {
		params["CustomField2"] = input.CustomField2
	}
	if input.CustomField3 != "" {
		params["CustomField3"] = input.CustomField3
	}
	if input.CustomField4 != "" {
		params["CustomField4"] = input.CustomField4
	}

	keys := make([]string, 0, len(params))
	for key := range params {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	parts := make([]string, 0, len(keys)+2)
	parts = append(parts, fmt.Sprintf("HashKey=%s", cfg.HashKey))
	for _, key := range keys {
		parts = append(parts, fmt.Sprintf("%s=%s", key, params[key]))
	}
	parts = append(parts, fmt.Sprintf("HashIV=%s", cfg.HashIV))

	encoded := url.QueryEscape(strings.Join(parts, "&"))
	encoded = strings.ReplaceAll(encoded, "%20", "+")
	encoded = strings.ToLower(encoded)

	hash := sha256.Sum256([]byte(encoded))
	return strings.ToUpper(hex.EncodeToString(hash[:]))
}
