package auth

import (
	"context"
	"fmt"
	"html"
	"net"
	"net/http"
	"strings"
)

type CallbackResult struct {
	Code  string
	Error string
}

const callbackPageTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width,initial-scale=1">
<title>Terra3 Login</title>
<style>
  *,*::before,*::after{box-sizing:border-box;margin:0;padding:0}
  body{
    font-family:"Helvetica Neue",Arial,sans-serif;
    background:#edece9;
    color:#444742;
    min-height:100vh;
    display:flex;
    flex-direction:column;
    align-items:center;
    justify-content:center;
    padding:24px;
  }
  .card{
    background:#fff;
    width:100%;
    max-width:460px;
    border-top:7px solid {{ACCENT}};
    padding:40px 40px 36px;
  }

  .status-row{
    display:flex;
    align-items:center;
    gap:14px;
    margin-bottom:16px;
  }
  .status-icon{
    flex-shrink:0;
    width:36px;
    height:36px;
    border-radius:50%;
    background:{{ICON_BG}};
    display:flex;
    align-items:center;
    justify-content:center;
  }
  .status-icon svg{stroke:#fff;fill:none;stroke-width:2.5;stroke-linecap:round;stroke-linejoin:round}
  h1{
    font-size:20px;
    font-weight:700;
    letter-spacing:0.03em;
    color:#444742;
    margin:0;
    white-space:nowrap;
  }
  p{
    font-size:14px;
    line-height:1.55;
    color:#6e6f6a;
    margin-bottom:0;
  }
  .error-detail{
    margin-top:16px;
    background:#f8f8f8;
    border-left:3px solid #ed6845;
    padding:10px 14px;
    font-size:13px;
    font-family:Menlo,monospace;
    color:#6e6f6a;
    word-break:break-all;
  }
  .footer{
    margin-top:40px;
    font-size:12px;
    color:#8a8b86;
    letter-spacing:0.04em;
    text-align:center;
  }
</style>
</head>
<body>
<div class="card">
  <div class="status-row">
    <div class="status-icon">{{ICON_SVG}}</div>
    <h1>{{TITLE}}</h1>
  </div>
  <p>{{BODY}}</p>
  {{EXTRA}}
</div>
<div class="footer">Terra3 Platform</div>
</body>
</html>`

func callbackPage(accent, iconBg, iconSVG, title, body, extra string) string {
	r := strings.NewReplacer(
		"{{ACCENT}}", accent,
		"{{ICON_BG}}", iconBg,
		"{{ICON_SVG}}", iconSVG,
		"{{TITLE}}", title,
		"{{BODY}}", body,
		"{{EXTRA}}", extra,
	)
	return r.Replace(callbackPageTemplate)
}

var iconCheck = `<svg width="20" height="20" viewBox="0 0 24 24"><polyline points="20 6 9 17 4 12"/></svg>`
var iconX = `<svg width="20" height="20" viewBox="0 0 24 24"><line x1="18" y1="6" x2="6" y2="18"/><line x1="6" y1="6" x2="18" y2="18"/></svg>`

// StartCallbackServer starts a local HTTP server and returns the port and a channel that receives the auth code.
func StartCallbackServer() (int, <-chan CallbackResult, func()) {
	ch := make(chan CallbackResult, 1)

	listener, err := net.Listen("tcp", "127.0.0.1:9876")
	if err != nil {
		ch <- CallbackResult{Error: fmt.Sprintf("failed to start listener: %v", err)}
		return 0, ch, func() {}
	}

	port := listener.Addr().(*net.TCPAddr).Port

	mux := http.NewServeMux()
	srv := &http.Server{Handler: mux}

	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		errParam := r.URL.Query().Get("error")

		w.Header().Set("Content-Type", "text/html; charset=utf-8")

		if errParam != "" {
			desc := r.URL.Query().Get("error_description")
			extra := ""
			if desc != "" {
				extra = fmt.Sprintf(`<div class="error-detail">%s</div>`, html.EscapeString(desc))
			}
			fmt.Fprint(w, callbackPage(
				"#ed6845", "#ed6845", iconX,
				"Login failed",
				"Something went wrong during authentication. You can close this window and try again.",
				extra,
			))
			ch <- CallbackResult{Error: fmt.Sprintf("%s: %s", errParam, desc)}
			return
		}

		if code == "" {
			fmt.Fprint(w, callbackPage(
				"#ed6845", "#ed6845", iconX,
				"Login failed",
				"No authorization code was received. You can close this window and try again.",
				"",
			))
			ch <- CallbackResult{Error: "no authorization code received"}
			return
		}

		fmt.Fprint(w, callbackPage(
			"#129a6a", "#129a6a", iconCheck,
			"Logged in",
			"Authentication successful. You can close this window and return to the terminal.",
			"",
		))
		ch <- CallbackResult{Code: code}
	})

	go srv.Serve(listener)

	shutdown := func() {
		srv.Shutdown(context.Background())
	}

	return port, ch, shutdown
}
