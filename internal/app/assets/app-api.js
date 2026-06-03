async function apiFetch(url, method, payload, options = {}) {
  const headers = {
    "Content-Type": "application/json"
  };
  const csrfToken = readCookie("opsledger_csrf");
  if (csrfToken && !["GET", "HEAD", "OPTIONS"].includes(String(method).toUpperCase())) {
    headers["X-OpsLedger-CSRF"] = csrfToken;
  }
  const response = await fetch(url, {
    method,
    credentials: "same-origin",
    headers,
    body: payload ? JSON.stringify(payload) : undefined
  });

  const data = await response.json().catch(() => ({}));
  if (!response.ok) {
    handleAPIError(response, data, options);
  }
  return data;
}

async function apiFetchForm(url, method, formData, options = {}) {
  const headers = {};
  const csrfToken = readCookie("opsledger_csrf");
  if (csrfToken && !["GET", "HEAD", "OPTIONS"].includes(String(method).toUpperCase())) {
    headers["X-OpsLedger-CSRF"] = csrfToken;
  }
  const response = await fetch(url, {
    method,
    credentials: "same-origin",
    headers,
    body: formData
  });
  const data = await response.json().catch(() => ({}));
  if (!response.ok) {
    handleAPIError(response, data, options);
  }
  return data;
}

function handleAPIError(response, data, options = {}) {
  const message = data.error || "请求失败";
  if (response.status === 401 && !options.skipAuthRedirect) {
    state.currentUser = null;
    state.currentPermissions = [];
    state.page = "login";
    renderPage();
  }
  if (!options.silent) {
    showMessage(message, "error");
  }
  throw new Error(message);
}
