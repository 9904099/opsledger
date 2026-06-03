function syncFormsFromSelection() {
  const selectedCloudAccount = state.cloudAccounts.find((item) => item.id === state.selectedCloudAccountId);
  if (selectedCloudAccount) {
    refs.cloudAccountId.value = selectedCloudAccount.id;
    refs.cloudAccountPlatformCode.value = selectedCloudAccount.platform_code || "aws";
    refs.cloudAccountName.value = selectedCloudAccount.name || "";
    refs.cloudAccountAccountId.value = selectedCloudAccount.account_id || "";
    refs.cloudAccountDefaultRegion.value = selectedCloudAccount.default_region || "";
    refs.cloudAccountEnvironment.value = selectedCloudAccount.environment || "prod";
    refs.cloudAccountOwner.value = selectedCloudAccount.owner || "Ops";
    refs.cloudAccountCriticality.value = selectedCloudAccount.criticality || "medium";
    refs.cloudAccountAccessKeyId.value = "";
    refs.cloudAccountSecretAccessKey.value = "";
    refs.cloudAccountSyncEnabled.checked = !!selectedCloudAccount.sync_enabled;
    refs.cloudAccountSyncMode.value = selectedCloudAccount.sync_mode || "manual";
    applySyncScheduleToControls(selectedCloudAccount.sync_cron || "");
  } else {
    resetCloudAccountForm();
  }

  const selectedAsset = state.assets.find((item) => item.id === state.selectedAssetId);
  if (selectedAsset) {
    refs.assetId.value = selectedAsset.id;
    refs.assetPlatformCode.value = selectedAsset.platform_code || "";
    refs.assetPlatformName.value = selectedAsset.platform_name || "";
    refs.assetCloudAccountName.value = selectedAsset.cloud_account_name || "";
    refs.assetAccountId.value = selectedAsset.account_id || "";
    refs.assetProjectCode.value = selectedAsset.project_code || "public";
    refs.assetCategory.value = selectedAsset.category || "";
    refs.assetResourceType.value = selectedAsset.resource_type || "";
    refs.assetRegion.value = selectedAsset.region || "";
    refs.assetEnvironment.value = selectedAsset.environment || "prod";
    refs.assetName.value = selectedAsset.name || "";
    refs.assetEndpoint.value = selectedAsset.endpoint || "";
    refs.assetOwner.value = selectedAsset.owner || "";
    refs.assetStatus.value = selectedAsset.status || "active";
    refs.assetCriticality.value = selectedAsset.criticality || "medium";
    refs.assetLastCheckedAt.value = selectedAsset.last_checked_at || "";
    refs.assetTags.value = (selectedAsset.tags || []).join(", ");
    refs.assetNotes.value = selectedAsset.notes || "";
    refs.inspectionAssetId.value = selectedAsset.id;
    refs.inspectionAssetName.value = selectedAsset.name || "";
    refs.inspectionExecutor.value = selectedAsset.owner || "";
    refs.inspectionCheckedAt.value = new Date().toISOString().slice(0, 10);
  } else {
    resetAssetForm();
    resetInspectionForm();
  }

  const selectedTool = state.tools.find((item) => item.id === state.selectedToolId);
  if (selectedTool) {
    refs.toolId.value = selectedTool.id;
    refs.toolEnvironment.value = selectedTool.environment || "dev";
    refs.toolType.value = selectedTool.tool_type || "business";
    refs.toolName.value = selectedTool.asset_name || "";
    refs.toolEndpoint.value = selectedTool.endpoint || "";
    refs.toolOwner.value = selectedTool.owner || "Ops";
    refs.toolStatus.value = selectedTool.status || "active";
    refs.toolCriticality.value = selectedTool.criticality || "medium";
    refs.toolLoginPolicy.value = selectedTool.login_policy === "webssh" ? "sso" : (selectedTool.login_policy || "sso");
    refs.toolCredentialPolicy.value = selectedTool.credential_policy || "none";
    refs.toolTags.value = (selectedTool.tags || []).join(", ");
    refs.toolApprovalRequired.checked = !!selectedTool.approval_required;
    refs.toolWebsshEnabled.checked = !!selectedTool.webssh_enabled;
    refs.toolDescription.value = selectedTool.description || "";
  } else {
    resetToolForm();
  }

  if (!state.selectedCredentialId && refs.credentialForm) {
    resetCredentialForm();
  }

  const selectedUser = state.users.find((item) => item.id === state.selectedUserId);
  if (selectedUser) {
    refs.userId.value = selectedUser.id;
    refs.userUsername.value = selectedUser.username || "";
    refs.userDisplayName.value = selectedUser.display_name || "";
    refs.userEmail.value = selectedUser.email || "";
    refs.userPhone.value = selectedUser.phone || "";
    refs.userRole.value = selectedUser.role || "developer";
    refs.userTeam.value = selectedUser.team || "";
    refs.userPassword.value = "";
    refs.userStatus.value = selectedUser.status || "active";
  } else {
    resetUserForm();
  }
  renderRoleOptions();

  const selectedChange = state.changes.find((item) => item.id === state.selectedChangeId);
  if (selectedChange) {
    refs.changeId.value = selectedChange.id;
    refs.changeAssetId.value = selectedChange.asset_id || "";
    refs.changeTitle.value = selectedChange.title || "";
    refs.changeCategory.value = selectedChange.category || "release";
    refs.changeExecutor.value = selectedChange.executor || "";
    refs.changeRiskLevel.value = selectedChange.risk_level || "medium";
    refs.changeStatus.value = selectedChange.status || "planned";
    refs.changeWindow.value = selectedChange.window || "";
    refs.changeRollbackPlan.value = selectedChange.rollback_plan || "";
    refs.changeSummary.value = selectedChange.summary || "";
  } else {
    resetChangeForm();
    if (state.selectedAssetId) {
      refs.changeAssetId.value = state.selectedAssetId;
    }
  }
}

