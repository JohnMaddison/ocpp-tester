package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/johnmaddison/ocpp-go"
	"github.com/johnmaddison/ocpp-go/ocpp16"
	"github.com/johnmaddison/ocpp-go/server"
)

var ErrSessionNotFound = errors.New("ocpp session not found")
var ErrUnsupportedAction = errors.New("unsupported ocpp 1.6 action")
var ErrInvalidPayload = errors.New("invalid ocpp 1.6 payload")

type OCPP16Service struct {
	ocppServer *server.Server
}

type SendCallResult struct {
	ChargePointID string          `json:"chargePointId"`
	Action        string          `json:"action"`
	Payload       any             `json:"payload,omitempty"`
	CallError     *ocpp.CallError `json:"callError,omitempty"`
}

func NewOCPP16Service(ocppServer *server.Server) *OCPP16Service {
	return &OCPP16Service{ocppServer: ocppServer}
}

func SupportedOCPP16ServerCallActions() []string {
	actions := make([]string, 0, len(ocpp16ServerCallPayloads))
	for action := range ocpp16ServerCallPayloads {
		actions = append(actions, action)
	}
	slices.Sort(actions)
	return actions
}

func (s *OCPP16Service) SendCall(ctx context.Context, chargePointID string, action string, payloadJSON []byte) (*SendCallResult, error) {
	payload, err := buildOCPP16CallPayload(action, payloadJSON)
	if err != nil {
		return nil, err
	}

	session, ok := s.ocppServer.Session(chargePointID)
	if !ok {
		return nil, ErrSessionNotFound
	}

	callCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	result, err := session.Send16CallWithContext(callCtx, ocpp16.Action(action), payload)
	if err != nil {
		return nil, err
	}

	response := &SendCallResult{ChargePointID: chargePointID, Action: action}
	if result.IsCallError() {
		response.CallError = result.CallError
		return response, nil
	}
	if result.IsCallResult() {
		response.Payload = result.CallResult.Payload
	}
	return response, nil
}

func buildOCPP16CallPayload(action string, payloadJSON []byte) (any, error) {
	factory, ok := ocpp16ServerCallPayloads[action]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrUnsupportedAction, action)
	}

	payload := factory()
	trimmed := strings.TrimSpace(string(payloadJSON))
	if trimmed == "" || trimmed == "null" {
		trimmed = "{}"
	}

	if err := json.Unmarshal([]byte(trimmed), payload); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidPayload, err)
	}
	if err := validateOCPP16Payload(reflect.ValueOf(payload), "payload"); err != nil {
		return nil, err
	}
	return reflect.ValueOf(payload).Elem().Interface(), nil
}

var ocpp16ServerCallPayloads = map[string]func() any{
	string(ocpp16.ActionChangeAvailability):     func() any { return &ocpp16.ChangeAvailabilityRequest{} },
	string(ocpp16.ActionChangeConfiguration):    func() any { return &ocpp16.ChangeConfigurationRequest{} },
	string(ocpp16.ActionClearCache):             func() any { return &ocpp16.ClearCacheRequest{} },
	string(ocpp16.ActionDataTransfer):           func() any { return &ocpp16.DataTransferRequest{} },
	string(ocpp16.ActionGetConfiguration):       func() any { return &ocpp16.GetConfigurationRequest{} },
	string(ocpp16.ActionRemoteStartTransaction): func() any { return &ocpp16.RemoteStartTransactionRequest{} },
	string(ocpp16.ActionRemoteStopTransaction):  func() any { return &ocpp16.RemoteStopTransactionRequest{} },
	string(ocpp16.ActionReset):                  func() any { return &ocpp16.ResetRequest{} },
	string(ocpp16.ActionUnlockConnector):        func() any { return &ocpp16.UnlockConnectorRequest{} },
	string(ocpp16.ActionGetLocalListVersion):    func() any { return &ocpp16.GetLocalListVersionRequest{} },
	string(ocpp16.ActionSendLocalList):          func() any { return &ocpp16.SendLocalListRequest{} },
	string(ocpp16.ActionCancelReservation):      func() any { return &ocpp16.CancelReservationRequest{} },
	string(ocpp16.ActionReserveNow):             func() any { return &ocpp16.ReserveNowRequest{} },
	string(ocpp16.ActionClearChargingProfile):   func() any { return &ocpp16.ClearChargingProfileRequest{} },
	string(ocpp16.ActionGetCompositeSchedule):   func() any { return &ocpp16.GetCompositeScheduleRequest{} },
	string(ocpp16.ActionSetChargingProfile):     func() any { return &ocpp16.SetChargingProfileRequest{} },
	string(ocpp16.ActionGetDiagnostics):         func() any { return &ocpp16.GetDiagnosticsRequest{} },
	string(ocpp16.ActionUpdateFirmware):         func() any { return &ocpp16.UpdateFirmwareRequest{} },
	string(ocpp16.ActionTriggerMessage):         func() any { return &ocpp16.TriggerMessageRequest{} },
}

