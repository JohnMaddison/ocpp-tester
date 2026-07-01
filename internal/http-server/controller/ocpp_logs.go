package controller

import (
	"context"
	"encoding/json"
	"html/template"
	"net/http"
	"strconv"

	"github.com/johnmaddison/ocpp-tester/internal/http-server/data"
	"github.com/johnmaddison/ocpp-tester/internal/http-server/service"
)

type OCPPLogController struct {
	service interface {
		ListMessages(ctx context.Context, chargePointID string, limit int) ([]data.OCPPLogMessage, error)
	}
}

type ocppLogResponse struct {
	ChargePointID string                `json:"chargePointId"`
	Messages      []data.OCPPLogMessage `json:"messages"`
}

func NewOCPPLogController(ocppLogService *service.OCPPLogService) *OCPPLogController {
	return &OCPPLogController{service: ocppLogService}
}

func (c *OCPPLogController) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/chargepoints/{chargePointID}/ocpp-log", c.handleAPIOCPPLog)
	mux.HandleFunc("GET /chargepoints/{chargePointID}/ocpp-log", c.handleOCPPLogView)
}

func (c *OCPPLogController) handleAPIOCPPLog(w http.ResponseWriter, r *http.Request) {
	chargePointID := r.PathValue("chargePointID")
	limit, ok := parseOCPPLogLimit(w, r)
	if !ok {
		return
	}

	messages, err := c.service.ListMessages(r.Context(), chargePointID, limit)
	if err != nil {
		http.Error(w, "failed to list ocpp log", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(ocppLogResponse{ChargePointID: chargePointID, Messages: messages}); err != nil {
		http.Error(w, "failed to encode ocpp log", http.StatusInternalServerError)
	}
}

func (c *OCPPLogController) handleOCPPLogView(w http.ResponseWriter, r *http.Request) {
	chargePointID := r.PathValue("chargePointID")
	limit, ok := parseOCPPLogLimit(w, r)
	if !ok {
		return
	}

	messages, err := c.service.ListMessages(r.Context(), chargePointID, limit)
	if err != nil {
		http.Error(w, "failed to list ocpp log", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := ocppLogTemplate.Execute(w, ocppLogResponse{ChargePointID: chargePointID, Messages: messages}); err != nil {
		http.Error(w, "failed to render ocpp log", http.StatusInternalServerError)
	}
}

func parseOCPPLogLimit(w http.ResponseWriter, r *http.Request) (int, bool) {
	rawLimit := r.URL.Query().Get("limit")
	if rawLimit == "" {
		return service.DefaultOCPPLogLimit, true
	}

	limit, err := strconv.Atoi(rawLimit)
	if err != nil || limit < 1 || limit > service.MaxOCPPLogLimit {
		http.Error(w, "limit must be an integer between 1 and 1000", http.StatusBadRequest)
		return 0, false
	}

	return limit, true
}

func ocppLogAction(message data.OCPPLogMessage) string {
	if message.Action != nil {
		return *message.Action
	}
	if message.MessageType != nil {
		return *message.MessageType
	}
	return ""
}

var ocppLogTemplate = template.Must(template.New("ocpp-log").Funcs(template.FuncMap{
	"ocppLogAction": ocppLogAction,
}).Parse(`<!doctype html>
<html lang="en">
<head>
	<meta charset="utf-8">
	<title>OCPP Log {{.ChargePointID}}</title>
</head>
<body>
	<h1>OCPP Log {{.ChargePointID}}</h1>
	<table>
		<thead>
			<tr>
				<th>Timestamp</th>
				<th>Direction</th>
				<th>Protocol</th>
				<th>Action/Type</th>
				<th>Unique ID</th>
				<th>Raw Message</th>
			</tr>
		</thead>
		<tbody>
			{{range .Messages}}
			<tr>
				<td>{{.CreatedAt.Format "2006-01-02T15:04:05Z07:00"}}</td>
				<td>{{.Direction}}</td>
				<td>{{.Protocol}}</td>
				<td>{{ocppLogAction .}}</td>
				<td>{{if .UniqueID}}{{.UniqueID}}{{end}}</td>
				<td><pre>{{.Message}}</pre></td>
			</tr>
			{{else}}
			<tr>
				<td colspan="6">No OCPP log messages</td>
			</tr>
			{{end}}
		</tbody>
	</table>
</body>
</html>
`))
