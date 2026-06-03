// Configuration-center rendering and governance helpers.

function renderPlatforms() {
  refs.cloudAccountPlatformCode.innerHTML = state.platforms.map((item) => `
    <option value="${escapeHTML(item.code)}">${escapeHTML(item.name)}</option>
  `).join("");

  if (!refs.cloudAccountPlatformCode.value && state.platforms[0]) {
    refs.cloudAccountPlatformCode.value = state.platforms[0].code;
  }
}

function renderEnvironments() {
  const options = allEnvironments();
  const toolOptions = [{code: "global", name: "全局工具"}, ...options];
  refs.toolEnvironment.innerHTML = toolOptions.map((item) => `
    <option value="${escapeHTML(item.code)}">${escapeHTML(item.code)} / ${escapeHTML(item.name || item.code)}</option>
  `).join("");
  if (!refs.toolEnvironment.value && toolOptions[0]) {
    refs.toolEnvironment.value = toolOptions[0].code;
  }
}

function allEnvironments() {
  return state.environments.length ? state.environments : fallbackEnvironments;
}

function developerWorkbenchEnvironments() {
  const enabled = allEnvironments().filter((item) => ["active", "guarded"].includes(item.status || "active"));
  return enabled.length ? enabled : allEnvironments();
}

function renderSectionVisibility() {
  document.querySelectorAll("[data-section-body]").forEach((node) => {
    const sectionKey = node.dataset.sectionBody;
    const collapsed = state.collapsedSections.has(sectionKey);
    node.classList.toggle("hidden", collapsed);
  });

  updateSectionToggleLabel(refs.toggleCloudAccountSectionButton, "cloudAccount");
  updateSectionToggleLabel(refs.toggleAssetSectionButton, "asset");
  updateSectionToggleLabel(refs.toggleToolSectionButton, "tool");
  updateSectionToggleLabel(refs.toggleUserSectionButton, "user");
  updateSectionToggleLabel(refs.toggleChangeSectionButton, "change");
  updateSectionToggleLabel(refs.toggleInspectionSectionButton, "inspection");
  applyI18n();
}

function renderConfigDomain() {
  if (!refs.configDomainTabs) {
    return;
  }
  const activeDomain = state.configDomain || "cloud";
  refs.configDomainTabs.querySelectorAll("[data-config-domain]").forEach((button) => {
    button.classList.toggle("active", button.dataset.configDomain === activeDomain);
  });
  document.querySelectorAll("[data-config-domains]").forEach((node) => {
    const domains = String(node.dataset.configDomains || "").split(/\s+/).filter(Boolean);
    node.classList.toggle("hidden", !domains.includes(activeDomain));
  });
  applyI18n();
}

function renderCloudAccounts() {
  if (!state.cloudAccounts.length) {
    refs.cloudAccountsList.innerHTML = `<div class="empty">当前还没有云账号记录。</div>`;
    return;
  }

  refs.cloudAccountsList.innerHTML = state.cloudAccounts.map((item) => `
    <article class="change-card config-account-card ${item.id === state.selectedCloudAccountId ? "active" : ""}" data-cloud-account-id="${item.id}">
      <div class="section-head">
        <p class="change-title">${escapeHTML(item.name)}</p>
        <span class="status-badge status-${item.sync_enabled ? "done" : "planned"}">${escapeHTML(item.platform_name)}</span>
      </div>
      <span class="change-meta">${escapeHTML(item.account_id || "-")} / ${escapeHTML(item.environment)} / ${escapeHTML(item.owner || "-")}</span>
      <div class="change-meta">
        <span class="pill">${escapeHTML(item.access_key_id_masked || "未配置凭证")}</span>
        <span class="pill">${escapeHTML(item.default_region || "region optional")}</span>
        <span class="pill">${escapeHTML(renderSyncPolicy(item))}</span>
        <span class="pill">${escapeHTML(renderNextSyncHint(item))}</span>
      </div>
      <p class="change-summary">配置索引：详情、费用、巡检和资产分布请在资产树点击云账号查看。</p>
    </article>
  `).join("");

  refs.cloudAccountsList.querySelectorAll("[data-cloud-account-id]").forEach((card) => {
    card.addEventListener("click", () => {
      state.selectedCloudAccountId = card.dataset.cloudAccountId;
      syncFormsFromSelection();
      state.page = "config";
      showMessage(`已回填云账号 ${state.cloudAccounts.find((item) => item.id === state.selectedCloudAccountId)?.name || ""} 的配置表单。`, "success");
      render();
    });
  });
}

function renderCostMetric(label, value, currency, trend = "") {
  return `
    <span class="cost-metric ${trend}">
      <small>${escapeHTML(label)}</small>
      <strong>${escapeHTML(value ? `${currency || ""} ${value}`.trim() : "-")}</strong>
    </span>
  `;
}

function renderWorkbenchApprovals() {
  const pendingApprovals = state.approvals.filter((item) => item.status === "pending" && canDecideApproval(item));
  const visible = state.page === "workbench" && canDecideApprovals() && pendingApprovals.length > 0;
  refs.workbenchApprovalInboxButton.classList.toggle("hidden", !visible);
  refs.workbenchApprovalInboxButton.textContent = visible ? `有 ${pendingApprovals.length} 条审批待办，点击处理` : "";
}

function formatCostSyncMessage(result) {
  return `费用同步完成：上月整月 ${result.currency} ${result.last_month_cost}，上月同进度 ${result.currency} ${result.last_month_to_date_cost}，本月 ${result.currency} ${result.current_month_cost}，预计 ${result.currency} ${result.forecast_month_cost}。`;
}

function formatSyncMessage(result) {
  const warnings = (result.warnings || []).length;
  return `${formatSyncRecordSummary(result)}${warnings ? `，警告 ${warnings} 条` : ""}。`;
}

