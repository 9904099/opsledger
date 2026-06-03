package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"

	"github.com/9904099/opsledger/internal/model"
	"golang.org/x/net/websocket"
)

func (s *Server) handleWebSSHSessionPage(w http.ResponseWriter, r *http.Request) {
	user, ok := currentUser(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, errors.New("unauthorized"))
		return
	}
	sessionID := r.PathValue("session")
	assetID := r.URL.Query().Get("asset")
	if assetID == "" {
		sessionID = r.PathValue("session")
	}
	if _, err := s.store.ValidateWebSSHSession(r.Context(), user, sessionID, assetID); err != nil {
		writeStoreError(w, err)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	page := strings.NewReplacer(
		"__ASSET_HTML__", escapeHTMLForHTML(assetID),
		"__ASSET_JS__", jsonString(assetID),
		"__SESSION_JS__", jsonString(sessionID),
	).Replace(`<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>WebSSH 临时会话</title>
  <style>
    :root {
      color-scheme: dark;
      font-family: Inter, "Microsoft YaHei", Arial, sans-serif;
      --chrome: 86px;
      --bg: #07111f;
      --panel: #0d1728;
      --terminal: #020817;
      --line: #1c3358;
      --text: #dbeafe;
      --muted: #8ea6c7;
      --accent: #38bdf8;
    }
    * { box-sizing: border-box; }
    html, body { margin: 0; width: 100%; height: 100%; overflow: hidden; background: var(--bg); color: var(--text); }
    body { min-height: 100vh; }
    main { width: 100vw; height: 100vh; display: grid; grid-template-rows: var(--chrome) 1fr; background: var(--terminal); }
    header {
      min-height: var(--chrome);
      padding: 16px 24px;
      border-bottom: 1px solid var(--line);
      background: linear-gradient(180deg, #111c30 0%, var(--panel) 100%);
      display: flex;
      justify-content: space-between;
      gap: 18px;
      align-items: center;
    }
    .actions { display: flex; gap: 10px; align-items: center; }
    h1 { margin: 0; font-size: 20px; line-height: 1.25; letter-spacing: 0; }
    p { color: var(--muted); margin: 8px 0 0; font-size: 13px; }
    button { border: 1px solid #334155; border-radius: 8px; background: #111827; color: #dbeafe; padding: 8px 12px; cursor: pointer; font-weight: 700; }
    button:hover { border-color: var(--accent); }
    .terminal {
      min-height: 0;
      height: calc(100vh - var(--chrome));
      padding: 18px 22px 22px;
      background: radial-gradient(circle at 80% 0%, rgba(56,189,248,.08), transparent 28%), var(--terminal);
      font: 14px/1.5 "JetBrains Mono", "Cascadia Mono", Consolas, monospace;
    }
    #terminal {
      width: 100%;
      height: 100%;
      overflow: auto;
      white-space: pre-wrap;
      word-break: break-word;
      outline: none;
      caret-color: #86efac;
      scrollbar-color: #64748b transparent;
      scrollbar-width: thin;
    }
    #terminal::-webkit-scrollbar { width: 10px; height: 10px; }
    #terminal::-webkit-scrollbar-thumb { background: #475569; border-radius: 999px; border: 2px solid var(--terminal); }
    #terminal::-webkit-scrollbar-track { background: transparent; }
    .status { min-width: 108px; text-align: center; border: 1px solid #1e3a8a; border-radius: 999px; padding: 6px 10px; color: #bfdbfe; background: #0b1730; font-weight: 800; font-size: 12px; }
    .status.connected { border-color: #047857; color: #86efac; background: #052e25; }
    .status.closed { border-color: #475569; color: #cbd5e1; background: #111827; }
    .status.failed { border-color: #b91c1c; color: #fecaca; background: #450a0a; }
    .ok { color: #86efac; }
    .warn { color: #fbbf24; }
    .error { color: #fca5a5; }
    code { color: #bfdbfe; }
    @media (max-width: 760px) {
      :root { --chrome: 126px; }
      header { align-items: flex-start; flex-direction: column; }
      .actions { width: 100%; justify-content: space-between; }
      .terminal { padding: 14px; font-size: 13px; }
    }
  </style>
</head>
<body>
  <main>
    <header>
      <div>
        <h1>WebSSH 临时会话</h1>
        <p>资产 <code>__ASSET_HTML__</code> 的临时授权已校验，正在接入 SSH PTY。</p>
      </div>
      <div class="actions">
        <span id="connectionStatus" class="status">CONNECTING</span>
        <button id="closeSessionButton" type="button">关闭会话</button>
      </div>
    </header>
    <section class="terminal">
      <div id="terminal" tabindex="0"><span class="ok">$ opsledger webssh open --asset __ASSET_HTML__</span>
temporary credential accepted.
connecting interactive SSH PTY...
</div>
    </section>
  </main>
  <script>
	    const assetID = __ASSET_JS__;
	    const sessionID = __SESSION_JS__;
	    const terminal = document.getElementById("terminal");
    const statusEl = document.getElementById("connectionStatus");
    const closeButton = document.getElementById("closeSessionButton");
    const scheme = location.protocol === "https:" ? "wss" : "ws";
	    const ws = new WebSocket(scheme + "://" + location.host + "/webssh/ws/" + encodeURIComponent(sessionID) + "?asset=" + encodeURIComponent(assetID));
    const specialKeys = {
      ArrowUp: "\x1b[A",
      ArrowDown: "\x1b[B",
      ArrowRight: "\x1b[C",
      ArrowLeft: "\x1b[D",
      Delete: "\x1b[3~",
      Home: "\x1b[H",
      End: "\x1b[F",
      PageUp: "\x1b[5~",
      PageDown: "\x1b[6~"
    };
    function write(text) {
      terminal.textContent += text;
      terminal.scrollTop = terminal.scrollHeight;
    }
    function setStatus(text, className) {
      statusEl.textContent = text;
      statusEl.className = "status " + className;
    }
    ws.onopen = () => {
      write("\n[connected]\n");
      setStatus("CONNECTED", "connected");
      terminal.focus();
    };
    ws.onmessage = (event) => write(event.data);
    ws.onclose = () => {
      setStatus("CLOSED", "closed");
      write("\n[session closed]\n");
    };
    ws.onerror = () => {
      setStatus("FAILED", "failed");
      write("\n[websocket error]\n");
    };
    closeButton.addEventListener("click", () => {
      if (ws.readyState === WebSocket.OPEN || ws.readyState === WebSocket.CONNECTING) {
        ws.close();
      }
    });
    terminal.addEventListener("paste", (event) => {
      if (ws.readyState !== WebSocket.OPEN) return;
      const text = event.clipboardData.getData("text");
      if (text) ws.send(text);
      event.preventDefault();
    });
    terminal.addEventListener("keydown", (event) => {
      if (ws.readyState !== WebSocket.OPEN) return;
      if (event.ctrlKey && event.key.toLowerCase() === "c") {
        ws.send("\x03");
        event.preventDefault();
        return;
      }
      if (specialKeys[event.key]) {
        ws.send(specialKeys[event.key]);
        event.preventDefault();
        return;
      }
      if (event.key === "Enter") {
        ws.send("\r");
        event.preventDefault();
        return;
      }
      if (event.key === "Backspace") {
        ws.send("\x7f");
        event.preventDefault();
        return;
      }
      if (event.key === "Tab") {
        ws.send("\t");
        event.preventDefault();
        return;
      }
      if (event.key.length === 1 && !event.metaKey && !event.altKey) {
        ws.send(event.key);
        event.preventDefault();
      }
    });
    terminal.focus();
  </script>
</body>
</html>`)
	_, _ = io.WriteString(w, page)
}

