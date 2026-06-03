async function loadCurrentUser(message = "") {
  try {
    const result = await apiFetch("/api/auth/me", "GET", null, {skipAuthRedirect: true, silent: true});
    state.currentUser = result.user;
    state.currentPermissions = result.permissions || [];
    if (state.page === "login") {
      state.page = defaultPageForRole(state.currentUser.role);
    }
    await loadData(message || "已恢复登录状态。", message ? "success" : "");
  } catch (error) {
    state.currentUser = null;
    state.currentPermissions = [];
    await loadSetupStatus();
    renderPage();
  }
}

async function loadSetupStatus() {
  try {
    const result = await apiFetch("/api/setup", "GET", null, {skipAuthRedirect: true, silent: true});
    state.setup = result || {required: false};
    state.page = result && result.required ? "setup" : "login";
  } catch (error) {
    state.setup = {required: false, database_ready: false, message: error.message || "setup check failed"};
    state.page = "login";
  }
}

async function loadData(message = "", level = "success") {
  const data = await apiFetch("/api/bootstrap", "GET");
  state.platforms = data.platforms || [];
  state.cloudAccounts = data.cloud_accounts || [];
  state.costRecords = data.cloud_account_cost_records || [];
  state.assets = data.assets || [];
  state.environments = data.environments || [];
  state.tools = data.tools || [];
  state.users = data.users || [];
  state.roles = data.roles || [];
  state.permissions = data.permissions || [];
  state.approvalFlows = data.approval_flows || [];
  state.approvals = data.approvals || [];
  state.accessGrants = data.access_grants || [];
  state.credentials = data.credentials || [];
  state.changes = data.changes || [];
  state.inspections = data.inspections || [];
  state.attachments = data.attachments || [];
  state.probes = data.probes || [];
  state.alerts = data.alerts || [];
  state.recentSyncs = data.recent_syncs || [];
  state.auditEvents = data.audit_events || [];
  state.summary = data.summary || {};

  if (state.selectedCloudAccountId && !state.cloudAccounts.find((item) => item.id === state.selectedCloudAccountId)) {
    state.selectedCloudAccountId = "";
  }
  if (state.selectedAssetId && !state.assets.find((item) => item.id === state.selectedAssetId)) {
    state.selectedAssetId = "";
  }
  if (state.selectedChangeId && !state.changes.find((item) => item.id === state.selectedChangeId)) {
    state.selectedChangeId = "";
  }
  if (state.selectedToolId && !state.tools.find((item) => item.id === state.selectedToolId)) {
    state.selectedToolId = "";
  }
  if (state.selectedUserId && !state.users.find((item) => item.id === state.selectedUserId)) {
    state.selectedUserId = "";
  }
  const workbenchEnvs = developerWorkbenchEnvironments();
  if (!workbenchEnvs.find((item) => item.code === state.developerEnvironment) && workbenchEnvs[0]) {
    state.developerEnvironment = workbenchEnvs[0].code;
  }

  syncFormsFromSelection();
  refs.lastUpdated.textContent = `${t("message.lastUpdated")}${formatDateTime(data.generated_at)}`;
  render();

  if (message) {
    showMessage(message, level);
  }
}

function render() {
  renderPage();
  renderCurrentUser();
  if (state.page === "login" || state.page === "setup") {
    renderSetupState();
    applyI18n();
    return;
  }
  renderSummary();
  renderWorkbenchApprovals();
  renderEnvironments();
  renderAssetFilters();
  renderAssetTree();
  renderAssetDetail();
  if (state.page === "developer") {
    renderDeveloperPage();
    renderApprovals();
  }
  if (state.page === "config") {
    renderPlatforms();
    renderCloudAccounts();
    renderTools();
    renderCredentialOptions();
    renderCredentials();
    renderTagGovernance();
    renderChargeback();
    renderUsers();
    renderPermissions();
    renderApprovals();
    renderSectionVisibility();
    renderConfigDomain();
    renderChangeOptions();
    renderInspectionConfigState();
    renderAlerts();
    renderRecentSyncs();
  }
  if (state.page === "audit") {
    renderAuditEvents();
  }
  refs.deleteAssetButton.disabled = !refs.assetId.value;
  refs.deleteChangeButton.disabled = !refs.changeId.value;
  refs.assetFormTitle.textContent = refs.assetId.value ? "编辑资产" : "新增资产";
  refs.toolFormTitle.textContent = refs.toolId.value ? "编辑工具资产" : "新增工具资产";
  refs.userFormTitle.textContent = refs.userId.value ? "编辑用户" : "用户权限";
  refs.changeFormTitle.textContent = refs.changeId.value ? "编辑变更" : "新增变更";
  applyI18n();
}

