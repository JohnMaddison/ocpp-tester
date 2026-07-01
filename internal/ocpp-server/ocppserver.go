package ocppserver

import (
	"context"
	"log"
	"time"

	"github.com/JohnMaddison/ocpp-go"
	"github.com/JohnMaddison/ocpp-go/server"
	"github.com/JohnMaddison/ocpp-tester/internal/ocppclients"
)

func StartOCPPServer(context.Context) {

	opts := []server.Option{
		server.WithPath("ocpp"),
		//server.WithOCPP16AuthorizeHandler(authorizeHandler),
		//server.WithOCPP16BootNotificationHandler(bootNotificationHandler),
		//server.WithOCPP16HeartbeatHandler(heartbeatHandler),
		//server.WithOCPP16MeterValuesHandler(meterValuesHandler),
		//server.WithOCPP16StartTransactionHandler(startTransactionHandler),
		//server.WithOCPP16StatusNotificationHandler(statusNotificationHandler),
		//server.WithOCPP16StopTransactionHandler(stopTransactionHandler),
		//server.WithConnectRequestHandler(ConnectHandler),
		server.WithConnectedHandler(connectedHandler),
		server.WithDisconnectHandler(disconnectHandler),
		server.WithTrafficLogging(),
		//server.WithKeepaliveLogging(),
		server.WithWebsocketKeepalive(10*time.Second, 5*time.Second),
	}

	log.Print("Listening for OCPP traffic at 0.0.0.0:8081/ocpp/{chargepointid}")

	err := server.NewServer("0.0.0.0:8081", opts...).Serve()

	if err != nil {
		log.Fatalf("Failed to start server %s", err)
	}

}

func connectedHandler(info ocpp.ConnectionInfo) {
	ocppclients.UpsertClient(info.ChargePointID)
}

func disconnectHandler(info ocpp.ConnectionInfo) {

}