function formatSyncRecordSummary(sync) {
  const stale = Number(sync.stale_assets || 0);
  return `发现 ${sync.discovered_assets || 0} 条，新增 ${sync.created_assets || 0} 条，更新 ${sync.updated_assets || 0} 条${stale ? `，标记 stale ${stale} 条` : ""}`;
}

function renderSyncPolicy(account) {
  if (!account.sync_enabled || (account.sync_mode || "manual") === "manual") {
    return "manual";
  }
  return `${account.sync_mode || "auto"} ${account.sync_cron || "6h"}`.trim();
}

function renderNextSyncHint(account) {
  if (!account.sync_enabled || (account.sync_mode || "manual") === "manual") {
    return "未启用自动同步";
  }
  const intervalMs = syncIntervalMs(account.sync_mode, account.sync_cron);
  if (!intervalMs) {
    return currentLanguage() === "en" ? "Sync schedule pending" : "同步时间待确认";
  }
  if (!account.last_sync_at) {
    return "等待调度";
  }
  const last = Date.parse(account.last_sync_at);
  if (Number.isNaN(last)) {
    return "等待调度";
  }
  const nextAt = new Date(last + intervalMs);
  if (nextAt.getTime() <= Date.now()) {
    return "即将同步";
  }
  return formatDateTime(nextAt.toISOString());
}

function syncIntervalMs(mode, expr) {
  mode = String(mode || "manual").toLowerCase();
  expr = String(expr || "").trim();
  if (mode === "manual") return 0;
  const durationMatch = expr.match(/^(\d+)(ms|s|m|h)$/);
  if (durationMatch) {
    const value = Number(durationMatch[1]);
    const unit = durationMatch[2];
    return value * ({ms: 1, s: 1000, m: 60000, h: 3600000}[unit] || 0);
  }
  const composed = composedIntervalMs(expr);
  if (composed) return composed;
  const fields = expr.split(/\s+/).filter(Boolean);
  if (fields.length === 5) {
    const [minute, hour] = fields;
    if (hour.startsWith("*/")) return Number(hour.slice(2)) * 3600000;
    if (minute.startsWith("*/")) return Number(minute.slice(2)) * 60000;
    if (hour === "*" && minute !== "*") return 3600000;
    if (hour === "*" && minute === "*") return 60000;
    return 24 * 3600000;
  }
  if (/^\d{1,2}:\d{2}$/.test(expr)) return 24 * 3600000;
  return ["interval", "auto", "scheduled", "cron"].includes(mode) ? 6 * 3600000 : 0;
}

function composedIntervalMs(expr) {
  const units = [
    ["mo", 30 * 24 * 3600000],
    ["y", 365 * 24 * 3600000],
    ["w", 7 * 24 * 3600000],
    ["d", 24 * 3600000],
    ["h", 3600000],
    ["m", 60000],
    ["s", 1000]
  ];
  let remaining = String(expr || "").trim();
  let total = 0;
  if (!remaining) return 0;
  while (remaining) {
    let matched = false;
    for (const [suffix, multiplier] of units) {
      const index = remaining.indexOf(suffix);
      if (index <= 0) continue;
      const value = Number(remaining.slice(0, index));
      if (!Number.isFinite(value) || value <= 0) return 0;
      total += value * multiplier;
      remaining = remaining.slice(index + suffix.length);
      matched = true;
      break;
    }
    if (!matched) return 0;
  }
  return total;
}

function renderTools() {
  if (!state.tools.length) {
    refs.toolsList.innerHTML = `<div class="empty">当前还没有工具资产。</div>`;
    return;
  }
  refs.toolsList.innerHTML = state.tools.map((tool) => `
    <article class="change-card ${tool.id === state.selectedToolId ? "active" : ""}" data-tool-id="${tool.id}">
      <div class="section-head">
        <p class="change-title">${escapeHTML(tool.asset_name)}</p>
        <span class="status-badge status-${escapeHTML(tool.status)}">${escapeHTML(renderToolScope(tool.environment))}</span>
      </div>
      <span class="change-meta">${escapeHTML(tool.tool_type)} / ${escapeHTML(tool.owner || "-")}</span>
      <div class="change-meta">
        <span class="pill">${escapeHTML(tool.login_policy || "sso")}</span>
        <span class="pill">${escapeHTML(tool.credential_policy || "none")}</span>
        ${tool.approval_required ? `<span class="pill">审批</span>` : ""}
      </div>
    </article>
  `).join("");
  refs.toolsList.querySelectorAll("[data-tool-id]").forEach((card) => {
    card.addEventListener("click", () => {
      state.selectedToolId = card.dataset.toolId;
      syncFormsFromSelection();
      state.page = "config";
      render();
    });
  });
}

function renderCredentialOptions() {
  const ownerType = refs.credentialOwnerType.value || "asset";
  const owners = ownerType === "cloud_account"
    ? state.cloudAccounts.map((item) => ({
      id: item.id,
      label: `${item.name} / ${item.platform_name || item.platform_code} / ${item.environment}`,
      environment: item.environment,
      project: "cloud"
    }))
    : [
      ...state.tools.map((tool) => ({
        id: tool.asset_id,
        label: `${tool.asset_name} / ${renderToolScope(tool.environment)} / ${tool.tool_type}`,
        environment: tool.environment,
        project: "public"
      })),
      ...state.assets
        .filter((asset) => asset.category !== "tool")
        .slice(0, 120)
        .map((asset) => ({
          id: asset.id,
          label: `${asset.name} / ${asset.resource_type} / ${asset.environment}`,
          environment: asset.environment,
          project: asset.project_code || "public"
        }))
    ];
  refs.credentialOwnerId.innerHTML = owners.map((owner) => `
    <option value="${escapeHTML(owner.id)}" data-env="${escapeHTML(owner.environment || "")}" data-project="${escapeHTML(owner.project || "")}">${escapeHTML(owner.label)}</option>
  `).join("");
  if (!refs.credentialOwnerId.value && owners[0]) {
    refs.credentialOwnerId.value = owners[0].id;
  }
  const selected = owners.find((owner) => owner.id === refs.credentialOwnerId.value);
  if (selected && !refs.credentialEnvironment.value) {
    refs.credentialEnvironment.value = selected.environment || "";
  }
  if (selected && !refs.credentialProjectCode.value) {
    refs.credentialProjectCode.value = selected.project || "";
  }
}

