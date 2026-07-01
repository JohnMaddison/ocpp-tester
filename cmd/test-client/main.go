package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/johnmaddison/ocpp-go"
	"github.com/johnmaddison/ocpp-go/client"
	"github.com/johnmaddison/ocpp-go/ocpp16"
)

var profiler = flag.Bool("profile", false, "enable profiler")

var model = "Simulator"
var vendor = "OCPP-GO"

func main() {
	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Print("Received interrupt signal, shutting down gracefully...")
		cancel()
	}()

	if *profiler {
		go func() {
			log.Println("Starting pprof server on :6060")
			log.Println(http.ListenAndServe("localhost:6060", nil))
		}()
	}

	chargePointID := "CP_000001"
	cp := client.New16(chargePointID, "ws://127.0.0.1:8081/ocpp").
		With16GetConfigurationHandler(getConfigurationHandler).
		WithConnectedHandler(ConnectedHandler).
		WithDisconnectHandler(DisconnectHandler).
		WithWebsocketKeepalive(5*time.Second, 45*time.Second).
		WithKeepaliveLogging().
		WithTrafficLogging()

	for {
		err := cp.Connect()
		if err != nil {
			log.Printf("Failed to connect to server %v", err)
			time.Sleep(10 * time.Second)
		} else {
			break
		}
	}

	bootNotificationResponseInterval := 10

	for {
		request, err := cp.Send16Call(ocpp16.ActionBootNotification, ocpp16.BootNotificationRequest{ChargePointModel: model, ChargePointVendor: vendor})

		if err != nil {
			log.Printf("Failed to send bootnotification %s", err)
		}

		if request.IsCallError() {
			log.Printf("Received callerror: %s %+v\n", chargePointID, request.CallError)
			time.Sleep(time.Duration(10) * time.Second)
		} else if request.IsCallResult() {

			bootNotificationResponse, _ := request.GetPayload().(ocpp16.BootNotificationResponse)

			if bootNotificationResponse.Status == ocpp16.RegistrationStatusAccepted {
				log.Printf("Received accepted boot response")
				bootNotificationResponseInterval = bootNotificationResponse.Interval
				break
			} else {
				log.Printf("Received non accepted boot response")
				time.Sleep(time.Duration(bootNotificationResponse.Interval) * time.Second)
			}

		}
	}

	cp.Send16Call(ocpp16.ActionStatusNotification, ocpp16.StatusNotificationRequest{ConnectorID: 0, ErrorCode: ocpp16.ChargePointErrorCodeNoError, Status: ocpp16.ChargePointStatusAvailable})
	cp.Send16Call(ocpp16.ActionStatusNotification, ocpp16.StatusNotificationRequest{ConnectorID: 1, ErrorCode: ocpp16.ChargePointErrorCodeNoError, Status: ocpp16.ChargePointStatusAvailable})
	cp.Send16Call(ocpp16.ActionStatusNotification, ocpp16.StatusNotificationRequest{ConnectorID: 2, ErrorCode: ocpp16.ChargePointErrorCodeNoError, Status: ocpp16.ChargePointStatusAvailable})

	chargingConnectorID := 1
	cp.Send16Call(ocpp16.ActionStatusNotification, ocpp16.StatusNotificationRequest{ConnectorID: chargingConnectorID, ErrorCode: ocpp16.ChargePointErrorCodeNoError, Status: ocpp16.ChargePointStatusPreparing})

	tagIdentifier := "0000-0000-0001"
	authorizeRequest, err := cp.Send16Call(ocpp16.ActionAuthorize, ocpp16.AuthorizeRequest{IDTag: tagIdentifier})

	if err != nil {
		log.Printf("Failed to send authorize request %s", err)
		return
	}

	if authorizeRequest.IsCallError() {
		log.Printf("Received callerror: %s %+v\n", chargePointID, authorizeRequest.CallError)
	} else if authorizeRequest.IsCallResult() {

		authorizeResponse, _ := authorizeRequest.GetPayload().(ocpp16.AuthorizeResponse)

		if authorizeResponse.IDTagInfo.Status == ocpp16.AuthorizationStatusAccepted {
			meterStart := 0
			startTransactionRequest, err := cp.Send16Call(ocpp16.ActionStartTransaction, ocpp16.StartTransactionRequest{ConnectorID: chargingConnectorID, IDTag: tagIdentifier, MeterStart: meterStart, Timestamp: time.Now()})

			if err != nil {
				log.Printf("Failed to send startTransaction request %s", err)
				return
			}

			if startTransactionRequest.IsCallResult() {

				startTransactionResponse, _ := startTransactionRequest.GetPayload().(ocpp16.StartTransactionResponse)

				if startTransactionResponse.IDTagInfo.Status == ocpp16.AuthorizationStatusAccepted {

					cp.Send16Call(ocpp16.ActionStatusNotification, ocpp16.StatusNotificationRequest{ConnectorID: chargingConnectorID, ErrorCode: ocpp16.ChargePointErrorCodeNoError, Status: ocpp16.ChargePointStatusCharging})

					meterValuesTicker := time.NewTicker(5 * time.Second)
					defer meterValuesTicker.Stop()

				Outer:
					for {
						select {
						case <-meterValuesTicker.C:
							cp.Send16Call(ocpp16.ActionMeterValues, ocpp16.MeterValuesRequest{
								ConnectorID:   chargingConnectorID,
								TransactionID: &startTransactionResponse.TransactionID,
								MeterValue: []ocpp16.MeterValue{
									{
										Timestamp: time.Now(),
										SampledValue: []ocpp16.SampledValue{
											{
												Value:     strconv.Itoa(meterStart),
												Context:   ocpp.Ptr(ocpp16.ReadingContextSamplePeriodic),
												Measurand: ocpp.Ptr(ocpp16.MeasurandEnergyActiveImportRegister),
												Unit:      ocpp.Ptr("Wh"),
											},
										},
									},
								},
							})

							meterStart = meterStart + 10

							if meterStart > 10 {
								break Outer
							}

						case <-ctx.Done():
							log.Println("Stopping MeterValues loop")
							break Outer
						}
					}
				}

				_, err := cp.Send16Call(ocpp16.ActionStopTransaction, ocpp16.StopTransactionRequest{IDTag: &tagIdentifier, MeterStop: meterStart, Timestamp: time.Now(), TransactionID: startTransactionResponse.TransactionID, Reason: ocpp.Ptr(ocpp16.ReasonEVDisconnected)})

				if err != nil {
					log.Printf("Failed to send stopTransaction request %s", err)
					return
				}

				cp.Send16Call(ocpp16.ActionStatusNotification, ocpp16.StatusNotificationRequest{ConnectorID: chargingConnectorID, ErrorCode: ocpp16.ChargePointErrorCodeNoError, Status: ocpp16.ChargePointStatusFinishing})
				cp.Send16Call(ocpp16.ActionStatusNotification, ocpp16.StatusNotificationRequest{ConnectorID: chargingConnectorID, ErrorCode: ocpp16.ChargePointErrorCodeNoError, Status: ocpp16.ChargePointStatusAvailable})
			}

		}

	}

	heartbeatTicker := time.NewTicker(time.Duration(bootNotificationResponseInterval) * time.Second)
	defer heartbeatTicker.Stop()

	for {
		select {
		case <-heartbeatTicker.C:
			cp.Send16Call(ocpp16.ActionHeartbeat, ocpp16.EmptyPayload{})

		case <-ctx.Done():
			log.Print("Shutdown complete")
			return
		}
	}

}

func ConnectedHandler(info ocpp.ConnectionInfo) {
	log.Printf("Connected: %s", info.ChargePointID)
}

func DisconnectHandler(info ocpp.ConnectionInfo) {
	log.Printf("Disconnected: %s", info.ChargePointID)
}

func getConfigurationHandler(ctx *ocpp16.Context, request ocpp16.GetConfigurationRequest) (*ocpp16.GetConfigurationResponse, *ocpp16.Error) {
	log.Printf("GetConfiguration request received from server: %s %+v\n", ctx.ChargePointID, request)

	conf := []ocpp16.KeyValue{
		{
			Key:      "ChargePointVendor",
			Value:    &vendor,
			Readonly: true,
		},
		{
			Key:      "ChargePointModel",
			Value:    &model,
			Readonly: true,
		},
	}
	return &ocpp16.GetConfigurationResponse{ConfigurationKey: conf}, nil
}
