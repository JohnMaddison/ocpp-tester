package controller

import (
	"encoding/json"
	"html/template"
	"net/http"
	"net/url"

	"github.com/johnmaddison/ocpp-tester/internal/http-server/data"
	"github.com/johnmaddison/ocpp-tester/internal/http-server/service"
)

type SessionsController struct {
	service *service.SessionsService
}

type sessionsResponse struct {
	Sessions []data.Session `json:"sessions"`
}

type sessionViewData struct {
	Session data.Session
	Actions []ocpp16CallControl
}

type ocpp16CallControl struct {
	Name        string
	PayloadJSON string
}

func NewSessionsController(service *service.SessionsService) *SessionsController {
	return &SessionsController{service: service}
}

func (c *SessionsController) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /", c.handleRootView)
	mux.HandleFunc("GET /api/sessions", c.handleAPISessions)
	mux.HandleFunc("GET /sessions", c.handleSessionsView)
	mux.HandleFunc("GET /sessions/{chargePointID}", c.handleSessionView)
}

func (c *SessionsController) handleAPISessions(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(sessionsResponse{Sessions: c.service.ListSessions()}); err != nil {
		http.Error(w, "failed to encode sessions", http.StatusInternalServerError)
	}
}

func (c *SessionsController) handleSessionsView(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := sessionsTemplate.Execute(w, sessionsResponse{Sessions: c.service.ListSessions()}); err != nil {
		http.Error(w, "failed to render sessions", http.StatusInternalServerError)
	}
}

func (c *SessionsController) handleRootView(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	c.handleSessionsView(w, r)
}