function renderCredentials() {
  if (!refs.credentialsList) return;
  if (!state.credentials.length) {
    refs.credentialsList.innerHTML = `<div class="empty">当前还没有凭证项。</div>`;
    return;
  }
  refs.credentialsList.innerHTML = state.credentials.map((item) => `
    <article class="change-card credential-card ${item.id === state.selectedCredentialId ? "active" : ""}" data-credential-id="${escapeHTML(item.id)}">
      <div class="section-head">
        <p class="change-title">${escapeHTML(item.owner_name || item.owner_id)}</p>
        <span class="status-badge status-${escapeHTML(item.status)}">${escapeHTML(item.kind)}</span>
      </div>
      <span class="change-meta">${escapeHTML(item.owner_type)} / ${escapeHTML(item.environment || "-")} / ${escapeHTML(item.project_code || "-")}</span>
      <div class="change-meta">
        <span class="pill">${escapeHTML(item.key_name || "default")}</span>
        <span class="pill">${escapeHTML(item.masked_value || "******")}</span>
        <span class="pill">${escapeHTML(item.access_policy || "ops_only")}</span>
      </div>
      <div class="credential-actions">
        <button type="button" data-reveal-credential="${escapeHTML(item.id)}">查看</button>
        <button type="button" data-copy-credential="${escapeHTML(item.id)}">复制</button>
      </div>
    </article>
  `).join("");
  refs.credentialsList.querySelectorAll("[data-credential-id]").forEach((card) => {
    card.addEventListener("click", (event) => {
      if (event.target.closest("button")) return;
      selectCredential(card.dataset.credentialId);
    });
  });
  refs.credentialsList.querySelectorAll("[data-reveal-credential]").forEach((button) => {
    button.addEventListener("click", async () => {
      const result = await apiFetch(`/api/credentials/${button.dataset.revealCredential}/reveal`, "POST", {});
      state.selectedCredentialId = result.credential.id;
      showMessage(`凭证明文：${result.value}`, "success");
      await loadData("", "success");
    });
  });
  refs.credentialsList.querySelectorAll("[data-copy-credential]").forEach((button) => {
    button.addEventListener("click", async () => {
      const result = await apiFetch(`/api/credentials/${button.dataset.copyCredential}/reveal`, "POST", {});
      await navigator.clipboard.writeText(result.value);
      await apiFetch(`/api/credentials/${button.dataset.copyCredential}/copy`, "POST", {});
      state.selectedCredentialId = result.credential.id;
      await loadData("凭证已复制，并已写入审计。", "success");
    });
  });
}

function selectCredential(id) {
  const item = state.credentials.find((credential) => credential.id === id);
  if (!item) return;
  state.selectedCredentialId = id;
  state.page = "config";
  render();
  fillCredentialForm(item);
}

function fillCredentialForm(item) {
  refs.credentialOwnerType.value = item.owner_type === "cloud_account" ? "cloud_account" : "asset";
  renderCredentialOptions();
  refs.credentialOwnerId.value = item.owner_id || "";
  refs.credentialKind.value = item.kind || "password";
  refs.credentialKeyName.value = item.key_name || "default";
  refs.credentialEnvironment.value = item.environment || "";
  refs.credentialProjectCode.value = item.project_code || "";
  refs.credentialAccessPolicy.value = item.access_policy || "ops_only";
  refs.credentialStatus.value = item.status || "active";
  refs.credentialValue.value = "";
}

function renderTagGovernance() {
  if (!refs.tagGovernanceSummary || !refs.tagGovernanceList) {
    return;
  }
  const issues = buildTagGovernanceIssues();
  const totals = {
    missingProject: issues.filter((item) => item.type === "missing-project").length,
    missingEnv: issues.filter((item) => item.type === "missing-env").length,
    emptyValue: issues.filter((item) => item.type === "empty-value").length,
    invalidValue: issues.filter((item) => item.type === "invalid-value").length,
    unknownEnv: issues.filter((item) => item.type === "unknown-env").length
  };
  refs.tagGovernanceSummary.innerHTML = `
    ${renderTagGovernanceMetric("缺 Project", totals.missingProject)}
    ${renderTagGovernanceMetric("缺 Environment", totals.missingEnv)}
    ${renderTagGovernanceMetric("空标签值", totals.emptyValue)}
    ${renderTagGovernanceMetric("值不合规", totals.invalidValue)}
    ${renderTagGovernanceMetric("环境 unknown", totals.unknownEnv)}
  `;
  if (!issues.length) {
    refs.tagGovernanceList.innerHTML = `<div class="empty">当前资产标签治理项为空。重新同步 AWS 后可在这里复核真实 tag 值。</div>`;
    return;
  }
  refs.tagGovernanceList.innerHTML = issues.slice(0, 80).map((issue) => `
    <article class="change-card tag-governance-card" data-governance-asset-id="${escapeHTML(issue.asset.id)}">
      <div class="section-head">
        <p class="change-title">${escapeHTML(issue.asset.name)}</p>
        <span class="status-badge status-${issue.level === "danger" ? "cancelled" : "in_progress"}">${escapeHTML(issue.label)}</span>
      </div>
      <span class="change-meta">${escapeHTML(projectLabel(issue.asset.project_code))} / ${escapeHTML(issue.asset.cloud_account_name || "-")} / ${escapeHTML(issue.asset.resource_type || "-")} / ${escapeHTML(issue.asset.environment || "-")}</span>
      <p class="change-summary">${escapeHTML(issue.message)}</p>
      <div class="tag-chip-row">${renderAssetTagChips(issue.asset.tags || [])}</div>
    </article>
  `).join("");
  refs.tagGovernanceList.querySelectorAll("[data-governance-asset-id]").forEach((card) => {
    card.addEventListener("click", () => {
      state.selectedAssetId = card.dataset.governanceAssetId;
      state.detailMode = "asset";
      state.page = "workbench";
      render();
    });
  });
}