function resetCloudAccountForm() {
  refs.cloudAccountForm.reset();
  refs.cloudAccountId.value = "";
  if (state.platforms[0]) {
    refs.cloudAccountPlatformCode.value = state.platforms[0].code;
  }
  refs.cloudAccountEnvironment.value = "prod";
  refs.cloudAccountOwner.value = "Ops";
  refs.cloudAccountCriticality.value = "medium";
  refs.cloudAccountSyncMode.value = "manual";
  refs.cloudAccountSyncEnabled.checked = false;
  setSyncPeriodValues({hours: 6});
  refs.cloudAccountSyncCron.value = "";
  updateSyncControlVisibility();
}

function applySyncScheduleToControls(value) {
  const expr = String(value || "").trim();
  refs.cloudAccountSyncCron.value = expr;
  const period = parseSyncPeriod(expr);
  if (period) {
    refs.cloudAccountSyncMode.value = "scheduled";
    setSyncPeriodValues(period);
    updateSyncControlVisibility();
    return;
  }
  const fields = expr.split(/\s+/).filter(Boolean);
  if (fields.length === 5 && /^\d+$/.test(fields[0]) && /^\d+$/.test(fields[1])) {
    refs.cloudAccountSyncMode.value = "scheduled";
    setSyncPeriodValues({days: 1});
    updateSyncControlVisibility();
    return;
  }
  setSyncPeriodValues({hours: 6});
  updateSyncControlVisibility();
}

function syncScheduleValueFromControls() {
  if (refs.cloudAccountSyncMode.value === "manual" || !refs.cloudAccountSyncEnabled.checked) {
    return "";
  }
  const parts = [
    ["years", "y"],
    ["months", "mo"],
    ["weeks", "w"],
    ["days", "d"],
    ["hours", "h"],
    ["minutes", "m"],
    ["seconds", "s"]
  ];
  const result = parts.map(([key, suffix]) => {
    const value = syncPeriodValue(key);
    return value > 0 ? `${value}${suffix}` : "";
  }).filter(Boolean).join("");
  return result || "6h";
}

function updateSyncControlVisibility() {
  const scheduled = refs.cloudAccountSyncMode.value !== "manual";
  if (!scheduled) {
    refs.cloudAccountSyncEnabled.checked = false;
  }
  document.getElementById("syncPeriodField").classList.toggle("hidden", !scheduled);
  refs.cloudAccountSyncEnabled.closest("label").classList.toggle("hidden", !scheduled);
  updateSyncControlLabels();
}

function updateSyncControlLabels() {
  if (!refs.cloudAccountSyncMode || typeof t !== "function") {
    return;
  }
  const modeLabels = {
    manual: t("sync.manual"),
    scheduled: t("sync.scheduled")
  };
  Array.from(refs.cloudAccountSyncMode.options).forEach((option) => {
    option.textContent = modeLabels[option.value] || option.value;
  });
}

function parseSyncPeriod(expr) {
  if (!expr) {
    return null;
  }
  if (/^\d{1,2}:\d{2}$/.test(expr)) {
    return {days: 1};
  }
  const simple = expr.match(/^(\d+)(ms|s|m|h)$/);
  if (simple) {
    const value = Number(simple[1]);
    return simple[2] === "s" ? {seconds: value} : simple[2] === "m" ? {minutes: value} : simple[2] === "h" ? {hours: value} : null;
  }
  const result = {};
  const pattern = /(\d+)(y|mo|w|d|h|m|s)/g;
  let match;
  let matched = false;
  while ((match = pattern.exec(expr)) !== null) {
    matched = true;
    const value = Number(match[1]);
    const unit = match[2];
    if (unit === "y") result.years = value;
    if (unit === "mo") result.months = value;
    if (unit === "w") result.weeks = value;
    if (unit === "d") result.days = value;
    if (unit === "h") result.hours = value;
    if (unit === "m") result.minutes = value;
    if (unit === "s") result.seconds = value;
  }
  return matched ? result : null;
}

function setSyncPeriodValues(values = {}) {
  refs.cloudAccountSyncYears.value = values.years || 0;
  refs.cloudAccountSyncMonths.value = values.months || 0;
  refs.cloudAccountSyncWeeks.value = values.weeks || 0;
  refs.cloudAccountSyncDays.value = values.days || 0;
  refs.cloudAccountSyncHours.value = values.hours || 0;
  refs.cloudAccountSyncMinutes.value = values.minutes || 0;
  refs.cloudAccountSyncSeconds.value = values.seconds || 0;
}