func (c *SessionsController) handleSessionView(w http.ResponseWriter, r *http.Request) {
	chargePointID := r.PathValue("chargePointID")
	session, ok := c.service.GetSession(chargePointID)
	if !ok {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := sessionTemplate.Execute(w, sessionViewData{
		Session: session,
		Actions: ocpp16CallControls(),
	}); err != nil {
		http.Error(w, "failed to render session", http.StatusInternalServerError)
	}
}

func pathEscape(value string) string {
	return url.PathEscape(value)
}

func ocpp16CallControls() []ocpp16CallControl {
	actions := service.SupportedOCPP16ServerCallActions()
	controls := make([]ocpp16CallControl, 0, len(actions))
	for _, action := range actions {
		controls = append(controls, ocpp16CallControl{
			Name:        action,
			PayloadJSON: defaultOCPP16Payload(action),
		})
	}
	return controls
}

func defaultOCPP16Payload(action string) string {
	switch action {
	case "CancelReservation":
		return `{"reservationId":1}`
	case "ChangeAvailability":
		return `{"connectorId":0,"type":"Operative"}`
	case "ChangeConfiguration":
		return `{"key":"HeartbeatInterval","value":"60"}`
	case "ClearChargingProfile":
		return `{}`
	case "DataTransfer":
		return `{"vendorId":"ocpp-tester","messageId":"test","data":"{}"}`
	case "GetCompositeSchedule":
		return `{"connectorId":1,"duration":300}`
	case "GetConfiguration":
		return `{"key":["HeartbeatInterval"]}`
	case "GetDiagnostics":
		return `{"location":"https://example.invalid/diagnostics"}`
	case "GetLocalListVersion":
		return `{}`
	case "RemoteStartTransaction":
		return `{"idTag":"test"}`
	case "RemoteStopTransaction":
		return `{"transactionId":1}`
	case "ReserveNow":
		return `{"connectorId":1,"expiryDate":"2026-07-16T12:00:00Z","idTag":"test","reservationId":1}`
	case "Reset":
		return `{"type":"Soft"}`
	case "SendLocalList":
		return `{"listVersion":1,"localAuthorizationList":[],"updateType":"Full"}`
	case "SetChargingProfile":
		return `{"connectorId":1,"csChargingProfiles":{"chargingProfileId":1,"stackLevel":0,"chargingProfilePurpose":"TxDefaultProfile","chargingProfileKind":"Absolute","chargingSchedule":{"chargingRateUnit":"A","chargingSchedulePeriod":[{"startPeriod":0,"limit":16}]}}}`
	case "TriggerMessage":
		return `{"requestedMessage":"StatusNotification"}`
	case "UnlockConnector":
		return `{"connectorId":1}`
	case "UpdateFirmware":
		return `{"location":"https://example.invalid/firmware.bin","retrieveDate":"2026-07-16T12:00:00Z"}`
	default:
		return `{}`
	}
}

var sessionsTemplate = template.Must(template.New("sessions").Funcs(template.FuncMap{
	"pathEscape": pathEscape,
}).Parse(`<!doctype html>
<html lang="en">
<head>
	<meta charset="utf-8">
	<meta name="viewport" content="width=device-width, initial-scale=1">
	<title>OCPP Sessions</title>
	<style>
		:root {
			color-scheme: light;
			--bg: #f6f7f8;
			--panel: #ffffff;
			--text: #202327;
			--muted: #626a73;
			--line: #d8dde3;
			--accent: #0f766e;
			--accent-dark: #115e59;
			--error: #b42318;
		}
		* { box-sizing: border-box; }
		body {
			margin: 0;
			background: var(--bg);
			color: var(--text);
			font: 14px/1.45 system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
		}
		main {
			width: min(1180px, calc(100% - 32px));
			margin: 24px auto;
		}
		header {
			display: flex;
			align-items: end;
			justify-content: space-between;
			gap: 16px;
			margin-bottom: 16px;
		}
		h1 {
			margin: 0;
			font-size: 22px;
			line-height: 1.2;
		}
		.status {
			color: var(--muted);
			font-size: 13px;
		}
		.table-wrap {
			overflow-x: auto;
			background: var(--panel);
			border: 1px solid var(--line);
			border-radius: 6px;
		}
		table {
			width: 100%;
			border-collapse: collapse;
			min-width: 760px;
		}
		th, td {
			padding: 9px 10px;
			border-bottom: 1px solid var(--line);
			text-align: left;
			vertical-align: top;
			white-space: nowrap;
		}
		th {
			background: #eef2f4;
			color: #343a40;
			font-size: 12px;
			font-weight: 700;
			text-transform: uppercase;
		}
		tbody tr:last-child td { border-bottom: 0; }
		a {
			color: var(--accent-dark);
			font-weight: 650;
			text-decoration: none;
		}
		a:hover { text-decoration: underline; }
		.empty {
			color: var(--muted);
			text-align: center;
		}
		@media (max-width: 720px) {
			main { width: calc(100% - 20px); margin: 12px auto; }
			header { display: block; }
			.status { margin-top: 6px; }
		}
	</style>
</head>
<body>
	<main>
		<header>
			<h1>Active OCPP Sessions</h1>
			<div class="status">{{len .Sessions}} connected</div>
		</header>
		<div class="table-wrap">
			<table>
				<thead>
					<tr>
						<th>Charge Point ID</th>
						<th>Protocol</th>
						<th>Remote Address</th>
						<th>Local Address</th>
						<th>Connected At</th>
					</tr>
				</thead>
				<tbody>
					{{range .Sessions}}
					<tr>
						<td><a href="/sessions/{{pathEscape .ChargePointID}}">{{.ChargePointID}}</a></td>
						<td>{{.Protocol}}</td>
						<td>{{.RemoteAddr}}</td>
						<td>{{.LocalAddr}}</td>
						<td>{{.ConnectedAt.Format "2006-01-02T15:04:05Z07:00"}}</td>
					</tr>
					{{else}}
					<tr>
						<td colspan="5" class="empty">No active sessions</td>
					</tr>
					{{end}}
				</tbody>
			</table>
		</div>
	</main>
</body>
</html>
`))

var sessionTemplate = template.Must(template.New("session").Funcs(template.FuncMap{
	"pathEscape": pathEscape,
}).Parse(`<!doctype html>
<html lang="en">
<head>
	<meta charset="utf-8">
	<meta name="viewport" content="width=device-width, initial-scale=1">
	<title>OCPP Session {{.Session.ChargePointID}}</title>
	<style>
		:root {
			color-scheme: light;
			--bg: #f6f7f8;
			--panel: #ffffff;
			--text: #202327;
			--muted: #626a73;
			--line: #d8dde3;
			--accent: #0f766e;
			--accent-dark: #115e59;
			--error: #b42318;
			--ok: #067647;
		}
		* { box-sizing: border-box; }
		body {
			margin: 0;
			background: var(--bg);
			color: var(--text);
			font: 14px/1.45 system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif;
		}
		main {
			width: min(1280px, calc(100% - 32px));
			margin: 18px auto 28px;
		}
		a { color: var(--accent-dark); font-weight: 650; text-decoration: none; }
		a:hover { text-decoration: underline; }
		.top {
			display: flex;
			align-items: end;
			justify-content: space-between;
			gap: 16px;
			margin-bottom: 14px;
		}
		.back { font-size: 13px; }
		h1 {
			margin: 4px 0 0;
			font-size: 22px;
			line-height: 1.2;
		}
		.meta {
			display: grid;
			grid-template-columns: repeat(4, minmax(0, 1fr));
			gap: 1px;
			background: var(--line);
			border: 1px solid var(--line);
			border-radius: 6px;
			overflow: hidden;
			margin-bottom: 14px;
		}
		.meta div {
			background: var(--panel);
			padding: 9px 10px;
			min-width: 0;
		}
		.meta dt {
			margin: 0 0 3px;
			color: var(--muted);
			font-size: 12px;
			font-weight: 700;
			text-transform: uppercase;
		}
		.meta dd {
			margin: 0;
			overflow-wrap: anywhere;
			font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
			font-size: 13px;
		}
		.layout {
			display: grid;
			grid-template-columns: minmax(320px, 420px) minmax(0, 1fr);
			gap: 14px;
			align-items: start;
		}
		.panel {
			background: var(--panel);
			border: 1px solid var(--line);
			border-radius: 6px;
			overflow: hidden;
		}
		.panel h2 {
			margin: 0;
			padding: 10px 12px;
			border-bottom: 1px solid var(--line);
			background: #eef2f4;
			font-size: 14px;
		}
		form {
			display: grid;
			gap: 10px;
			padding: 12px;
		}
		label {
			display: grid;
			gap: 5px;
			color: var(--muted);
			font-size: 12px;
			font-weight: 700;
			text-transform: uppercase;
		}
		select, textarea, button {
			font: inherit;
		}
		select, textarea {
			width: 100%;
			border: 1px solid #b9c1cb;
			border-radius: 5px;
			background: #ffffff;
			color: var(--text);
		}
		select {
			height: 36px;
			padding: 0 9px;
		}
		textarea {
			min-height: 220px;
			resize: vertical;
			padding: 9px;
			font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
			font-size: 13px;
			line-height: 1.45;
		}
		button {
			min-height: 36px;
			width: max-content;
			border: 1px solid var(--accent-dark);
			border-radius: 5px;
			background: var(--accent);
			color: #ffffff;
			font-weight: 700;
			padding: 0 12px;
			cursor: pointer;
		}
		button:disabled {
			cursor: wait;
			opacity: 0.72;
		}
		.result {
			margin: 0 12px 12px;
			border: 1px solid var(--line);
			border-radius: 5px;
			background: #f8fafb;
			min-height: 88px;
			overflow: auto;
		}
		.result[data-state="ok"] { border-color: #75b798; }
		.result[data-state="error"] { border-color: #f1aeb5; }
		pre {
			margin: 0;
			white-space: pre-wrap;
			overflow-wrap: anywhere;
			font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
			font-size: 12px;
			line-height: 1.45;
		}
		.result pre { padding: 10px; }
		.log-head {
			display: flex;
			align-items: center;
			justify-content: space-between;
			gap: 12px;
			padding: 10px 12px;
			border-bottom: 1px solid var(--line);
			background: #eef2f4;
		}
		.log-head h2 {
			padding: 0;
			border: 0;
			background: transparent;
		}
		.log-status {
			color: var(--muted);
			font-size: 12px;
			white-space: nowrap;
		}
		.table-wrap { overflow-x: auto; }
		.log-table {
			width: 100%;
			border-collapse: collapse;
			table-layout: fixed;
			min-width: 0;
		}
		.log-time { width: 128px; }
		.log-dir { width: 44px; }
		.log-protocol { width: 70px; }
		.log-action { width: 112px; }
		.log-uid { width: 120px; }
		.log-message { width: auto; }
		.log-table th, .log-table td {
			padding: 6px 7px;
			border-bottom: 1px solid var(--line);
			text-align: left;
			vertical-align: top;
		}
		.log-table th {
			background: #f8fafb;
			color: #343a40;
			font-size: 11px;
			font-weight: 700;
			text-transform: uppercase;
			white-space: nowrap;
		}
		.log-table td {
			font-size: 12px;
			overflow-wrap: anywhere;
		}
		.log-raw pre {
			font-size: 11px;
			line-height: 1.35;
			white-space: pre-wrap;
			overflow-wrap: anywhere;
		}
		.empty {
			color: var(--muted);
			text-align: center;
		}
		@media (max-width: 900px) {
			main { width: calc(100% - 20px); margin: 12px auto; }
			.top { display: block; }
			.meta { grid-template-columns: 1fr 1fr; }
			.layout { grid-template-columns: 1fr; }
		}
		@media (max-width: 560px) {
			.meta { grid-template-columns: 1fr; }
		}
	</style>
</head>
<body>
	<main data-charge-point-id="{{.Session.ChargePointID}}">
		<div class="top">
			<div>
				<a class="back" href="/sessions">Sessions</a>
				<h1>{{.Session.ChargePointID}}</h1>
			</div>
			<a href="/chargepoints/{{pathEscape .Session.ChargePointID}}/ocpp-log">OCPP log page</a>
		</div>

		<dl class="meta">
			<div>
				<dt>Protocol</dt>
				<dd>{{.Session.Protocol}}</dd>
			</div>
			<div>
				<dt>Remote Address</dt>
				<dd>{{.Session.RemoteAddr}}</dd>
			</div>
			<div>
				<dt>Local Address</dt>
				<dd>{{.Session.LocalAddr}}</dd>
			</div>
			<div>
				<dt>Connected At</dt>
				<dd>{{.Session.ConnectedAt.Format "2006-01-02T15:04:05Z07:00"}}</dd>
			</div>
		</dl>

		<div class="layout">
			<section class="panel" aria-labelledby="call-title">
				<h2 id="call-title">Send OCPP 1.6 Call</h2>
				<form id="ocpp-call-form">
					<label>
						Action
						<select id="ocpp-action" name="action">
							{{range .Actions}}
							<option value="{{.Name}}" data-payload="{{.PayloadJSON}}">{{.Name}}</option>
							{{end}}
						</select>
					</label>
					<label>
						Request JSON
						<textarea id="ocpp-payload" name="payload" spellcheck="false"></textarea>
					</label>
					<button type="submit">Send Call</button>
				</form>
				<div id="call-result" class="result" data-state="">
					<pre>Select an action, edit the JSON payload, and submit the call.</pre>
				</div>
			</section>

			<section class="panel" aria-labelledby="log-title">
				<div class="log-head">
					<h2 id="log-title">OCPP Log</h2>
					<div id="log-status" class="log-status">Loading</div>
				</div>
				<div class="table-wrap">
					<table class="log-table">
						<colgroup>
							<col class="log-time">
							<col class="log-dir">
							<col class="log-protocol">
							<col class="log-action">
							<col class="log-uid">
							<col class="log-message">
						</colgroup>
						<thead>
							<tr>
								<th>Time</th>
								<th>Dir</th>
								<th>Proto</th>
								<th>Action</th>
								<th>UID</th>
								<th>Message</th>
							</tr>
						</thead>
						<tbody id="log-body">
							<tr><td colspan="6" class="empty">Loading OCPP log</td></tr>
						</tbody>
					</table>
				</div>
			</section>
		</div>
	</main>
	<script>
	(function () {
		var main = document.querySelector("main[data-charge-point-id]");
		var chargePointID = main.getAttribute("data-charge-point-id");
		var encodedID = encodeURIComponent(chargePointID);
		var actionSelect = document.getElementById("ocpp-action");
		var payloadInput = document.getElementById("ocpp-payload");
		var form = document.getElementById("ocpp-call-form");
		var result = document.getElementById("call-result");
		var resultPre = result.querySelector("pre");
		var logStatus = document.getElementById("log-status");
		var logBody = document.getElementById("log-body");

		function formatPayload() {
			var selected = actionSelect.options[actionSelect.selectedIndex];
			payloadInput.value = selected ? selected.getAttribute("data-payload") || "{}" : "{}";
		}

		function setResult(state, value) {
			result.setAttribute("data-state", state);
			resultPre.textContent = value;
		}

		function formatJSON(value) {
			try {
				return JSON.stringify(JSON.parse(value), null, 2);
			} catch (err) {
				return value;
			}
		}

		function pad2(value) {
			return String(value).padStart(2, "0");
		}

		function formatLogTime(value) {
			var date = new Date(value || "");
			if (Number.isNaN(date.getTime())) {
				return value || "";
			}

			var now = new Date();
			var time = pad2(date.getHours()) + ":" + pad2(date.getMinutes()) + ":" + pad2(date.getSeconds());
			if (
				date.getFullYear() === now.getFullYear() &&
				date.getMonth() === now.getMonth() &&
				date.getDate() === now.getDate()
			) {
				return time;
			}

			return date.getFullYear() + "-" + pad2(date.getMonth() + 1) + "-" + pad2(date.getDate()) + " " + time;
		}

		function formatDirection(value) {
			if (value === "received") {
				return "IN";
			}
			if (value === "sent") {
				return "OUT";
			}
			return value || "";
		}

		function cell(text, className) {
			var td = document.createElement("td");
			if (className) {
				td.className = className;
			}
			td.textContent = text || "";
			return td;
		}

		function preCell(text) {
			var td = document.createElement("td");
			td.className = "log-raw";
			var pre = document.createElement("pre");
			pre.textContent = formatJSON(text || "");
			td.appendChild(pre);
			return td;
		}

		function renderLog(messages) {
			logBody.textContent = "";
			if (!messages || messages.length === 0) {
				var emptyRow = document.createElement("tr");
				var emptyCell = cell("No OCPP log messages", "empty");
				emptyCell.colSpan = 6;
				emptyRow.appendChild(emptyCell);
				logBody.appendChild(emptyRow);
				return;
			}
			messages.forEach(function (message) {
				var row = document.createElement("tr");
				row.appendChild(cell(formatLogTime(message.createdAt), "log-time-cell"));
				row.appendChild(cell(formatDirection(message.direction), "log-dir-cell"));
				row.appendChild(cell(message.protocol === "{{.Session.Protocol}}" ? "" : message.protocol || "", "log-protocol-cell"));
				row.appendChild(cell(message.action || message.messageType || "", "log-action-cell"));
				row.appendChild(cell(message.uniqueId || "", "log-uid-cell"));
				row.appendChild(preCell(message.message || ""));
				logBody.appendChild(row);
			});
		}

		function refreshLog() {
			fetch("/api/chargepoints/" + encodedID + "/ocpp-log?limit=100", {
				headers: { "Accept": "application/json" }
			}).then(function (response) {
				if (!response.ok) {
					throw new Error("HTTP " + response.status);
				}
				return response.json();
			}).then(function (body) {
				renderLog(body.messages || []);
				logStatus.textContent = "Updated " + new Date().toLocaleTimeString();
			}).catch(function (err) {
				logStatus.textContent = "Refresh failed: " + err.message;
			});
		}

		actionSelect.addEventListener("change", formatPayload);
		form.addEventListener("submit", function (event) {
			event.preventDefault();
			var action = actionSelect.value;
			var payload = payloadInput.value.trim() || "{}";
			var button = form.querySelector("button");

			button.disabled = true;
			setResult("", "Sending " + action + "...");

			fetch("/api/ocpp16/" + encodedID + "/calls/" + encodeURIComponent(action), {
				method: "POST",
				headers: {
					"Accept": "application/json",
					"Content-Type": "application/json"
				},
				body: payload
			}).then(function (response) {
				return response.text().then(function (text) {
					return { ok: response.ok, status: response.status, text: text };
				});
			}).then(function (response) {
				var output = "HTTP " + response.status + "\n" + formatJSON(response.text);
				setResult(response.ok ? "ok" : "error", output);
				refreshLog();
			}).catch(function (err) {
				setResult("error", err.message);
			}).finally(function () {
				button.disabled = false;
			});
		});

		formatPayload();
		refreshLog();
		window.setInterval(refreshLog, 3000);
	})();
	</script>
</body>
</html>
`))