function renderTagGovernanceMetric(label, value) {
  return `
    <span class="tag-governance-metric ${value ? "warn" : "ok"}">
      <small>${escapeHTML(label)}</small>
      <strong>${escapeHTML(String(value))}</strong>
    </span>
  `;
}

function buildTagGovernanceIssues() {
  const issues = [];
  (state.assets || []).forEach((asset) => {
    if (asset.category === "tool") {
      return;
    }
    const tagMap = assetTagMap(asset.tags || []);
    const projectValue = firstTagValue(tagMap, ["project", "projectcode", "project-code", "biz", "business", "system", "app"]);
    const envValue = firstTagValue(tagMap, ["environment", "env", "stage", "profile"]);
    if (!projectValue) {
      issues.push(tagGovernanceIssue(asset, "missing-project", "缺 Project", "未发现 Project/ProjectCode/Biz/System/App 标签，项目归属只能依赖台账推断。", "warn"));
    }
    if (!envValue) {
      issues.push(tagGovernanceIssue(asset, "missing-env", "缺 Environment", "未发现 Environment/Env/Stage/Profile 标签，环境归属只能依赖台账或名称推断。", "warn"));
    }
    if (String(asset.environment || "").toLowerCase() === "unknown") {
      issues.push(tagGovernanceIssue(asset, "unknown-env", "环境 unknown", "混合账号资源未能从标签或名称判断环境，需要补 Environment 标签。", "danger"));
    }
    for (const [key, value] of tagMap.entries()) {
      if (!value) {
        issues.push(tagGovernanceIssue(asset, "empty-value", "空标签值", `标签 ${key} 没有值。`, "warn"));
      } else if (!isValidGovernanceTagValue(value)) {
        issues.push(tagGovernanceIssue(asset, "invalid-value", "值不合规", `标签 ${key}=${value} 包含不建议的字符。`, "warn"));
      }
    }
  });
  return issues.sort((a, b) => {
    const levelOrder = {danger: 0, warn: 1};
    return (levelOrder[a.level] ?? 9) - (levelOrder[b.level] ?? 9)
      || a.asset.cloud_account_name.localeCompare(b.asset.cloud_account_name)
      || a.asset.resource_type.localeCompare(b.asset.resource_type)
      || a.asset.name.localeCompare(b.asset.name);
  });
}

function tagGovernanceIssue(asset, type, label, message, level) {
  return {asset, type, label, message, level};
}

function assetTagMap(tags) {
  const result = new Map();
  tags.forEach((tag) => {
    const text = String(tag || "").trim();
    if (!text.toLowerCase().startsWith("tag:")) {
      return;
    }
    const body = text.slice(4);
    const splitAt = body.indexOf("=");
    const key = splitAt >= 0 ? body.slice(0, splitAt).trim() : body.trim();
    const value = splitAt >= 0 ? body.slice(splitAt + 1).trim() : "";
    if (key) {
      result.set(key, value);
    }
  });
  return result;
}

function firstTagValue(tagMap, keys) {
  const wanted = new Set(keys.map(normalizeGovernanceTagKey));
  for (const [key, value] of tagMap.entries()) {
    if (wanted.has(normalizeGovernanceTagKey(key))) {
      return value || "";
    }
  }
  return "";
}

function normalizeGovernanceTagKey(value) {
  return String(value || "").toLowerCase().replaceAll(/[_\-\s./]/g, "");
}

function isValidGovernanceTagValue(value) {
  return /^[\p{L}\p{N}_.:/@+=,\-\s]+$/u.test(String(value || ""));
}

function renderAssetTagChips(tags) {
  const values = (tags || []).filter(Boolean).slice(0, 8);
  if (!values.length) {
    return `<span class="pill">无标签</span>`;
  }
  return values.map((tag) => `<span class="pill">${escapeHTML(tag)}</span>`).join("");
}

function renderChargeback() {
  if (!refs.chargebackSummary || !refs.chargebackList) {
    return;
  }
  const rows = buildChargebackRows();
  const totalCurrent = rows.reduce((sum, row) => sum + row.currentCost, 0);
  const totalForecast = rows.reduce((sum, row) => sum + row.forecastCost, 0);
  const totalAssets = rows.reduce((sum, row) => sum + row.assetCount, 0);
  refs.chargebackSummary.innerHTML = `
    ${renderTagGovernanceMetric("项目数", rows.length)}
    ${renderTagGovernanceMetric("纳入资产", totalAssets)}
    ${renderTagGovernanceMetric("本月项目估算", formatMoneyValue(totalCurrent))}
    ${renderTagGovernanceMetric("预计项目估算", formatMoneyValue(totalForecast))}
    ${renderTagGovernanceMetric("费用账号", chargebackAccountCount())}
  `;
  if (!rows.length) {
    refs.chargebackList.innerHTML = `<div class="empty">当前没有可分摊的云账号费用。请先同步 AWS 费用和资产标签。</div>`;
    return;
  }
  refs.chargebackList.innerHTML = rows.map((row) => `
    <article class="change-card chargeback-card">
      <div class="section-head">
        <p class="change-title">${escapeHTML(projectLabel(row.project))}</p>
        <span class="status-badge status-done">${escapeHTML(row.currency || "USD")}</span>
      </div>
      <span class="change-meta">资产 ${row.assetCount} / 账号 ${row.accountCount} / 占比 ${escapeHTML(formatPercent(row.weight))}</span>
      <div class="cost-strip">
        ${renderCostMetric("本月当前", formatMoneyValue(row.currentCost), row.currency)}
        ${renderCostMetric("预计本月", formatMoneyValue(row.forecastCost), row.currency)}
        ${renderCostMetric("上月整月", formatMoneyValue(row.lastMonthCost), row.currency)}
        ${renderCostMetric("上月同进度", formatMoneyValue(row.lastMonthToDateCost), row.currency)}
      </div>
      <p class="change-summary">项目估算口径：按每个云账号内项目资产数量占比分摊。账号详情中的服务维度费用来自 AWS Cost Explorer 真实账单；项目精确分账后续接 Cost Allocation Tag / CUR。</p>
    </article>
  `).join("");
}

