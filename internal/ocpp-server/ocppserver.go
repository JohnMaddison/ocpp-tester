package ocppserver

import (
	"context"
	"log"
	"net"
	"net/http"
	"time"

	"github.com/johnmaddison/ocpp-go"
	"github.com/johnmaddison/ocpp-go/ocpp16"
	"github.com/johnmaddison/ocpp-go/server"
	"github.com/johnmaddison/ocpp-tester/internal/http-server/data"
)

type OCPPMessageRecorder interface {
	RecordOCPPMessage(ctx context.Context, message data.OCPPLogMessage) error
}

func NewOCPPServer(sessionStore *data.SessionStore, recorder OCPPMessageRecorder) *server.Server {
	var ocppServer *server.Server

	opts := []server.Option{
		server.WithPath("ocpp"),
		server.With16AuthorizeHandler(authorizeHandler),
		server.With16BootNotificationHandler(bootNotificationHandler),
		server.With16HeartbeatHandler(heartbeatHandler),
		server.With16MeterValuesHandler(meterValuesHandler),
		server.With16StartTransactionHandler(startTransactionHandler),
		server.With16StatusNotificationHandler(statusNotificationHandler),
		server.With16StopTransactionHandler(stopTransactionHandler),
		server.WithConnectedHandler(func(info ocpp.ConnectionInfo) {
			connectedHandler(sessionStore, ocppServer, info)
		}),
		server.WithDisconnectHandler(func(info ocpp.ConnectionInfo) {
			disconnectHandler(sessionStore, info)
		}),
		server.WithTrafficLogging(),
		//server.WithKeepaliveLogging(),
		server.WithWebsocketKeepalive(10*time.Second, 20*time.Second),
		server.WithHTTPTimeouts(time.Second*60, time.Second*60, time.Second*60),
		server.WithMessageReceivedHandler(func(info ocpp.MessageInfo) {
			recordOCPPMessage(recorder, info)
		}),
		server.WithMessageSentHandler(func(info ocpp.MessageInfo) {
			recordOCPPMessage(recorder, info)
		}),
	}

	ocppServer = server.NewServer("0.0.0.0:8081", opts...)
	return ocppServer
}

func recordOCPPMessage(recorder OCPPMessageRecorder, info ocpp.MessageInfo) {
	if recorder == nil {
		return
	}

	if err := recorder.RecordOCPPMessage(context.Background(), data.OCPPLogMessage{
		ChargePointID: info.ConnectionInfo.ChargePointID,
		Protocol:      info.Protocol,
		Direction:     data.OCPPMessageDirection(info.Direction),
		Message:       string(info.Message),
	}); err != nil {
		log.Printf("Failed to record OCPP %s message for charge point %q: %v", info.Direction, info.ConnectionInfo.ChargePointID, err)
	}
}

func StartOCPPServer(ctx context.Context, ocppServer *server.Server) {
	log.Print("Listening for OCPP traffic at ws://0.0.0.0:8081/ocpp/{chargepointid}")

	go func() {
		<-ctx.Done()

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		log.Print("Stopping OCPP server")
		if err := ocppServer.Shutdown(shutdownCtx); err != nil {
			log.Printf("Failed to shutdown OCPP server gracefully: %v", err)
		}
	}()

	err := ocppServer.Serve()

	if err != nil && err != http.ErrServerClosed {
		log.Fatalf("Failed to start server %s", err)
	}

}

func connectedHandler(store *data.SessionStore, ocppServer *server.Server, info ocpp.ConnectionInfo) {
	protocol := ""
	if ocppServer != nil {
		if session, ok := ocppServer.Session(info.ChargePointID); ok {
			protocol = session.Protocol()
		}
	}

	store.Upsert(data.Session{
		ChargePointID: info.ChargePointID,
		Protocol:      protocol,
		RemoteAddr:    addrString(info.RemoteAddr),
		LocalAddr:     addrString(info.LocalAddr),
		ConnectedAt:   time.Now().UTC(),
	})
}

func disconnectHandler(store *data.SessionStore, info ocpp.ConnectionInfo) {
	store.Delete(info.ChargePointID)
}

func addrString(addr net.Addr) string {
	if addr == nil {
		return ""
	}
	return addr.String()
}

func bootNotificationHandler(ctx *ocpp16.Context, request ocpp16.BootNotificationRequest) (*ocpp16.BootNotificationResponse, *ocpp16.Error) {

	return &ocpp16.BootNotificationResponse{
		CurrentTime: time.Now(),
		Status:      "Accepted",
		Interval:    10,
	}, nil
}

func heartbeatHandler(ctx *ocpp16.Context, request ocpp16.HeartbeatRequest) (*ocpp16.HeartbeatResponse, *ocpp16.Error) {
	return &ocpp16.HeartbeatResponse{
		CurrentTime: time.Now().UTC(),
	}, nil
}

func statusNotificationHandler(ctx *ocpp16.Context, request ocpp16.StatusNotificationRequest) (*ocpp16.StatusNotificationResponse, *ocpp16.Error) {
	return &ocpp16.StatusNotificationResponse{}, nil
}

func authorizeHandler(ctx *ocpp16.Context, request ocpp16.AuthorizeRequest) (*ocpp16.AuthorizeResponse, *ocpp16.Error) {
	return &ocpp16.AuthorizeResponse{IDTagInfo: ocpp16.IDTagInfo{Status: ocpp16.AuthorizationStatusAccepted}}, nil
}

func startTransactionHandler(ctx *ocpp16.Context, request ocpp16.StartTransactionRequest) (*ocpp16.StartTransactionResponse, *ocpp16.Error) {
	return &ocpp16.StartTransactionResponse{IDTagInfo: ocpp16.IDTagInfo{Status: ocpp16.AuthorizationStatusAccepted}, TransactionID: 10000}, nil
}

func meterValuesHandler(ctx *ocpp16.Context, request ocpp16.MeterValuesRequest) (*ocpp16.MeterValuesResponse, *ocpp16.Error) {
	return &ocpp16.MeterValuesResponse{}, nil
}

func stopTransactionHandler(ctx *ocpp16.Context, request ocpp16.StopTransactionRequest) (*ocpp16.StopTransactionResponse, *ocpp16.Error) {
	return &ocpp16.StopTransactionResponse{IDTagInfo: &ocpp16.IDTagInfo{Status: ocpp16.AuthorizationStatusAccepted}}, nil
}
