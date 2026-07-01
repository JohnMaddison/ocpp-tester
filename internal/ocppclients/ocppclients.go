package ocppclients

type OcppClient struct {
	ChargePointID string
}

var clients map[string]OcppClient

func Init() {
	clients = make(map[string]OcppClient)
}

func GetClients() map[string]OcppClient {
	return clients
}

func UpsertClient(clientId string) {

	clients[clientId] = OcppClient{ChargePointID: clientId}
}