function buildChargebackRows() {
  const accountAssets = new Map();
  (state.assets || []).forEach((asset) => {
    if (!asset.cloud_account_id || asset.status === "stale" || asset.category === "tool") {
      return;
    }
    if (!accountAssets.has(asset.cloud_account_id)) {
      accountAssets.set(asset.cloud_account_id, []);
    }
    accountAssets.get(asset.cloud_account_id).push(asset);
  });

  const rows = new Map();
  (state.cloudAccounts || []).forEach((account) => {
    const assets = accountAssets.get(account.id) || [];
    const currentCost = parseCostNumber(account.current_month_cost);
    const forecastCost = parseCostNumber(account.forecast_month_cost);
    const lastMonthCost = parseCostNumber(account.last_month_cost);
    const lastMonthToDateCost = parseCostNumber(account.last_month_to_date_cost);
    if (!assets.length || (currentCost === 0 && forecastCost === 0 && lastMonthCost === 0 && lastMonthToDateCost === 0)) {
      return;
    }
    const projectCounts = new Map();
    assets.forEach((asset) => {
      const project = asset.project_code || "public";
      projectCounts.set(project, (projectCounts.get(project) || 0) + 1);
    });
    for (const [project, count] of projectCounts.entries()) {
      const weight = count / assets.length;
      const row = rows.get(project) || {
        project,
        currency: account.cost_currency || "USD",
        assetCount: 0,
        accountCount: 0,
        accountIDs: new Set(),
        currentCost: 0,
        forecastCost: 0,
        lastMonthCost: 0,
        lastMonthToDateCost: 0,
        weight: 0
      };
      row.assetCount += count;
      row.accountIDs.add(account.id);
      row.accountCount = row.accountIDs.size;
      row.currentCost += currentCost * weight;
      row.forecastCost += forecastCost * weight;
      row.lastMonthCost += lastMonthCost * weight;
      row.lastMonthToDateCost += lastMonthToDateCost * weight;
      row.weight += weight;
      rows.set(project, row);
    }
  });
  return Array.from(rows.values()).sort((a, b) => b.currentCost - a.currentCost || projectLabel(a.project).localeCompare(projectLabel(b.project)));
}

function chargebackAccountCount() {
  return (state.cloudAccounts || []).filter((account) => parseCostNumber(account.current_month_cost) > 0 || parseCostNumber(account.forecast_month_cost) > 0).length;
}

function parseCostNumber(value) {
  const number = Number(String(value || "").replaceAll(",", ""));
  return Number.isFinite(number) ? number : 0;
}

function formatMoneyValue(value) {
  const number = Number(value || 0);
  return number.toLocaleString("en-US", {minimumFractionDigits: 2, maximumFractionDigits: 2});
}

function formatPercent(value) {
  const number = Number(value || 0);
  return `${Math.round(number * 100)}%`;
}

function renderUsers() {
  if (!state.users.length) {
    refs.usersList.innerHTML = `<div class="empty">当前还没有用户。</div>`;
    return;
  }
  refs.usersList.innerHTML = state.users.map((user) => `
    <article class="change-card ${user.id === state.selectedUserId ? "active" : ""}" data-user-id="${user.id}">
      <div class="section-head">
        <p class="change-title">${escapeHTML(user.display_name)}</p>
        <span class="status-badge status-${user.status === "active" ? "done" : "cancelled"}">${escapeHTML(user.role)}</span>
      </div>
      <span class="change-meta">${escapeHTML(user.username)} / ${escapeHTML(user.team || "-")} / ${escapeHTML(user.status)}</span>
      <span class="change-meta">${escapeHTML(user.email || "未配置邮箱")} / ${escapeHTML(user.last_login_at ? `最近登录 ${formatDateTime(user.last_login_at)}` : "未登录")}</span>
    </article>
  `).join("");
  refs.usersList.querySelectorAll("[data-user-id]").forEach((card) => {
    card.addEventListener("click", () => {
      state.selectedUserId = card.dataset.userId;
      syncFormsFromSelection();
      state.page = "config";
      render();
    });
  });
}

function renderPermissions() {
  renderRoleOptions();
  renderRoles();
  syncPermissionFormFromSelection();
  syncApprovalFlowFormFromSelection();
  renderApprovalFlows();
}