func escapeHTMLForHTML(value string) string {
	return html.EscapeString(value)
}

func jsonString(value string) string {
	payload, err := json.Marshal(value)
	if err != nil {
		return `""`
	}
	return string(payload)
}

func (s *Server) handleWebSSHWebSocket(ws *websocket.Conn) {
	defer ws.Close()
	request := ws.Request()
	user, ok := currentUser(request.Context())
	if !ok {
		_ = websocket.Message.Send(ws, "unauthorized\n")
		return
	}
	assetID := request.URL.Query().Get("asset")
	sessionID := strings.TrimPrefix(request.URL.Path, "/webssh/ws/")
	grant, err := s.store.ValidateWebSSHSession(request.Context(), user, sessionID, assetID)
	if err != nil {
		_ = websocket.Message.Send(ws, "forbidden\n")
		return
	}
	sessionCtx, cancelSession := context.WithCancel(request.Context())
	defer cancelSession()
	if expiresAt, err := time.Parse(time.RFC3339, grant.ExpiresAt); err == nil {
		timer := time.AfterFunc(time.Until(expiresAt), cancelSession)
		defer timer.Stop()
	}
	sessionStatus := "closed"
	closeReason := "client disconnected"
	errorMessage := ""
	defer func() {
		if strings.TrimSpace(sessionID) == "" {
			return
		}
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if err := s.store.CloseWebSSHSession(ctx, user, sessionID, sessionStatus, closeReason, errorMessage); err != nil {
			log.Printf("close webssh session %s failed: %v", sessionID, err)
		}
	}()
	asset, err := s.store.GetAsset(sessionCtx, assetID)
	if err != nil {
		sessionStatus = "failed"
		closeReason = "asset lookup failed"
		errorMessage = err.Error()
		_ = websocket.Message.Send(ws, err.Error()+"\n")
		return
	}
	if err := bridgeSSHPTY(sessionCtx, ws, asset); err != nil {
		sessionStatus = "failed"
		closeReason = "ssh bridge failed"
		errorMessage = err.Error()
		if errors.Is(err, context.Canceled) && time.Now().Format(time.RFC3339) >= grant.ExpiresAt {
			sessionStatus = "expired"
			closeReason = "grant expired"
			errorMessage = ""
		}
		_ = websocket.Message.Send(ws, "\n"+err.Error()+"\n")
	}
}