function renderPage() {
  enforceRolePage();
  const authPage = state.page === "login" || state.page === "setup";
  refs.topbar.classList.toggle("hidden", authPage);
  refs.loginPage.classList.toggle("hidden", !authPage);
  refs.mainWorkbench.classList.toggle("hidden", state.page !== "workbench");
  refs.developerPage.classList.toggle("hidden", state.page !== "developer");
  refs.auditPage.classList.toggle("hidden", state.page !== "audit");
  refs.configPage.classList.toggle("hidden", state.page !== "config");
  renderSetupState();
}

function renderSetupState() {
  const setupRequired = state.page === "setup";
  if (refs.loginForm) {
    refs.loginForm.classList.toggle("hidden", setupRequired);
  }
  if (refs.setupForm) {
    refs.setupForm.classList.toggle("hidden", !setupRequired);
  }
  if (refs.loginSubtitle) {
    refs.loginSubtitle.textContent = setupRequired
      ? t("setup.subtitle")
      : t("login.subtitle");
  }
  if (refs.setupDatabaseStatus) {
    const setup = state.setup || {};
    const driver = setup.driver ? ` / ${setup.driver}` : "";
    refs.setupDatabaseStatus.textContent = setup.database_ready
      ? `${t("setup.databaseReady")}${driver}`
      : t("setup.databaseChecking");
  }
}

function renderCurrentUser() {
  if (!state.currentUser) {
    refs.currentUserBadge.textContent = "";
    refs.logoutButton.classList.add("hidden");
    return;
  }
  refs.currentUserBadge.textContent = `${state.currentUser.display_name || state.currentUser.username} / ${state.currentUser.role}`;
  refs.logoutButton.classList.remove("hidden");
  refs.openDeveloperPageButton.classList.add("hidden");
  refs.closeDeveloperPageButton.classList.add("hidden");
  refs.openAuditPageButton.classList.toggle("hidden", !canOpenAudit());
  refs.openConfigPageButton.classList.toggle("hidden", !canOpenConfig());
}

function roleOfCurrentUser() {
  return state.currentUser ? state.currentUser.role : "";
}

function defaultPageForRole(role) {
  switch (role) {
    case "developer":
    case "lead":
      return "developer";
    case "admin":
    case "ops":
      return "workbench";
    case "auditor":
      return "audit";
    case "viewer":
    default:
      return "workbench";
  }
}

function canOpenDeveloper() {
  return ["developer", "lead"].includes(roleOfCurrentUser());
}

function canOpenConfig() {
  return ["admin", "ops"].includes(roleOfCurrentUser());
}

function canOpenAudit() {
  return ["admin", "ops", "auditor"].includes(roleOfCurrentUser());
}

function enforceRolePage() {
  if (!state.currentUser || state.page === "login") {
    return;
  }
  if (state.page === "config" && !canOpenConfig()) {
    state.page = defaultPageForRole(roleOfCurrentUser());
  }
  if (state.page === "audit" && !canOpenAudit()) {
    state.page = defaultPageForRole(roleOfCurrentUser());
  }
  if (state.page === "developer" && !canOpenDeveloper()) {
    state.page = defaultPageForRole(roleOfCurrentUser());
  }
  if (state.page === "workbench" && ["developer", "lead"].includes(roleOfCurrentUser())) {
    state.page = defaultPageForRole(roleOfCurrentUser());
  }
}

function renderAssetFilters() {
  const projects = Array.from(new Set(state.assets.map((item) => item.project_code || "public").filter(Boolean))).sort((a, b) => projectLabel(a).localeCompare(projectLabel(b)));
  const cloudAccounts = Array.from(new Set(state.assets.map((item) => item.cloud_account_name).filter(Boolean))).sort();
  const resourceTypes = Array.from(new Set(state.assets.map((item) => item.resource_type).filter(Boolean))).sort();

  refs.treeProjectFilter.innerHTML = `
    <option value="">全部项目</option>
    ${projects.map((value) => `<option value="${escapeHTML(value)}">${escapeHTML(projectLabel(value))}</option>`).join("")}
  `;
  refs.treeCloudAccountFilter.innerHTML = `
    <option value="">全部云账号</option>
    ${cloudAccounts.map((value) => `<option value="${escapeHTML(value)}">${escapeHTML(value)}</option>`).join("")}
  `;
  refs.treeResourceTypeFilter.innerHTML = `
    <option value="">全部资源类型</option>
    ${resourceTypes.map((value) => `<option value="${escapeHTML(value)}">${escapeHTML(value)}</option>`).join("")}
  `;

  refs.treeStatusFilter.value = state.treeStatusFilter;
  refs.treeProjectFilter.value = state.treeProjectFilter;
  refs.treeCloudAccountFilter.value = state.treeCloudAccountFilter;
  refs.treeResourceTypeFilter.value = state.treeResourceTypeFilter;
  renderAssetBulkScopeHint();
}

