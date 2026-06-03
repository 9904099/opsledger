function bindEvents() {
  refs.loginForm.addEventListener("submit", async (event) => {
    event.preventDefault();
    await completeLogin(refs.loginUsername.value, refs.loginPassword.value);
  });

  refs.setupForm.addEventListener("submit", async (event) => {
    event.preventDefault();
    await completeSetup();
  });

  refs.logoutButton.addEventListener("click", async () => {
    try {
      await apiFetch("/api/auth/logout", "POST", {}, {skipAuthRedirect: true, silent: true});
    } catch (error) {
      console.warn("logout request failed, clearing local session state", error);
    }
    state.currentUser = null;
    state.currentPermissions = [];
    state.page = "login";
    render();
    showMessage("已退出登录。", "success");
  });

  refs.languageToggleButton.addEventListener("click", () => {
    toggleLanguage();
  });

  document.querySelectorAll("[data-login-user]").forEach((button) => {
    button.addEventListener("click", async () => {
      refs.loginUsername.value = button.dataset.loginUser;
      refs.loginPassword.focus();
    });
  });

  refs.searchInput.addEventListener("input", (event) => {
    state.search = event.target.value.trim().toLowerCase();
    render();
  });

  refs.treeStatusFilter.addEventListener("change", (event) => {
    state.treeStatusFilter = event.target.value;
    render();
  });

  refs.treeModeAccountButton.addEventListener("click", () => {
    state.treeMode = "account";
    render();
  });

  refs.treeModeProjectButton.addEventListener("click", () => {
    state.treeMode = "project";
    render();
  });

  refs.treeProjectFilter.addEventListener("change", (event) => {
    state.treeProjectFilter = event.target.value;
    render();
  });

  refs.treeCloudAccountFilter.addEventListener("change", (event) => {
    state.treeCloudAccountFilter = event.target.value;
    render();
  });

  refs.treeResourceTypeFilter.addEventListener("change", (event) => {
    state.treeResourceTypeFilter = event.target.value;
    render();
  });

  refs.reloadButton.addEventListener("click", async () => {
    await loadData("已刷新台账数据。", "success");
  });

  refs.openConfigPageButton.addEventListener("click", () => {
    if (!canOpenConfig()) {
      showMessage("当前角色不能进入配置中心。", "error");
      return;
    }
    state.page = "config";
    render();
  });

  refs.openAuditPageButton.addEventListener("click", () => {
    if (!canOpenAudit()) {
      showMessage("当前角色不能进入审计工作台。", "error");
      return;
    }
    state.page = "audit";
    render();
  });

  refs.openDeveloperPageButton.addEventListener("click", () => {
    if (!canOpenDeveloper()) {
      showMessage("当前角色不能进入开发者工作台。", "error");
      return;
    }
    state.page = "developer";
    render();
  });

  refs.openApprovalModalButton.addEventListener("click", () => {
    openApprovalModal("request");
  });

  refs.workbenchApprovalInboxButton.addEventListener("click", () => {
    openApprovalModal("inbox");
  });

  refs.closeApprovalModalButton.addEventListener("click", () => {
    closeApprovalModal();
  });

  refs.closeDeveloperAssetModalButton.addEventListener("click", () => {
    closeDeveloperAssetModal();
  });

  refs.closeConfigPageButton.addEventListener("click", () => {
    state.page = "workbench";
    render();
  });

  refs.closeDeveloperPageButton.addEventListener("click", () => {
    state.page = "workbench";
    render();
  });

  refs.configDomainTabs.querySelectorAll("[data-config-domain]").forEach((button) => {
    button.addEventListener("click", () => {
      state.configDomain = button.dataset.configDomain;
      renderConfigDomain();
    });
  });

  refs.auditSearchInput.addEventListener("input", (event) => {
    state.auditSearch = event.target.value.trim().toLowerCase();
    renderAuditEvents();
  });

  refs.auditActionFilter.addEventListener("change", (event) => {
    state.auditActionFilter = event.target.value;
    renderAuditEvents();
  });

  refs.auditOutcomeFilter.addEventListener("change", (event) => {
    state.auditOutcomeFilter = event.target.value;
    renderAuditEvents();
  });

  bindSectionToggle(refs.toggleCloudAccountSectionButton, "cloudAccount");
  bindSectionToggle(refs.toggleAssetSectionButton, "asset");
  bindSectionToggle(refs.toggleToolSectionButton, "tool");
  bindSectionToggle(refs.toggleCredentialSectionButton, "credential");
  bindSectionToggle(refs.toggleUserSectionButton, "user");
  bindSectionToggle(refs.toggleChangeSectionButton, "change");
  bindSectionToggle(refs.toggleInspectionSectionButton, "inspection");

  refs.newCloudAccountButton.addEventListener("click", () => {
    state.selectedCloudAccountId = "";
    state.detailMode = "cloudAccount";
    resetCloudAccountForm();
    state.page = "config";
    render();
  });

  refs.syncCloudAccountButton.addEventListener("click", async () => {
    const cloudAccountId = refs.cloudAccountId.value;
    if (!cloudAccountId) {
      showMessage("请先选择云账号。", "error");
      return;
    }
    const result = await apiFetch("/api/cloud-accounts/sync", "POST", {
      cloud_account_id: cloudAccountId,
      region: refs.cloudAccountDefaultRegion.value
    });
    const warnings = (result.warnings || []).length;
    await loadData(formatSyncMessage(result), warnings ? "error" : "success");
  });

  refs.syncCloudAccountCostButton.addEventListener("click", async () => {
    const cloudAccountId = refs.cloudAccountId.value;
    if (!cloudAccountId) {
      showMessage("请先选择云账号。", "error");
      return;
    }
    const result = await apiFetch(`/api/cloud-accounts/${cloudAccountId}/cost-sync`, "POST", {});
    await loadData(formatCostSyncMessage(result), "success");
  });

  refs.cloudAccountForm.addEventListener("submit", async (event) => {
    event.preventDefault();
    refs.cloudAccountSyncCron.value = syncScheduleValueFromControls();
    const payload = {
      platform_code: refs.cloudAccountPlatformCode.value,
      name: refs.cloudAccountName.value,
      account_id: refs.cloudAccountAccountId.value,
      default_region: refs.cloudAccountDefaultRegion.value,
      environment: refs.cloudAccountEnvironment.value,
      owner: refs.cloudAccountOwner.value,
      criticality: refs.cloudAccountCriticality.value,
      access_key_id: refs.cloudAccountAccessKeyId.value,
      secret_access_key: refs.cloudAccountSecretAccessKey.value,
      sync_enabled: refs.cloudAccountSyncEnabled.checked,
      sync_mode: refs.cloudAccountSyncMode.value,
      sync_cron: refs.cloudAccountSyncCron.value
    };

    const id = refs.cloudAccountId.value;
    const method = id ? "PUT" : "POST";
    const url = id ? `/api/cloud-accounts/${id}` : "/api/cloud-accounts";
    const result = await apiFetch(url, method, payload);
    state.selectedCloudAccountId = result.id;
    state.detailMode = "cloudAccount";
    refs.cloudAccountAccessKeyId.value = "";
    refs.cloudAccountSecretAccessKey.value = "";
    await loadData(id ? "云账号已更新。" : "云账号已创建。", "success");
  });

  refs.cloudAccountSyncMode.addEventListener("change", () => {
    if (refs.cloudAccountSyncMode.value !== "manual") {
      refs.cloudAccountSyncEnabled.checked = true;
    }
    updateSyncControlVisibility();
  });
  refs.cloudAccountSyncEnabled.addEventListener("change", () => {
    if (refs.cloudAccountSyncEnabled.checked && refs.cloudAccountSyncMode.value === "manual") {
      refs.cloudAccountSyncMode.value = "scheduled";
    }
    updateSyncControlVisibility();
  });

  refs.newAssetButton.addEventListener("click", () => {
    state.selectedAssetId = "";
    state.detailMode = "asset";
    resetAssetForm();
    state.page = "config";
    render();
  });

  refs.exportAssetsButton.addEventListener("click", async () => {
    const result = await apiFetch("/api/assets/export", "GET");
    downloadJSON(`opsledger-assets-${new Date().toISOString().slice(0, 10)}.json`, result);
    showMessage(`已导出 ${result.assets?.length || 0} 条资产。`, "success");
  });

  refs.importAssetsButton.addEventListener("click", () => {
    refs.assetImportFile.value = "";
    refs.assetImportFile.click();
  });

  refs.assetImportFile.addEventListener("change", async () => {
    const file = refs.assetImportFile.files && refs.assetImportFile.files[0];
    if (!file) {
      return;
    }
    const text = await file.text();
    let payload;
    try {
      payload = JSON.parse(text);
    } catch (error) {
      showMessage("导入文件不是有效 JSON。", "error");
      return;
    }
    if (Array.isArray(payload)) {
      payload = {assets: payload};
    }
    const result = await apiFetch("/api/assets/import", "POST", payload);
    await loadData(`导入完成：新增 ${result.imported_assets || 0} 条，跳过 ${result.skipped_assets || 0} 条。`, result.imported_assets ? "success" : "error");
  });

  refs.newToolButton.addEventListener("click", () => {
    state.selectedToolId = "";
    resetToolForm();
    state.page = "config";
    render();
  });

  refs.toolForm.addEventListener("submit", async (event) => {
    event.preventDefault();
    const payload = {
      environment: refs.toolEnvironment.value,
      tool_type: refs.toolType.value,
      name: refs.toolName.value,
      endpoint: refs.toolEndpoint.value,
      owner: refs.toolOwner.value,
      status: refs.toolStatus.value,
      criticality: refs.toolCriticality.value,
      login_policy: refs.toolLoginPolicy.value,
      credential_policy: refs.toolCredentialPolicy.value,
      tags: refs.toolTags.value.split(",").map((item) => item.trim()).filter(Boolean),
      approval_required: refs.toolApprovalRequired.checked,
      webssh_enabled: false,
      description: refs.toolDescription.value
    };
    const id = refs.toolId.value;
    const method = id ? "PUT" : "POST";
    const url = id ? `/api/tools/${id}` : "/api/tools";
    const result = await apiFetch(url, method, payload);
    state.selectedToolId = result.id;
    state.developerEnvironment = result.environment || state.developerEnvironment;
    await loadData(id ? "工具资产已更新。" : "工具资产已创建。", "success");
  });

  refs.newCredentialButton.addEventListener("click", () => {
    state.selectedCredentialId = "";
    resetCredentialForm();
    renderCredentialOptions();
  });

  refs.credentialOwnerType.addEventListener("change", () => {
    refs.credentialOwnerId.value = "";
    renderCredentialOptions();
  });

  refs.credentialForm.addEventListener("submit", async (event) => {
    event.preventDefault();
    const payload = {
      owner_type: refs.credentialOwnerType.value,
      owner_id: refs.credentialOwnerId.value,
      kind: refs.credentialKind.value,
      key_name: refs.credentialKeyName.value || "default",
      value: refs.credentialValue.value,
      environment: refs.credentialEnvironment.value,
      project_code: refs.credentialProjectCode.value,
      access_policy: refs.credentialAccessPolicy.value,
      status: refs.credentialStatus.value
    };
    const result = await apiFetch("/api/credentials", "POST", payload);
    state.selectedCredentialId = result.id;
    refs.credentialValue.value = "";
    await loadData("凭证已保存，明文未回填。", "success");
  });

  refs.newUserButton.addEventListener("click", () => {
    state.selectedUserId = "";
    resetUserForm();
    state.page = "config";
    render();
  });

  refs.userForm.addEventListener("submit", async (event) => {
    event.preventDefault();
    restoreI18nFormValues(refs.userForm);
    const payload = {
      username: refs.userUsername.value,
      display_name: refs.userDisplayName.value,
      email: refs.userEmail.value,
      phone: refs.userPhone.value,
      role: refs.userRole.value,
      team: refs.userTeam.value,
      password: refs.userPassword.value,
      status: refs.userStatus.value
    };
    const id = refs.userId.value;
    const method = id ? "PUT" : "POST";
    const url = id ? `/api/users/${id}` : "/api/users";
    const result = await apiFetch(url, method, payload);
    state.selectedUserId = result.id;
    await loadData(id ? "用户已更新。" : "用户已创建。", "success");
  });

  refs.newRoleButton.addEventListener("click", () => {
    state.selectedRoleId = "";
    resetRoleForm();
    renderPermissions();
  });

  refs.roleForm.addEventListener("submit", async (event) => {
    event.preventDefault();
    restoreI18nFormValues(refs.roleForm);
    const payload = {
      code: refs.roleCode.value,
      name: refs.roleName.value,
      description: refs.roleDescription.value,
      level: Number(refs.roleLevel.value || 100),
      status: refs.roleStatus.value
    };
    const id = refs.roleId.value;
    const method = id ? "PUT" : "POST";
    const url = id ? `/api/roles/${id}` : "/api/roles";
    const result = await apiFetch(url, method, payload);
    state.selectedRoleId = result.id;
    await loadData(id ? "角色已更新。" : "角色已创建。", "success");
  });

  refs.newPermissionButton.addEventListener("click", () => {
    state.selectedPermissionId = "";
    resetPermissionForm();
    renderPermissions();
  });

  refs.permissionForm.addEventListener("submit", async (event) => {
    event.preventDefault();
    const payload = {
      role: refs.permissionRole.value,
      scope: refs.permissionScope.value,
      action: refs.permissionAction.value,
      environment: refs.permissionEnvironment.value,
      project_code: refs.permissionProjectCode.value,
      requires_approval: refs.permissionRequiresApproval.checked
    };
    const id = refs.permissionId.value;
    const method = id ? "PUT" : "POST";
    const url = id ? `/api/permissions/${id}` : "/api/permissions";
    const result = await apiFetch(url, method, payload);
    state.selectedPermissionId = result.id;
    await loadData(id ? "权限策略已更新。" : "权限策略已创建。", "success");
  });

  refs.addApprovalStepButton.addEventListener("click", () => {
    state.approvalFlowSteps.push({
      approver_role: "ops",
      approver_label: "Ops Engineer",
      required_action: "approved",
      timeout_minutes: 60
    });
    renderApprovalFlowSteps();
  });

  refs.newApprovalFlowButton.addEventListener("click", () => {
    state.selectedApprovalFlowId = "";
    resetApprovalFlowForm();
    renderApprovalFlows();
  });

  refs.saveApprovalFlowButton.addEventListener("click", async () => {
    restoreI18nFormValues(refs.approvalFlowForm);
    restoreI18nFormValues(refs.approvalFlowSteps);
    const payload = {
      name: refs.approvalFlowName.value,
      scope: refs.approvalFlowScope.value,
      environment: refs.approvalFlowEnvironment.value,
      status: refs.approvalFlowStatus.value,
      description: refs.approvalFlowDescription.value,
      steps: state.approvalFlowSteps.map((step) => ({
        ...step,
        approver_label: restoreTranslatedUIText(step.approver_label)
      }))
    };
    const id = refs.approvalFlowId.value;
    const method = id ? "PUT" : "POST";
    const url = id ? `/api/approval-flows/${id}` : "/api/approval-flows";
    const result = await apiFetch(url, method, payload);
    state.selectedApprovalFlowId = result.id;
    await loadData(id ? "审批流程已更新。" : "审批流程已创建。", "success");
  });

  refs.approvalForm.addEventListener("submit", async (event) => {
    event.preventDefault();
    const selectedOption = refs.approvalTargetId.selectedOptions[0];
    await apiFetch("/api/approvals", "POST", {
      requester: state.currentUser ? state.currentUser.username : refs.approvalRequester.value,
      request_type: refs.approvalRequestType.value,
      target_type: selectedOption ? selectedOption.dataset.targetType || "asset" : "asset",
      target_id: refs.approvalTargetId.value,
      environment: refs.approvalEnvironment.value,
      reason: refs.approvalReason.value,
      permission_level: refs.approvalPermissionLevel.value,
      duration_minutes: Number(refs.approvalDurationMinutes.value || 30)
    });
    refs.approvalReason.value = "";
    closeApprovalModal();
    await loadData("审批申请已提交。", "success");
  });

  refs.inspectionForm.addEventListener("submit", async (event) => {
    event.preventDefault();
    if (!refs.inspectionAssetId.value) {
      showMessage("请先选择一个资产。", "error");
      return;
    }
    await apiFetch("/api/inspections", "POST", {
      asset_id: refs.inspectionAssetId.value,
      executor: refs.inspectionExecutor.value,
      result: refs.inspectionResult.value,
      summary: refs.inspectionSummary.value,
      checked_at: refs.inspectionCheckedAt.value
    });
    state.assetDetailTab = "inspections";
    state.detailMode = "asset";
    await loadData("巡检记录已创建。", "success");
  });

  refs.newChangeButton.addEventListener("click", () => {
    state.selectedChangeId = "";
    resetChangeForm();
    state.page = "config";
    render();
  });

  refs.assetForm.addEventListener("submit", async (event) => {
    event.preventDefault();
    const payload = {
      platform_code: refs.assetPlatformCode.value,
      platform_name: refs.assetPlatformName.value,
      cloud_account_name: refs.assetCloudAccountName.value,
      account_id: refs.assetAccountId.value,
      project_code: refs.assetProjectCode.value,
      category: refs.assetCategory.value,
      resource_type: refs.assetResourceType.value,
      region: refs.assetRegion.value,
      environment: refs.assetEnvironment.value,
      name: refs.assetName.value,
      endpoint: refs.assetEndpoint.value,
      owner: refs.assetOwner.value,
      status: refs.assetStatus.value,
      criticality: refs.assetCriticality.value,
      last_checked_at: refs.assetLastCheckedAt.value,
      tags: refs.assetTags.value.split(",").map((item) => item.trim()).filter(Boolean),
      notes: refs.assetNotes.value
    };

    const id = refs.assetId.value;
    const method = id ? "PUT" : "POST";
    const url = id ? `/api/assets/${id}` : "/api/assets";
    const result = await apiFetch(url, method, payload);
    state.selectedAssetId = result.id;
    state.detailMode = "asset";
    state.selectedChangeId = "";
    await loadData(id ? "资产已更新。" : "资产已创建。", "success");
  });

  refs.assetBulkForm.addEventListener("submit", async (event) => {
    event.preventDefault();
    const assets = filterAssets();
    if (!assets.length) {
      showMessage("当前筛选条件下没有可更新资产。", "error");
      return;
    }
    const payload = buildAssetBulkPayload(assets);
    if (!hasAssetBulkPatch(payload)) {
      showMessage("请至少填写一个批量更新字段。", "error");
      return;
    }
    if (!confirm(`将批量更新当前筛选出的 ${assets.length} 条资产，继续吗？`)) {
      return;
    }
    const result = await apiFetch("/api/assets/bulk-update", "POST", payload);
    resetAssetBulkForm();
    await loadData(`批量更新完成：更新 ${result.updated_assets || 0} 条，跳过 ${result.skipped_assets || 0} 条。`, result.updated_assets ? "success" : "error");
  });

  refs.deleteAssetButton.addEventListener("click", async () => {
    if (!refs.assetId.value || !confirm("删除资产会同步删除关联变更，继续吗？")) {
      return;
    }
    await apiFetch(`/api/assets/${refs.assetId.value}`, "DELETE");
    state.selectedAssetId = "";
    state.detailMode = "asset";
    state.selectedChangeId = "";
    resetAssetForm();
    resetChangeForm();
    await loadData("资产已删除。", "success");
  });

  refs.changeForm.addEventListener("submit", async (event) => {
    event.preventDefault();
    const payload = {
      asset_id: refs.changeAssetId.value,
      title: refs.changeTitle.value,
      category: refs.changeCategory.value,
      executor: refs.changeExecutor.value,
      risk_level: refs.changeRiskLevel.value,
      status: refs.changeStatus.value,
      window: refs.changeWindow.value,
      rollback_plan: refs.changeRollbackPlan.value,
      summary: refs.changeSummary.value
    };

    const id = refs.changeId.value;
    const method = id ? "PUT" : "POST";
    const url = id ? `/api/changes/${id}` : "/api/changes";
    const result = await apiFetch(url, method, payload);
    state.selectedChangeId = result.id;
    state.selectedAssetId = result.asset_id;
    state.detailMode = "asset";
    if (!id) {
      state.assetDetailTab = "changes";
      state.pendingChangeTitlePrefix = "";
    }
    await loadData(id ? "变更已更新。" : "变更已创建。", "success");
  });

  refs.deleteChangeButton.addEventListener("click", async () => {
    if (!refs.changeId.value || !confirm("确认删除这条变更记录吗？")) {
      return;
    }
    await apiFetch(`/api/changes/${refs.changeId.value}`, "DELETE");
    state.selectedChangeId = "";
    resetChangeForm();
    await loadData("变更已删除。", "success");
  });
}