func bridgeSSHPTY(ctx context.Context, ws *websocket.Conn, asset model.Asset) error {
	host := sshHostForAsset(asset)
	if host == "" {
		return errors.New("asset has no ssh host")
	}
	user := "ubuntu"
	if asset.Specs != nil && strings.TrimSpace(asset.Specs["ssh_user"]) != "" {
		user = strings.TrimSpace(asset.Specs["ssh_user"])
	}
	sshCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	args := []string{
		"-tt",
		"-o", "ConnectTimeout=8",
		"-o", "StrictHostKeyChecking=" + sshStrictHostKeyCheckingValue(),
	}
	if knownHosts := sshKnownHostsPath(); knownHosts != "" {
		args = append(args, "-o", "UserKnownHostsFile="+knownHosts)
	}
	args = append(args, user+"@"+host)
	cmd := exec.CommandContext(sshCtx, "ssh", args...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}

	output := make(chan string, 32)
	var outputWG sync.WaitGroup
	outputWG.Add(2)
	go copySSHOutput(stdout, output, &outputWG)
	go copySSHOutput(stderr, output, &outputWG)
	go func() {
		outputWG.Wait()
		close(output)
	}()

	inputDone := make(chan struct{})
	go func() {
		defer close(inputDone)
		for {
			var input string
			if err := websocket.Message.Receive(ws, &input); err != nil {
				return
			}
			if _, err := io.WriteString(stdin, input); err != nil {
				return
			}
		}
	}()

	waitDone := make(chan error, 1)
	go func() {
		waitDone <- cmd.Wait()
	}()

	killedByClient := false
	for {
		select {
		case <-ctx.Done():
			cancel()
			_ = cmd.Process.Kill()
			return ctx.Err()
		case <-inputDone:
			killedByClient = true
			cancel()
			_ = cmd.Process.Kill()
			err := <-waitDone
			for text := range output {
				if sendErr := websocket.Message.Send(ws, text); sendErr != nil {
					return sendErr
				}
			}
			if err != nil && !killedByClient {
				return fmt.Errorf("ssh session exited: %w", err)
			}
			return nil
		case err := <-waitDone:
			for text := range output {
				if sendErr := websocket.Message.Send(ws, text); sendErr != nil {
					return sendErr
				}
			}
			if err != nil && !killedByClient {
				return fmt.Errorf("ssh session exited: %w", err)
			}
			return nil
		case text, ok := <-output:
			if !ok {
				continue
			}
			if err := websocket.Message.Send(ws, text); err != nil {
				cancel()
				_ = cmd.Process.Kill()
				return err
			}
		}
	}
}

func copySSHOutput(reader io.Reader, output chan<- string, wg *sync.WaitGroup) {
	defer wg.Done()
	buffer := make([]byte, 4096)
	for {
		n, err := reader.Read(buffer)
		if n > 0 {
			output <- string(buffer[:n])
		}
		if err != nil {
			return
		}
	}
}

func sshHostForAsset(asset model.Asset) string {
	if asset.Specs != nil {
		if host := strings.TrimSpace(asset.Specs["public_ip"]); host != "" {
			return host
		}
		if host := strings.TrimSpace(asset.Specs["private_ip"]); host != "" {
			return host
		}
	}
	parts := strings.Split(asset.Endpoint, "/")
	for i := len(parts) - 1; i >= 0; i-- {
		host := strings.TrimSpace(parts[i])
		if host != "" {
			return host
		}
	}
	return strings.TrimSpace(asset.Endpoint)
}

func (s *Server) handleOpenWebSSH(w http.ResponseWriter, r *http.Request) {
	var req model.WebSSHOpenRequest
	if err := decodeJSON(r.Body, &req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	user, ok := currentUser(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, errors.New("unauthorized"))
		return
	}
	session, err := s.store.OpenWebSSH(r.Context(), user, req.AssetID, clientIP(r), r.UserAgent())
	if err != nil {
		s.audit(r.Context(), r, user, "webssh.open", "asset", req.AssetID, "", "failed", err.Error(), nil)
		writeStoreError(w, err)
		return
	}
	s.audit(r.Context(), r, user, "webssh.open", "asset", session.AssetID, session.AssetName, "success", "打开 WebSSH 会话", map[string]string{
		"session_id":      session.ID,
		"access_grant_id": session.AccessGrantID,
		"expires_at":      session.ExpiresAt,
	})
	writeJSON(w, http.StatusCreated, session)
}

func sshStrictHostKeyCheckingValue() string {
	if !envBool("OPSLEDGER_SSH_STRICT_HOST_KEY", false) {
		return "no"
	}
	return "yes"
}

func sshKnownHostsPath() string {
	return strings.TrimSpace(os.Getenv("OPSLEDGER_SSH_KNOWN_HOSTS"))
}