function renderRoleOptions() {
  const roles = state.roles.length ? state.roles : [
    {code: "developer", name: "Developer"},
    {code: "lead", name: "Development Lead"},
    {code: "ops", name: "Ops Engineer"},
    {code: "admin", name: "Platform Admin"}
  ];
  const roleOptions = roles.map((role) => `<option value="${escapeHTML(role.code)}">${escapeHTML(role.name)} / ${escapeHTML(role.code)}</option>`).join("");
  refs.userRole.innerHTML = roleOptions;
  refs.permissionRole.innerHTML = roleOptions;

  const envOptions = [{code: "*", name: "全部环境"}, ...state.environments]
    .map((env) => `<option value="${escapeHTML(env.code)}">${escapeHTML(env.name || env.code)}</option>`).join("");
  refs.permissionEnvironment.innerHTML = envOptions;
  refs.approvalFlowEnvironment.innerHTML = envOptions;
}

function renderRoles() {
  refs.roleList.innerHTML = state.roles.length ? state.roles.map((role) => `
    <article class="permission-item role-card ${role.id === state.selectedRoleId ? "active" : ""}" data-role-id="${escapeHTML(role.id)}">
      <strong>${escapeHTML(role.name)} <span class="pill">${escapeHTML(role.code)}</span></strong>
      <span>层级 ${escapeHTML(String(role.level || 0))} / ${escapeHTML(role.status)}</span>
      <small>${escapeHTML(role.description || "-")}</small>
    </article>
  `).join("") : `<div class="empty">暂无角色定义。</div>`;
  refs.roleList.querySelectorAll("[data-role-id]").forEach((card) => {
    card.addEventListener("click", () => {
      state.selectedRoleId = card.dataset.roleId;
      syncRoleFormFromSelection();
      renderRoles();
    });
  });
  syncRoleFormFromSelection();
}

function syncRoleFormFromSelection() {
  const role = state.roles.find((item) => item.id === state.selectedRoleId);
  if (!role) {
    if (!refs.roleId.value) {
      resetRoleForm();
    }
    return;
  }
  refs.roleId.value = role.id;
  refs.roleCode.value = role.code || "";
  refs.roleName.value = role.name || "";
  refs.roleLevel.value = role.level || 100;
  refs.roleStatus.value = role.status || "active";
  refs.roleDescription.value = role.description || "";
}

function syncPermissionFormFromSelection() {
  const permission = state.permissions.find((item) => item.id === state.selectedPermissionId);
  if (!permission) {
    if (!refs.permissionId.value) {
      resetPermissionForm();
    }
  } else {
    refs.permissionId.value = permission.id;
    refs.permissionRole.value = permission.role || "developer";
    refs.permissionEnvironment.value = permission.environment || "*";
    refs.permissionProjectCode.value = permission.project_code || "*";
    refs.permissionScope.value = permission.scope || "tool";
    refs.permissionAction.value = permission.action || "view";
    refs.permissionRequiresApproval.checked = !!permission.requires_approval;
  }

  refs.permissionList.innerHTML = `
    <h3>权限策略</h3>
    <div class="permission-grid">
      ${state.permissions.map((item) => `
        <div class="permission-item ${item.id === state.selectedPermissionId ? "active" : ""}" data-permission-id="${escapeHTML(item.id)}">
          <strong>${escapeHTML(item.role)}</strong>
          <span>${escapeHTML(item.environment)} / ${escapeHTML(item.project_code || "*")} / ${escapeHTML(item.scope)} / ${escapeHTML(item.action)}</span>
          <small>${item.requires_approval ? "需要审批" : "直接允许"}</small>
        </div>
      `).join("") || `<div class="empty">暂无权限策略。</div>`}
    </div>
  `;
  refs.permissionList.querySelectorAll("[data-permission-id]").forEach((card) => {
    card.addEventListener("click", () => {
      state.selectedPermissionId = card.dataset.permissionId;
      syncPermissionFormFromSelection();
    });
  });
}

function syncApprovalFlowFormFromSelection() {
  const flow = state.approvalFlows.find((item) => item.id === state.selectedApprovalFlowId);
  if (!flow) {
    if (!refs.approvalFlowId.value) {
      resetApprovalFlowForm();
    }
    return;
  }
  refs.approvalFlowId.value = flow.id;
  refs.approvalFlowName.value = flow.name || "";
  refs.approvalFlowScope.value = flow.scope || "credential";
  refs.approvalFlowEnvironment.value = flow.environment || "*";
  refs.approvalFlowStatus.value = flow.status || "active";
  refs.approvalFlowDescription.value = flow.description || "";
  state.approvalFlowSteps = (flow.steps || []).map((step) => ({
    approver_role: step.approver_role,
    approver_label: step.approver_label,
    required_action: step.required_action,
    timeout_minutes: step.timeout_minutes
  }));
  renderApprovalFlowSteps();
}

function renderApprovalFlows() {
  refs.approvalFlowList.innerHTML = state.approvalFlows.length ? state.approvalFlows.map((flow) => `
    <article class="change-card ${flow.id === state.selectedApprovalFlowId ? "active" : ""}" data-flow-id="${escapeHTML(flow.id)}">
      <div class="section-head">
        <p class="change-title">${escapeHTML(flow.name)}</p>
        <span class="status-badge status-${flow.status === "active" ? "done" : "cancelled"}">${escapeHTML(flow.status)}</span>
      </div>
      <span class="change-meta">${escapeHTML(flow.environment)} / ${escapeHTML(flow.scope)} / ${escapeHTML(String((flow.steps || []).length))} 步</span>
      <p class="change-summary">${escapeHTML(flow.description || "-")}</p>
    </article>
  `).join("") : `<div class="empty">暂无审批流程。</div>`;
  refs.approvalFlowList.querySelectorAll("[data-flow-id]").forEach((card) => {
    card.addEventListener("click", () => {
      state.selectedApprovalFlowId = card.dataset.flowId;
      syncApprovalFlowFormFromSelection();
      renderApprovalFlows();
    });
  });
  renderApprovalFlowSteps();
}