function syncPeriodValue(key) {
  const input = refs[`cloudAccountSync${key.charAt(0).toUpperCase()}${key.slice(1)}`];
  const value = Number(input && input.value);
  return Number.isFinite(value) && value > 0 ? Math.floor(value) : 0;
}

function resetAssetForm() {
  refs.assetForm.reset();
  refs.assetId.value = "";
  refs.assetProjectCode.value = "public";
  refs.assetEnvironment.value = "prod";
  refs.assetStatus.value = "active";
  refs.assetCriticality.value = "high";
  refs.assetLastCheckedAt.value = new Date().toISOString().slice(0, 10);
}

function resetAssetBulkForm() {
  refs.assetBulkForm.reset();
  renderAssetBulkScopeHint();
}

function buildAssetBulkPayload(assets) {
  const payload = {
    asset_ids: assets.map((asset) => asset.id)
  };
  assignBulkField(payload, "project_code", refs.assetBulkProjectCode.value);
  assignBulkField(payload, "environment", refs.assetBulkEnvironment.value);
  assignBulkField(payload, "status", refs.assetBulkStatus.value);
  assignBulkField(payload, "criticality", refs.assetBulkCriticality.value);
  assignBulkField(payload, "owner", refs.assetBulkOwner.value);
  assignBulkField(payload, "category", refs.assetBulkCategory.value);
  assignBulkField(payload, "resource_type", refs.assetBulkResourceType.value);
  assignBulkField(payload, "region", refs.assetBulkRegion.value);
  assignBulkField(payload, "notes", refs.assetBulkNotes.value);
  if (refs.assetBulkTags.value.trim()) {
    payload.tags = refs.assetBulkTags.value.split(",").map((item) => item.trim()).filter(Boolean);
  }
  return payload;
}

function assignBulkField(payload, key, value) {
  if (String(value || "").trim() !== "") {
    payload[key] = String(value).trim();
  }
}

function hasAssetBulkPatch(payload) {
  return Object.keys(payload).some((key) => key !== "asset_ids");
}

function resetToolForm() {
  refs.toolForm.reset();
  refs.toolId.value = "";
  refs.toolEnvironment.value = "global";
  refs.toolType.value = "business";
  refs.toolOwner.value = "Ops";
  refs.toolStatus.value = "active";
  refs.toolCriticality.value = "medium";
  refs.toolLoginPolicy.value = "sso";
  refs.toolCredentialPolicy.value = "none";
}

function resetCredentialForm() {
  refs.credentialForm.reset();
  refs.credentialOwnerType.value = "asset";
  refs.credentialKind.value = "password";
  refs.credentialKeyName.value = "default";
  refs.credentialAccessPolicy.value = "ops_only";
  refs.credentialStatus.value = "active";
  refs.credentialValue.value = "";
  renderCredentialOptions();
}

function resetUserForm() {
  refs.userForm.reset();
  refs.userId.value = "";
  refs.userRole.value = "developer";
  refs.userStatus.value = "active";
  refs.userPassword.value = "";
}

function resetRoleForm() {
  refs.roleForm.reset();
  refs.roleId.value = "";
  refs.roleCode.value = "";
  refs.roleName.value = "";
  refs.roleLevel.value = "50";
  refs.roleStatus.value = "active";
}

function resetPermissionForm() {
  refs.permissionForm.reset();
  refs.permissionId.value = "";
  refs.permissionRole.value = state.roles[0]?.code || "developer";
  refs.permissionEnvironment.value = "*";
  refs.permissionProjectCode.value = "*";
  refs.permissionScope.value = "tool";
  refs.permissionAction.value = "view";
  refs.permissionRequiresApproval.checked = false;
}

function resetApprovalFlowForm() {
  refs.approvalFlowForm.reset();
  refs.approvalFlowId.value = "";
  refs.approvalFlowName.value = "";
  refs.approvalFlowScope.value = "credential";
  refs.approvalFlowEnvironment.value = "*";
  refs.approvalFlowStatus.value = "active";
  refs.approvalFlowDescription.value = "";
  state.approvalFlowSteps = [{
    approver_role: "ops",
    approver_label: "Ops Engineer",
    required_action: "approved",
    timeout_minutes: 60
  }];
  renderApprovalFlowSteps();
}

function resetChangeForm() {
  refs.changeForm.reset();
  refs.changeId.value = "";
  refs.changeCategory.value = "release";
  refs.changeRiskLevel.value = "medium";
  refs.changeStatus.value = "planned";
  if (state.pendingChangeTitlePrefix) {
    refs.changeTitle.value = state.pendingChangeTitlePrefix;
  }
}

function resetInspectionForm() {
  refs.inspectionForm.reset();
  refs.inspectionAssetId.value = "";
  refs.inspectionAssetName.value = "";
  refs.inspectionCheckedAt.value = new Date().toISOString().slice(0, 10);
}