async function completeLogin(username, password) {
  const submitButton = refs.loginForm.querySelector('button[type="submit"]');
  const quickButtons = Array.from(document.querySelectorAll("[data-login-user]"));
  const setDisabled = (disabled) => {
    if (submitButton) {
      submitButton.disabled = disabled;
      submitButton.textContent = disabled ? "登录中..." : "登录";
    }
    quickButtons.forEach((button) => {
      button.disabled = disabled;
    });
  };

  try {
    setDisabled(true);
    const result = await apiFetch("/api/auth/login", "POST", {
      username,
      password
    }, {skipAuthRedirect: true});
    refs.loginPassword.value = "";
    if (result.user) {
      state.currentUser = result.user;
      state.currentPermissions = [];
      state.page = defaultPageForRole(state.currentUser.role);
      renderPage();
      renderCurrentUser();
      await loadData("登录成功。", "success");
      return;
    }
    await loadCurrentUser("登录成功。");
  } finally {
    setDisabled(false);
  }
}

async function completeSetup() {
  const submitButton = refs.setupForm.querySelector('button[type="submit"]');
  const setDisabled = (disabled) => {
    if (submitButton) {
      submitButton.disabled = disabled;
      submitButton.textContent = disabled ? "初始化中..." : "完成初始化";
    }
  };
  try {
    setDisabled(true);
    const result = await apiFetch("/api/setup", "POST", {
      username: refs.setupUsername.value,
      display_name: refs.setupDisplayName.value,
      email: refs.setupEmail.value,
      password: refs.setupPassword.value,
      confirm_password: refs.setupConfirmPassword.value
    }, {skipAuthRedirect: true});
    refs.setupPassword.value = "";
    refs.setupConfirmPassword.value = "";
    state.setup = {required: false, setup_completed: true};
    if (result.user) {
      state.currentUser = result.user;
      state.currentPermissions = [];
      state.page = defaultPageForRole(state.currentUser.role);
      renderPage();
      renderCurrentUser();
      await loadData("初始化完成。", "success");
      return;
    }
    state.page = "login";
    renderPage();
    showMessage("初始化完成，请使用管理员账号登录。", "success");
  } finally {
    setDisabled(false);
  }
}