function renderApprovalFlowSteps() {
  refs.approvalFlowSteps.innerHTML = state.approvalFlowSteps.length ? state.approvalFlowSteps.map((step, index) => `
    <div class="approval-step" draggable="true" data-step-index="${index}">
      <span class="drag-handle">${index + 1}</span>
      <label>审批角色<select data-step-field="approver_role">
        ${state.roles.map((role) => `<option value="${escapeHTML(role.code)}" ${role.code === step.approver_role ? "selected" : ""}>${escapeHTML(role.name)} / ${escapeHTML(role.code)}</option>`).join("")}
      </select></label>
      <label>显示名称<input data-step-field="approver_label" value="${escapeHTML(step.approver_label || "")}"></label>
      <label>动作<select data-step-field="required_action">
        <option value="approved" ${step.required_action === "approved" ? "selected" : ""}>approved</option>
        <option value="reviewed" ${step.required_action === "reviewed" ? "selected" : ""}>reviewed</option>
      </select></label>
      <label>超时分钟<input type="number" min="5" max="1440" data-step-field="timeout_minutes" value="${escapeHTML(String(step.timeout_minutes || 60))}"></label>
      <button type="button" data-remove-step="${index}" class="danger">删除</button>
    </div>
  `).join("") : `<div class="empty">暂无流程步骤，先新增一个审批步骤。</div>`;

  refs.approvalFlowSteps.querySelectorAll("[data-step-field]").forEach((field) => {
    field.addEventListener("input", updateApprovalStepFromField);
    field.addEventListener("change", updateApprovalStepFromField);
  });
  refs.approvalFlowSteps.querySelectorAll("[data-remove-step]").forEach((button) => {
    button.addEventListener("click", () => {
      state.approvalFlowSteps.splice(Number(button.dataset.removeStep), 1);
      renderApprovalFlowSteps();
    });
  });
  refs.approvalFlowSteps.querySelectorAll(".approval-step").forEach((step) => {
    step.addEventListener("dragstart", (event) => {
      event.dataTransfer.setData("text/plain", step.dataset.stepIndex);
    });
    step.addEventListener("dragover", (event) => event.preventDefault());
    step.addEventListener("drop", (event) => {
      event.preventDefault();
      const from = Number(event.dataTransfer.getData("text/plain"));
      const to = Number(step.dataset.stepIndex);
      if (Number.isNaN(from) || Number.isNaN(to) || from === to) {
        return;
      }
      const [moved] = state.approvalFlowSteps.splice(from, 1);
      state.approvalFlowSteps.splice(to, 0, moved);
      renderApprovalFlowSteps();
    });
  });
}

function updateApprovalStepFromField(event) {
  const wrapper = event.target.closest("[data-step-index]");
  if (!wrapper) return;
  const index = Number(wrapper.dataset.stepIndex);
  const field = event.target.dataset.stepField;
  state.approvalFlowSteps[index][field] = field === "timeout_minutes" ? Number(event.target.value || 60) : event.target.value;
}

function renderChangeOptions() {
  const current = refs.changeAssetId.value || state.selectedAssetId;
  refs.changeAssetId.innerHTML = state.assets.map((asset) => `
    <option value="${asset.id}">${escapeHTML(asset.name)} | ${escapeHTML(asset.platform_name || asset.platform_code)} | ${escapeHTML(asset.environment)}</option>
  `).join("");

  if (current && state.assets.find((item) => item.id === current)) {
    refs.changeAssetId.value = current;
  } else if (state.assets[0]) {
    refs.changeAssetId.value = state.assets[0].id;
  }
}

function renderInspectionConfigState() {
  if (!refs.inspectionForm) {
    return;
  }
  const selectedAsset = state.assets.find((item) => item.id === state.selectedAssetId);
  refs.inspectionAssetId.value = selectedAsset ? selectedAsset.id : "";
  const fields = refs.inspectionForm.querySelectorAll("input, select, textarea, button[type='submit']");
  fields.forEach((field) => {
    field.disabled = true;
  });
  refs.inspectionAssetName.value = selectedAsset ? selectedAsset.name : "";
  refs.inspectionExecutor.value = "auto-inspection";
  refs.inspectionResult.value = selectedAsset && selectedAsset.status !== "active" ? "warning" : "ok";
  refs.inspectionCheckedAt.value = new Date().toISOString().slice(0, 10);
  refs.inspectionSummary.value = "巡检由系统自动拨测、同步检查和资产健康规则生成，配置页不再手工录入。";
}