function renderAssetBulkScopeHint() {
  if (!refs.assetBulkScopeHint) {
    return;
  }
  refs.assetBulkScopeHint.textContent = `当前筛选命中 ${filterAssets().length} 条资产`;
}

function projectLabel(code) {
  const labels = {
    edge: "edge",
    cloud: "cloud",
    enterprise: "enterprise",
    business: "business",
    public: "公共资源",
    pve: "pve"
  };
  return labels[code || "public"] || code || "公共资源";
}

function renderSummary() {
  const summary = state.summary || {};
  const cards = [
    ["总资产", summary.total_assets || 0, "", "dashboard"],
    ["运行中", summary.active_assets || 0, "ok", "active"],
    ["维护中", summary.maintenance_assets || 0, "warn", "maintenance"],
    ["高重要级别", summary.critical_assets || 0, "danger", "critical"],
    ["拨测异常", summary.probe_alerts || 0, summary.probe_alerts ? "danger" : "ok", "probe alerts"],
    ["未处理告警", summary.open_alerts || 0, summary.open_alerts ? "danger" : "ok", "open alerts"],
    ["工具资产", summary.tool_assets || 0, "", "tools"],
    ["待审批", summary.pending_approvals || 0, summary.pending_approvals ? "warn" : "ok", "approvals"],
    ["待执行变更", summary.planned_changes || 0, "", "planned"],
    ["执行中变更", summary.in_progress_changes || 0, "warn", "in_progress"],
    ["云账号数", state.cloudAccounts.length || 0, "", "accounts"]
  ];

  refs.summaryCards.innerHTML = cards.map(([title, value, className, hint]) => `
    <article class="summary-card ${className}">
      <span>${title}</span>
      <strong>${value}</strong>
      <small>${hint || "dashboard"}</small>
    </article>
  `).join("");
}

function filterAssets() {
  return state.assets.filter((asset) => {
    if (state.treeStatusFilter && asset.status !== state.treeStatusFilter) {
      return false;
    }
    if (state.treeProjectFilter && (asset.project_code || "public") !== state.treeProjectFilter) {
      return false;
    }
    if (state.treeCloudAccountFilter && asset.cloud_account_name !== state.treeCloudAccountFilter) {
      return false;
    }
    if (state.treeResourceTypeFilter && asset.resource_type !== state.treeResourceTypeFilter) {
      return false;
    }
    if (!state.search) {
      return true;
    }
    const blob = [
      asset.platform_name,
      asset.platform_code,
      asset.project_code,
      projectLabel(asset.project_code),
      asset.cloud_account_name,
      asset.account_id,
      asset.category,
      asset.resource_type,
      asset.region,
      asset.environment,
      asset.name,
      asset.endpoint,
      asset.owner,
      asset.status,
      asset.criticality,
      asset.notes,
      ...(asset.tags || [])
    ].join(" ").toLowerCase();
    return blob.includes(state.search);
  });
}

function bindSectionToggle(button, sectionKey) {
  if (!button) {
    return;
  }
  button.addEventListener("click", () => {
    if (state.collapsedSections.has(sectionKey)) {
      state.collapsedSections.delete(sectionKey);
    } else {
      state.collapsedSections.add(sectionKey);
    }
    renderSectionVisibility();
  });
}

function updateSectionToggleLabel(button, sectionKey) {
  if (!button) {
    return;
  }
  button.textContent = state.collapsedSections.has(sectionKey) ? "展开" : "收起";
}

function showMessage(message, level = "success") {
  refs.flashMessage.textContent = translateUIMessage(message);
  refs.flashMessage.className = `flash ${level}`;
  if (!message) {
    refs.flashMessage.classList.add("hidden");
    return;
  }
  window.clearTimeout(showMessage.timer);
  showMessage.timer = window.setTimeout(() => {
    refs.flashMessage.className = "flash hidden";
    refs.flashMessage.textContent = "";
  }, 4200);
}