func validateOCPP16Payload(value reflect.Value, path string) error {
	if !value.IsValid() {
		return nil
	}
	if value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return nil
		}
		value = value.Elem()
	}

	switch value.Kind() {
	case reflect.Struct:
		if value.Type() == reflect.TypeOf(time.Time{}) {
			return nil
		}
		for i := 0; i < value.NumField(); i++ {
			field := value.Type().Field(i)
			if field.PkgPath != "" {
				continue
			}
			fieldPath := path + "." + jsonFieldName(field)
			fieldValue := value.Field(i)
			if err := validateTags(fieldValue, field.Tag.Get("validate"), fieldPath); err != nil {
				return err
			}
			if err := validateOCPP16Payload(fieldValue, fieldPath); err != nil {
				return err
			}
		}
	case reflect.Slice, reflect.Array:
		for i := 0; i < value.Len(); i++ {
			if err := validateOCPP16Payload(value.Index(i), fmt.Sprintf("%s[%d]", path, i)); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateTags(value reflect.Value, tag string, path string) error {
	if tag == "" || tag == "-" {
		return nil
	}
	if value.Kind() == reflect.Pointer && value.IsNil() && !strings.Contains(tag, "required") {
		return nil
	}
	for _, rule := range strings.Split(tag, ",") {
		name, arg, _ := strings.Cut(rule, "=")
		switch name {
		case "required":
			if isZero(value) {
				return fmt.Errorf("%w: %s is required", ErrInvalidPayload, path)
			}
		case "min":
			limit, err := strconv.ParseFloat(arg, 64)
			if err == nil && numericOrLength(value) < limit {
				return fmt.Errorf("%w: %s must be at least %s", ErrInvalidPayload, path, arg)
			}
		case "max":
			limit, err := strconv.ParseFloat(arg, 64)
			if err == nil && numericOrLength(value) > limit {
				return fmt.Errorf("%w: %s must be at most %s", ErrInvalidPayload, path, arg)
			}
		case "gt":
			limit, err := strconv.ParseFloat(arg, 64)
			if err == nil && numericOrLength(value) <= limit {
				return fmt.Errorf("%w: %s must be greater than %s", ErrInvalidPayload, path, arg)
			}
		case "gte":
			limit, err := strconv.ParseFloat(arg, 64)
			if err == nil && numericOrLength(value) < limit {
				return fmt.Errorf("%w: %s must be greater than or equal to %s", ErrInvalidPayload, path, arg)
			}
		case "oneof":
			if !valueIsOneOf(value, strings.Fields(arg)) {
				return fmt.Errorf("%w: %s must be one of %s", ErrInvalidPayload, path, arg)
			}
		}
	}
	return nil
}

func isZero(value reflect.Value) bool {
	if value.Kind() == reflect.Pointer {
		return value.IsNil() || value.Elem().IsZero()
	}
	return value.IsZero()
}

func numericOrLength(value reflect.Value) float64 {
	if value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return 0
		}
		value = value.Elem()
	}
	switch value.Kind() {
	case reflect.String, reflect.Slice, reflect.Array, reflect.Map:
		return float64(value.Len())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return float64(value.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return float64(value.Uint())
	case reflect.Float32, reflect.Float64:
		return value.Float()
	default:
		return 0
	}
}

func valueIsOneOf(value reflect.Value, allowed []string) bool {
	if value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return true
		}
		value = value.Elem()
	}
	if value.Kind() != reflect.String {
		return true
	}
	actual := value.String()
	for _, item := range allowed {
		if actual == item {
			return true
		}
	}
	return false
}

func jsonFieldName(field reflect.StructField) string {
	name := strings.Split(field.Tag.Get("json"), ",")[0]
	if name == "" || name == "-" {
		return field.Name
	}
	return name
}