function renderAlerts() {
  if (!refs.alertList) {
    return;
  }
  if (!state.alerts.length) {
    refs.alertList.innerHTML = `<div class="empty">当前没有告警记录。</div>`;
    return;
  }

  const alerts = [...state.alerts].sort((a, b) => {
    const statusOrder = {open: 0, acknowledged: 1, resolved: 2};
    const statusDelta = (statusOrder[a.status] ?? 9) - (statusOrder[b.status] ?? 9);
    if (statusDelta !== 0) {
      return statusDelta;
    }
    return String(b.last_seen_at || "").localeCompare(String(a.last_seen_at || ""));
  }).slice(0, 80);

  refs.alertList.innerHTML = alerts.map((item) => {
    const open = item.status === "open" || item.status === "acknowledged";
    const asset = state.assets.find((candidate) => candidate.id === item.asset_id);
    return `
      <article class="change-card alert-card ${open ? "active" : ""}" data-alert-id="${escapeHTML(item.id)}">
        <div class="section-head">
          <div>
            <p class="change-title">${escapeHTML(item.title || "告警")}</p>
            <span class="change-meta">${escapeHTML(item.asset_name || asset?.name || item.asset_id)} / ${escapeHTML(item.source)} / ${escapeHTML(item.last_seen_at || "-")}</span>
          </div>
          <span class="status-badge status-${item.severity === "critical" ? "failed" : "planned"}">${escapeHTML(item.severity)}</span>
        </div>
        <p class="change-summary">${escapeHTML(item.summary || "无摘要")}</p>
        <div class="change-meta">
          <span class="pill">${escapeHTML(item.status)}</span>
          <span class="pill">次数 ${Number(item.event_count || 0)}</span>
          <span class="pill">首次 ${escapeHTML(formatDateTime(item.first_seen_at))}</span>
          ${item.resolved_at ? `<span class="pill">处理 ${escapeHTML(formatDateTime(item.resolved_at))}</span>` : ""}
        </div>
        <div class="actions">
          <button type="button" data-select-alert-asset="${escapeHTML(item.asset_id)}">查看资产</button>
          ${open ? `<button type="button" data-resolve-alert="${escapeHTML(item.id)}">处理告警</button>` : ""}
        </div>
      </article>
    `;
  }).join("");

  refs.alertList.querySelectorAll("[data-select-alert-asset]").forEach((button) => {
    button.addEventListener("click", () => {
      state.selectedAssetId = button.dataset.selectAlertAsset;
      state.detailMode = "asset";
      state.assetDetailTab = "inspections";
      state.page = "workbench";
      render();
    });
  });
  refs.alertList.querySelectorAll("[data-resolve-alert]").forEach((button) => {
    button.addEventListener("click", async () => {
      const resolution = window.prompt("处理结论", "已确认并处理");
      if (resolution === null) {
        return;
      }
      await apiFetch(`/api/alerts/${button.dataset.resolveAlert}/resolve`, "POST", {resolution});
      await loadData("告警已处理。", "success");
    });
  });
}

function renderRecentSyncs() {
  if (!state.recentSyncs.length) {
    refs.recentSyncsList.innerHTML = `<div class="empty">当前还没有同步记录。</div>`;
    return;
  }

  refs.recentSyncsList.innerHTML = state.recentSyncs.map((item) => `
    <article class="change-card">
      <div class="section-head">
        <p class="change-title">${escapeHTML(item.cloud_account_id)}</p>
        <span class="status-badge status-${item.status === "success" ? "done" : "planned"}">${escapeHTML(item.status)}</span>
      </div>
      <span class="change-meta">${escapeHTML(item.started_at)} -> ${escapeHTML(item.finished_at)}</span>
      <p class="change-summary">${escapeHTML(item.summary || "无摘要")}</p>
    </article>
  `).join("");
}

function renderAuditEvents() {
  if (!refs.auditEventsList) {
    return;
  }

  const events = state.auditEvents || [];
  const actions = Array.from(new Set(events.map((item) => item.action).filter(Boolean))).sort();
  const currentAction = state.auditActionFilter;
  refs.auditActionFilter.innerHTML = `
    <option value="">全部动作</option>
    ${actions.map((action) => `<option value="${escapeHTML(action)}">${escapeHTML(action)}</option>`).join("")}
  `;
  refs.auditActionFilter.value = actions.includes(currentAction) ? currentAction : "";
  state.auditActionFilter = refs.auditActionFilter.value;
  refs.auditOutcomeFilter.value = state.auditOutcomeFilter;
  refs.auditSearchInput.value = state.auditSearch;

  const query = String(state.auditSearch || "").trim().toLowerCase();
  const filtered = events.filter((item) => {
    if (state.auditActionFilter && item.action !== state.auditActionFilter) {
      return false;
    }
    if (state.auditOutcomeFilter && item.outcome !== state.auditOutcomeFilter) {
      return false;
    }
    if (!query) {
      return true;
    }
    const haystack = [
      item.actor,
      item.actor_role,
      item.action,
      item.target_type,
      item.target_id,
      item.target_name,
      item.outcome,
      item.ip,
      item.summary,
      Object.entries(item.metadata || {}).map(([key, value]) => `${key}:${value}`).join(" ")
    ].join(" ").toLowerCase();
    return haystack.includes(query);
  });

  if (!filtered.length) {
    refs.auditEventsList.innerHTML = `<div class="empty">当前筛选下没有审计事件。</div>`;
    return;
  }

  refs.auditEventsList.innerHTML = `
    <div class="audit-table" role="table" aria-label="审计日志">
      <div class="audit-row audit-head" role="row">
        <span>操作时间</span>
        <span>操作人</span>
        <span>动作</span>
        <span>对象</span>
        <span>结果</span>
        <span>来源</span>
        <span>摘要</span>
      </div>
      ${filtered.map((item) => renderAuditEventRow(item)).join("")}
    </div>
  `;
}

function renderAuditEventRow(item) {
  const target = [item.target_type, item.target_name || item.target_id].filter(Boolean).join(" / ") || "-";
  const metadata = Object.entries(item.metadata || {})
    .slice(0, 4)
    .map(([key, value]) => `<span class="pill">${escapeHTML(key)}=${escapeHTML(value)}</span>`)
    .join("");
  const outcomeClass = item.outcome === "success" ? "ok" : item.outcome === "denied" ? "warn" : "danger";
  return `
    <div class="audit-row" role="row">
      <span class="audit-time">${escapeHTML(formatDateTime(item.created_at))}</span>
      <span>
        <strong>${escapeHTML(item.actor || "system")}</strong>
        <small>${escapeHTML(item.actor_role || "system")}</small>
      </span>
      <span><code>${escapeHTML(item.action || "-")}</code></span>
      <span class="audit-target">${escapeHTML(target)}</span>
      <span><b class="audit-outcome ${outcomeClass}">${escapeHTML(item.outcome || "-")}</b></span>
      <span class="audit-source">${escapeHTML(item.ip || "-")}</span>
      <span class="audit-summary">
        ${escapeHTML(item.summary || "-")}
        ${metadata ? `<em>${metadata}</em>` : ""}
      </span>
    </div>
  `;
}
