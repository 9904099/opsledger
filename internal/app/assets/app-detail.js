function renderCloudAccountInspectionCards(account) {
  const assets = state.assets.filter((asset) => asset.cloud_account_id === account.id || asset.cloud_account_name === account.name);
  const assetIds = new Set(assets.map((asset) => asset.id));
  const latestProbeByAsset = latestProbeMap();
  const probeAlerts = [...latestProbeByAsset.values()].filter((probe) => assetIds.has(probe.asset_id) && probe.status !== "up").length;
  const highAssets = assets.filter((asset) => asset.criticality === "high").length;
  const staleSync = account.last_sync_status !== "success" || isOlderThan(account.last_sync_at, 24 * 60 * 60 * 1000);
  const costDelta = Number(account.month_over_month_delta || 0);
  const cards = [
    ["凭证", account.access_key_id_masked ? "已配置" : "缺失", account.access_key_id_masked ? "ok" : "danger"],
    ["最近同步", staleSync ? "异常" : "正常", staleSync ? "warn" : "ok"],
    ["本月费用", account.current_month_cost ? `${account.cost_currency || ""} ${account.current_month_cost}`.trim() : "未同步", account.current_month_cost ? "ok" : "warn"],
    ["同进度差额", account.month_over_month_delta ? `${account.cost_currency || ""} ${account.month_over_month_delta}`.trim() : "-", costDelta > 0 ? "warn" : "ok"],
    ["拨测异常", String(probeAlerts), probeAlerts ? "danger" : "ok"],
    ["高危资产", String(highAssets), highAssets ? "warn" : "ok"]
  ];
  return renderInspectionCards(cards, "account-inspection-grid");
}

function renderAssetInspectionCards(asset, latestProbe, latestSync, relatedChanges, relatedInspections) {
  const autoInspections = relatedInspections.filter((item) => String(item.executor || "").startsWith("auto"));
  const cards = [
    ["资产状态", asset.status || "-", asset.status === "active" ? "ok" : "warn"],
    ["重要级别", asset.criticality || "-", asset.criticality === "high" ? "warn" : "ok"],
    ["拨测状态", latestProbe ? latestProbe.status : "未拨测", latestProbe && latestProbe.status !== "up" ? "danger" : latestProbe ? "ok" : "warn"],
    ["响应耗时", latestProbe && latestProbe.latency_ms ? `${latestProbe.latency_ms} ms` : "-", latestProbe && latestProbe.latency_ms > 3000 ? "warn" : "ok"],
    ["最近同步", latestSync ? formatDateTime(latestSync.finished_at) : "未同步", latestSync ? "ok" : "warn"],
    ["关联变更", String(relatedChanges.length), relatedChanges.length ? "warn" : "ok"],
    ["自动巡检", autoInspections.length ? `${autoInspections.length} 条` : "自动汇总", relatedInspections.some((item) => item.result === "failed") ? "danger" : "ok"],
    ["配置完整性", asset.owner && asset.environment && asset.resource_type ? "完整" : "缺失", asset.owner && asset.environment && asset.resource_type ? "ok" : "warn"]
  ];
  return renderInspectionCards(cards, "asset-inspection-grid");
}

function renderInspectionCards(cards, className = "") {
  return `
    <div class="inspection-card-grid ${className}">
      ${cards.map(([label, value, level]) => `
        <div class="inspection-card ${level || ""}">
          <span>${escapeHTML(label)}</span>
          <strong>${escapeHTML(value)}</strong>
        </div>
      `).join("")}
    </div>
  `;
}

function latestProbeMap() {
  const result = new Map();
  state.probes.forEach((probe) => {
    const existing = result.get(probe.asset_id);
    if (!existing || probe.checked_at > existing.checked_at) {
      result.set(probe.asset_id, probe);
    }
  });
  return result;
}

function isOlderThan(value, durationMS) {
  if (!value) {
    return true;
  }
  const timestamp = new Date(value).getTime();
  if (Number.isNaN(timestamp)) {
    return true;
  }
  return Date.now() - timestamp > durationMS;
}

function renderAssetDetail() {
  if (state.detailMode === "cloudAccount") {
    renderCloudAccountDetail();
    return;
  }

  const asset = state.assets.find((item) => item.id === state.selectedAssetId);
  if (!asset) {
    refs.assetDetail.innerHTML = `<div class="empty">请选择左侧云账号或资产查看详情。</div>`;
    return;
  }

  const filteredAssets = filterAssets();
  const selectedIndex = filteredAssets.findIndex((item) => item.id === asset.id);
  const prevAsset = selectedIndex > 0 ? filteredAssets[selectedIndex - 1] : null;
  const nextAsset = selectedIndex >= 0 && selectedIndex < filteredAssets.length - 1 ? filteredAssets[selectedIndex + 1] : null;

  const detailItems = [
    ["平台", asset.platform_name || asset.platform_code || "-"],
    ["云账号", asset.cloud_account_name || "-"],
    ["账号 ID", asset.account_id || "-"],
    ["项目", projectLabel(asset.project_code)],
    ["大类", asset.category || "-"],
    ["资源类型", asset.resource_type || "-"],
    ["资源名称", asset.name || "-"],
    ["地域", asset.region || "-"],
    ["环境", asset.environment || "-"],
    ["负责人", asset.owner || "-"],
    ["状态", asset.status || "-"],
    ["重要级别", asset.criticality || "-"],
    ["入口 / 地址", asset.endpoint || "-"],
    ["最近巡检", asset.last_checked_at || "-"],
    ["来源", asset.source || "-"],
    ["外部 ID", asset.external_id || "-"],
    ["标签", (asset.tags || []).join(", ") || "-"],
    ["备注", asset.notes || "-"]
  ];

  const specs = Object.entries(asset.specs || {}).filter(([, value]) => value);
  const relatedChanges = state.changes.filter((item) => item.asset_id === asset.id);
  const relatedInspections = state.inspections.filter((item) => item.asset_id === asset.id);
  const relatedProbes = state.probes.filter((item) => item.asset_id === asset.id);
  const relatedSyncs = state.recentSyncs.filter((item) => item.cloud_account_id === asset.cloud_account_id);
  const siblingAssets = state.assets.filter((item) => item.category === asset.category && item.resource_type === asset.resource_type);
  const latestInspection = relatedInspections[0] || null;
  const latestProbe = relatedProbes[0] || null;
  const latestSync = relatedSyncs[0] || null;
  const tabs = [
    ["overview", "概览"],
    ["specs", "规格"],
    ["changes", `关联变更${relatedChanges.length ? ` (${relatedChanges.length})` : ""}`],
    ["inspections", `巡检记录${relatedProbes.length ? ` (${relatedProbes.length})` : ""}`],
    ["syncs", `同步记录${relatedSyncs.length ? ` (${relatedSyncs.length})` : ""}`]
  ];

  refs.assetDetail.innerHTML = `
    <div class="asset-detail-card">
      <div class="asset-detail-sticky ${asset.status !== "active" ? "risk" : ""} ${asset.criticality === "high" ? "critical" : ""}">
        <div class="asset-detail-head">
          <div>
            <h3>${escapeHTML(asset.name)}</h3>
            <p class="muted">${escapeHTML(projectLabel(asset.project_code))} / ${escapeHTML(asset.resource_type)} / ${escapeHTML(asset.category)} / ${escapeHTML(asset.platform_name || asset.platform_code)} / ${selectedIndex >= 0 ? `${selectedIndex + 1} / ${filteredAssets.length}` : "-"}</p>
          </div>
          <div class="asset-head-side">
            <div class="asset-nav">
              <button type="button" id="prevAssetButton" ${prevAsset ? "" : "disabled"}>上一条</button>
              <button type="button" id="nextAssetButton" ${nextAsset ? "" : "disabled"}>下一条</button>
            </div>
            <span class="status-badge status-${escapeHTML(asset.status)}">${escapeHTML(asset.status)}</span>
          </div>
        </div>
        <div class="asset-detail-actions">
          <button type="button" id="jumpToCloudAccountButton">查看所属云账号</button>
          <button type="button" id="editAssetButton">编辑资产</button>
          <button type="button" id="createChangeForAssetButton">发起变更</button>
        </div>
        <div class="asset-detail-tabs">
          ${tabs.map(([tab, label]) => `
            <button type="button" class="asset-detail-tab ${state.assetDetailTab === tab ? "active" : ""}" data-asset-tab="${tab}">${escapeHTML(label)}</button>
          `).join("")}
        </div>
      </div>
      <div class="asset-detail-panel">
        <div class="asset-overview-strip">
          ${renderOverviewMetric("所属云账号", asset.cloud_account_name || "-", asset.account_id || "", "account")}
          ${renderOverviewMetric("最近同步", latestSync ? formatDateTime(latestSync.finished_at) : "未同步", latestSync ? latestSync.summary : "无同步记录", "syncs")}
          ${renderOverviewMetric("最近巡检", latestProbe ? formatDateTime(latestProbe.checked_at) : latestInspection ? latestInspection.checked_at : "未巡检", latestProbe ? `${latestProbe.status} / ${latestProbe.latency_ms} ms` : latestInspection ? latestInspection.result : "无拨测记录", "inspections")}
          ${renderOverviewMetric("关联变更", String(relatedChanges.length), relatedChanges.length ? "可在下方查看详情" : "暂无变更", "changes")}
          ${renderOverviewMetric("同类资源", String(siblingAssets.length), `${asset.resource_type}`, "peers")}
        </div>
        ${renderAssetInspectionCards(asset, latestProbe, latestSync, relatedChanges, relatedInspections)}
        ${renderAssetDetailPanel(asset, detailItems, specs, relatedChanges, relatedInspections, relatedSyncs, relatedProbes)}
      </div>
    </div>
  `;

  bindAssetDetailEvents(asset, prevAsset, nextAsset);
}

function renderCloudAccountDetail() {
  const account = state.cloudAccounts.find((item) => item.id === state.selectedCloudAccountId);
  if (!account) {
    refs.assetDetail.innerHTML = `<div class="empty">请选择左侧云账号查看账号详情。</div>`;
    return;
  }

  const assets = state.assets.filter((asset) => asset.cloud_account_id === account.id || asset.cloud_account_name === account.name);
  const assetIds = new Set(assets.map((asset) => asset.id));
  const typeRows = [...assets.reduce((groups, asset) => {
    const key = asset.resource_type || "Unknown";
    groups.set(key, (groups.get(key) || 0) + 1);
    return groups;
  }, new Map()).entries()].sort((left, right) => right[1] - left[1]);
  const recentSyncs = state.recentSyncs.filter((item) => item.cloud_account_id === account.id).slice(0, 6);
  const costSnapshot = buildAccountCostSnapshot(account);
  const autoInspections = state.inspections
    .filter((item) => assetIds.has(item.asset_id) && String(item.executor || "").startsWith("auto"))
    .slice(0, 8);
  const probeAlerts = [...latestProbeMap().values()].filter((probe) => assetIds.has(probe.asset_id) && probe.status !== "up");
  const detailItems = [
    ["平台", account.platform_name || account.platform_code || "-"],
    ["云账号", account.name || "-"],
    ["账号 ID / 标识", account.account_id || "-"],
    ["默认 Region", account.default_region || "-"],
    ["环境", account.environment || "-"],
    ["负责人", account.owner || "-"],
    ["重要级别", account.criticality || "-"],
    ["凭证", account.access_key_id_masked || "未配置"],
    ["同步模式", account.sync_mode || "manual"],
    ["自动同步", account.sync_enabled ? "启用" : "未启用"],
    ["同步计划", account.sync_cron || "-"],
    ["下次自动同步", renderNextSyncHint(account)],
    ["最近同步", account.last_sync_at ? formatDateTime(account.last_sync_at) : "-"],
    ["同步状态", account.last_sync_status || "-"],
    ["费用同步", account.last_cost_sync_at ? formatDateTime(account.last_cost_sync_at) : "-"]
  ];

  refs.assetDetail.innerHTML = `
    <div class="asset-detail-card account-detail-card">
      <div class="asset-detail-sticky ${account.criticality === "high" ? "critical" : ""}">
        <div class="asset-detail-head">
          <div>
            <h3>${escapeHTML(account.name)}</h3>
            <p class="muted">${escapeHTML(account.platform_name || account.platform_code)} / ${escapeHTML(account.environment || "-")} / ${assets.length} 个资产</p>
          </div>
          <div class="asset-head-side">
            <span class="status-badge status-${account.last_sync_status === "success" ? "done" : "planned"}">${escapeHTML(account.last_sync_status || "not_synced")}</span>
          </div>
        </div>
        <div class="asset-detail-actions">
          <button type="button" id="editCloudAccountFromDetailButton">编辑配置</button>
          <button type="button" id="syncCloudAccountFromDetailButton">同步资产</button>
          ${account.platform_code === "aws" ? `<button type="button" id="syncCloudAccountCostFromDetailButton">同步费用</button>` : ""}
        </div>
      </div>
      <div class="asset-detail-panel">
        <div class="cost-strip account-cost-strip">
          ${renderCostMetric("上月费用", account.last_month_cost, account.cost_currency)}
          ${renderCostMetric("上月同进度", account.last_month_to_date_cost, account.cost_currency)}
          ${renderCostMetric("本月当前使用", account.current_month_cost, account.cost_currency)}
          ${renderCostMetric("本月预计", account.forecast_month_cost, account.cost_currency)}
          ${renderCostMetric("同进度差额", account.month_over_month_delta, account.cost_currency, Number(account.month_over_month_delta || 0) > 0 ? "up" : Number(account.month_over_month_delta || 0) < 0 ? "down" : "")}
        </div>
        ${renderCloudAccountInspectionCards(account)}
        ${renderAccountCostExplorerPanel(costSnapshot)}
        <div class="asset-spec-grid">
          ${detailItems.map(([label, value]) => `
            <div class="asset-spec-item">
              <span class="asset-spec-label">${escapeHTML(label)}</span>
              <span class="asset-spec-value">${escapeHTML(value)}</span>
            </div>
          `).join("")}
        </div>
        <div class="account-detail-grid">
          <section class="account-detail-section">
            <h4>资产类型分布</h4>
            ${typeRows.length ? typeRows.map(([type, count]) => `
              <button type="button" class="account-type-row" data-account-type="${escapeHTML(type)}">
                <span>${escapeHTML(type)}</span>
                <strong>${count}</strong>
              </button>
            `).join("") : `<div class="empty">当前云账号没有资产。</div>`}
          </section>
          <section class="account-detail-section">
            <h4>自动巡检异常</h4>
            ${probeAlerts.length || autoInspections.length ? `
              <div class="asset-related-list">
                ${probeAlerts.slice(0, 5).map((probe) => {
                  const asset = state.assets.find((item) => item.id === probe.asset_id);
                  return `
                    <article class="change-card" data-account-asset-id="${escapeHTML(probe.asset_id)}">
                      <div class="section-head">
                        <p class="change-title">${escapeHTML(asset ? asset.name : probe.asset_id)}</p>
                        <span class="status-badge status-cancelled">${escapeHTML(probe.status)}</span>
                      </div>
                      <span class="change-meta">${escapeHTML(formatDateTime(probe.checked_at))} / HTTP ${probe.status_code || "-"} / ${probe.latency_ms || 0} ms</span>
                    </article>
                  `;
                }).join("")}
                ${autoInspections.slice(0, 5).map((inspection) => `
                  <article class="change-card" data-account-asset-id="${escapeHTML(inspection.asset_id)}">
                    <div class="section-head">
                      <p class="change-title">${escapeHTML(inspection.summary || "自动巡检")}</p>
                      <span class="status-badge status-${inspection.result === "ok" ? "done" : inspection.result === "warning" ? "in_progress" : "cancelled"}">${escapeHTML(inspection.result)}</span>
                    </div>
                    <span class="change-meta">${escapeHTML(inspection.executor)} / ${escapeHTML(inspection.checked_at)}</span>
                  </article>
                `).join("")}
              </div>
            ` : `<div class="empty">当前账号没有自动巡检异常。</div>`}
          </section>
          <section class="account-detail-section span-2">
            <h4>最近同步留痕</h4>
            ${recentSyncs.length ? `
              <div class="asset-related-list">
                ${recentSyncs.map((sync) => `
                  <article class="change-card">
                    <div class="section-head">
                      <p class="change-title">${escapeHTML(sync.summary || sync.id)}</p>
                      <span class="status-badge status-${sync.status === "success" ? "done" : "planned"}">${escapeHTML(sync.status)}</span>
                    </div>
                    <span class="change-meta">${escapeHTML(sync.started_at)} -> ${escapeHTML(sync.finished_at)}</span>
                    <p class="change-summary">${escapeHTML(formatSyncRecordSummary(sync))}</p>
                  </article>
                `).join("")}
              </div>
            ` : `<div class="empty">当前账号还没有同步记录。</div>`}
          </section>
        </div>
      </div>
    </div>
  `;

  bindCloudAccountDetailEvents(account);
}

function buildAccountCostSnapshot(account) {
  const records = (state.costRecords || []).filter((item) => item.cloud_account_id === account.id);
  const currentMonthStart = new Date();
  currentMonthStart.setDate(1);
  const monthKey = currentMonthStart.toISOString().slice(0, 7);
  const serviceRecords = records
    .filter((item) => item.granularity === "monthly" && item.dimension_type === "service" && String(item.period_start || "").startsWith(monthKey))
    .sort((left, right) => Number(right.amount || 0) - Number(left.amount || 0));
  const dailyTotal = records
    .filter((item) => item.granularity === "daily" && item.dimension_type === "total")
    .sort((left, right) => String(right.period_start || "").localeCompare(String(left.period_start || "")));
  return {
    serviceRecords,
    dailyTotal,
    latestDaily: dailyTotal[0] || null,
    currency: account.cost_currency || (serviceRecords[0] && serviceRecords[0].currency) || (dailyTotal[0] && dailyTotal[0].currency) || "USD"
  };
}

function renderAccountCostExplorerPanel(snapshot) {
  const serviceRows = snapshot.serviceRecords.slice(0, 8);
  const dailyRows = snapshot.dailyTotal.slice(0, 7);
  return `
    <section class="account-cost-explorer">
      <div class="section-head">
        <div>
          <h4>AWS Cost Explorer 真实费用</h4>
          <p class="muted">服务维度与每日快照来自 AWS 账单接口，不按资产数量估值。</p>
        </div>
        <span class="sync-hint">${snapshot.latestDaily ? `最新日快照 ${escapeHTML(snapshot.latestDaily.period_start)}` : "暂无日快照"}</span>
      </div>
      <div class="account-cost-explorer-grid">
        <div class="account-cost-section">
          <h5>本月服务费用</h5>
          ${serviceRows.length ? serviceRows.map((item) => `
            <div class="cost-service-row">
              <span title="${escapeHTML(item.dimension_name)}">${escapeHTML(shortAWSServiceName(item.dimension_name))}</span>
              <strong>${escapeHTML(`${item.currency || snapshot.currency} ${item.amount || "0.00"}`)}</strong>
            </div>
          `).join("") : `<div class="empty compact">请先同步费用，或确认账号已开通 Cost Explorer。</div>`}
        </div>
        <div class="account-cost-section">
          <h5>最近每日记录</h5>
          ${dailyRows.length ? dailyRows.map((item) => `
            <div class="cost-service-row">
              <span>${escapeHTML(item.period_start)}</span>
              <strong>${escapeHTML(`${item.currency || snapshot.currency} ${item.amount || "0.00"}`)}</strong>
            </div>
          `).join("") : `<div class="empty compact">暂无每日费用快照。</div>`}
        </div>
      </div>
    </section>
  `;
}

function shortAWSServiceName(name) {
  const text = String(name || "").trim();
  const aliases = {
    "Amazon Elastic Compute Cloud - Compute": "EC2 - Compute",
    "EC2 - Other": "EC2 - Other",
    "Amazon Elastic File System": "Elastic File System",
    "Amazon Relational Database Service": "RDS",
    "Amazon Simple Storage Service": "S3",
    "Amazon CloudFront": "CloudFront",
    "AWS Cost Explorer": "Cost Explorer",
    "Tax": "Tax"
  };
  return aliases[text] || text.replace(/^Amazon\s+/, "").replace(/^AWS\s+/, "") || "Unknown";
}

function bindCloudAccountDetailEvents(account) {
  const editButton = document.getElementById("editCloudAccountFromDetailButton");
  if (editButton) {
    editButton.addEventListener("click", () => {
      state.page = "config";
      syncFormsFromSelection();
      render();
      refs.cloudAccountName.focus();
    });
  }

  const syncButton = document.getElementById("syncCloudAccountFromDetailButton");
  if (syncButton) {
    syncButton.addEventListener("click", async () => {
      syncButton.disabled = true;
      syncButton.textContent = "同步中...";
      try {
        const result = await apiFetch("/api/cloud-accounts/sync", "POST", {
          cloud_account_id: account.id,
          region: account.default_region || ""
        });
        const warnings = (result.warnings || []).length;
        state.detailMode = "cloudAccount";
        state.selectedCloudAccountId = account.id;
        await loadData(formatSyncMessage(result), warnings ? "error" : "success");
      } finally {
        syncButton.disabled = false;
        syncButton.textContent = "同步资产";
      }
    });
  }

  const costButton = document.getElementById("syncCloudAccountCostFromDetailButton");
  if (costButton) {
    costButton.addEventListener("click", async () => {
      costButton.disabled = true;
      costButton.textContent = "同步中...";
      try {
        const result = await apiFetch(`/api/cloud-accounts/${account.id}/cost-sync`, "POST", {});
        state.detailMode = "cloudAccount";
        state.selectedCloudAccountId = account.id;
        await loadData(formatCostSyncMessage(result), "success");
      } finally {
        costButton.disabled = false;
        costButton.textContent = "同步费用";
      }
    });
  }

  refs.assetDetail.querySelectorAll("[data-account-type]").forEach((button) => {
    button.addEventListener("click", () => {
      state.treeCloudAccountFilter = account.name;
      state.treeResourceTypeFilter = button.dataset.accountType;
      state.detailMode = "cloudAccount";
      render();
    });
  });

  refs.assetDetail.querySelectorAll("[data-account-asset-id]").forEach((card) => {
    card.addEventListener("click", () => {
      state.selectedAssetId = card.dataset.accountAssetId;
      state.selectedChangeId = "";
      state.assetDetailTab = "inspections";
      state.detailMode = "asset";
      syncFormsFromSelection();
      render();
    });
  });
}

function bindAssetDetailEvents(asset, prevAsset, nextAsset) {
  refs.assetDetail.querySelectorAll("[data-asset-tab]").forEach((button) => {
    button.addEventListener("click", () => {
      state.assetDetailTab = button.dataset.assetTab;
      renderAssetDetail();
    });
  });

  refs.assetDetail.querySelectorAll("[data-overview-action]").forEach((card) => {
    card.addEventListener("click", () => {
      const action = card.dataset.overviewAction;
      if (action === "account") {
        const jumpButton = document.getElementById("jumpToCloudAccountButton");
        if (jumpButton) {
          jumpButton.click();
        }
        return;
      }
      if (action === "syncs" || action === "inspections" || action === "changes") {
        state.assetDetailTab = action;
        renderAssetDetail();
      }
    });
  });

  refs.assetDetail.querySelectorAll("[data-detail-change-id]").forEach((card) => {
    card.addEventListener("click", () => {
      state.selectedChangeId = card.dataset.detailChangeId;
      state.page = "config";
      syncFormsFromSelection();
      showMessage("已回填该变更到配置页表单。", "success");
      render();
    });
  });

  const jumpButton = document.getElementById("jumpToCloudAccountButton");
  if (jumpButton) {
    jumpButton.addEventListener("click", () => {
      if (!asset.cloud_account_id) {
        showMessage("当前资产没有归属云账号。", "error");
        return;
      }
      state.selectedCloudAccountId = asset.cloud_account_id;
      state.selectedAssetId = "";
      state.detailMode = "cloudAccount";
      state.page = "workbench";
      syncFormsFromSelection();
      render();
    });
  }

  const editButton = document.getElementById("editAssetButton");
  if (editButton) {
    editButton.addEventListener("click", () => {
      syncFormsFromSelection();
      state.page = "config";
      showMessage(`已切换到资产 ${asset.name} 的编辑表单。`, "success");
      render();
      refs.assetName.focus();
    });
  }

  const createChangeButton = document.getElementById("createChangeForAssetButton");
  if (createChangeButton) {
    createChangeButton.addEventListener("click", () => {
      state.selectedChangeId = "";
      state.pendingChangeTitlePrefix = `[${asset.name}] `;
      resetChangeForm();
      refs.changeAssetId.value = asset.id;
      refs.changeTitle.value = state.pendingChangeTitlePrefix;
      refs.changeSummary.value = `关联资产：${asset.name}`;
      refs.changeExecutor.value = asset.owner || "";
      refs.changeRiskLevel.value = asset.criticality === "high" ? "high" : "medium";
      refs.changeCategory.value = asset.category === "network" ? "network" : "release";
      state.page = "config";
      showMessage(`已为资产 ${asset.name} 预填变更表单。`, "success");
      render();
      refs.changeTitle.focus();
    });
  }

  const runProbeButton = document.getElementById("runProbeButton");
  if (runProbeButton) {
    runProbeButton.addEventListener("click", async () => {
      runProbeButton.disabled = true;
      runProbeButton.textContent = "拨测中...";
      try {
        const result = await apiFetch(`/api/assets/${asset.id}/probe`, "POST", {});
        state.assetDetailTab = "inspections";
        await loadData(`拨测完成：${result.status} / ${result.latency_ms} ms。`, result.status === "up" ? "success" : "error");
      } finally {
        runProbeButton.disabled = false;
        runProbeButton.textContent = "手动补测";
      }
    });
  }

  document.querySelectorAll("[data-upload-inspection-attachment]").forEach((input) => {
    input.addEventListener("change", async () => {
      const file = input.files && input.files[0];
      if (!file) {
        return;
      }
      const form = new FormData();
      form.append("file", file);
      await apiFetchForm(`/api/inspections/${input.dataset.uploadInspectionAttachment}/attachments`, "POST", form);
      state.assetDetailTab = "inspections";
      await loadData("巡检附件已上传。", "success");
    });
  });

  const prevAssetButton = document.getElementById("prevAssetButton");
  if (prevAssetButton) {
    prevAssetButton.addEventListener("click", () => {
      if (!prevAsset) {
        return;
      }
      state.selectedAssetId = prevAsset.id;
      state.selectedChangeId = "";
      state.detailMode = "asset";
      state.assetDetailTab = "overview";
      syncFormsFromSelection();
      render();
    });
  }

  const nextAssetButton = document.getElementById("nextAssetButton");
  if (nextAssetButton) {
    nextAssetButton.addEventListener("click", () => {
      if (!nextAsset) {
        return;
      }
      state.selectedAssetId = nextAsset.id;
      state.selectedChangeId = "";
      state.detailMode = "asset";
      state.assetDetailTab = "overview";
      syncFormsFromSelection();
      render();
    });
  }
}

function renderAssetDetailPanel(asset, detailItems, specs, relatedChanges, relatedInspections, relatedSyncs, relatedProbes) {
  switch (state.assetDetailTab) {
    case "specs":
      return specs.length ? `
        <div class="asset-spec-section">
          <h4>规格细项</h4>
          <div class="asset-spec-grid">
            ${specs.map(([label, value]) => `
              <div class="asset-spec-item">
                <span class="asset-spec-label">${escapeHTML(formatSpecLabel(label))}</span>
                <span class="asset-spec-value">${escapeHTML(value)}</span>
              </div>
            `).join("")}
          </div>
        </div>
      ` : `<div class="empty">当前资产暂无结构化规格细项。</div>`;
    case "changes":
      return relatedChanges.length ? `
        <div class="asset-related-list">
          ${relatedChanges.map((change) => `
            <article class="change-card ${change.id === state.selectedChangeId ? "active" : ""}" data-detail-change-id="${change.id}">
              <div class="section-head">
                <p class="change-title">${escapeHTML(change.title)}</p>
                <span class="status-badge status-${escapeHTML(change.status)}">${escapeHTML(change.status)}</span>
              </div>
              <span class="change-meta">${escapeHTML(change.executor)} / ${escapeHTML(change.window)}</span>
              <p class="change-summary">${escapeHTML(change.summary || "无摘要")}</p>
            </article>
          `).join("")}
        </div>
      ` : `<div class="empty">当前资产还没有关联变更记录。</div>`;
    case "inspections":
      return renderProbePanel(asset, relatedInspections, relatedProbes);
    case "syncs":
      return relatedSyncs.length ? `
        <div class="asset-related-list">
          ${relatedSyncs.map((sync) => `
            <article class="change-card">
              <div class="section-head">
                <p class="change-title">${escapeHTML(sync.summary || sync.id)}</p>
                <span class="status-badge status-${sync.status === "success" ? "done" : "planned"}">${escapeHTML(sync.status)}</span>
              </div>
              <span class="change-meta">${escapeHTML(sync.started_at)} -> ${escapeHTML(sync.finished_at)}</span>
              <p class="change-summary">${escapeHTML(formatSyncRecordSummary(sync))}</p>
            </article>
          `).join("")}
        </div>
      ` : `<div class="empty">当前资产所属云账号还没有同步记录。</div>`;
    case "overview":
    default:
      return `
        <div class="asset-spec-grid">
          ${detailItems.map(([label, value]) => `
            <div class="asset-spec-item">
              <span class="asset-spec-label">${escapeHTML(label)}</span>
              <span class="asset-spec-value">${escapeHTML(value)}</span>
            </div>
          `).join("")}
        </div>
      `;
  }
}

function renderOverviewMetric(label, value, hint, action = "") {
  return `
    <button type="button" class="asset-overview-metric ${action ? "clickable" : ""}" ${action ? `data-overview-action="${escapeHTML(action)}"` : ""}>
      <span class="asset-overview-label">${escapeHTML(label)}</span>
      <strong class="asset-overview-value">${escapeHTML(value)}</strong>
      <small class="asset-overview-hint">${escapeHTML(hint || "-")}</small>
    </button>
  `;
}

function renderProbePanel(asset, relatedInspections, relatedProbes) {
  const probe = buildProbeSnapshot(asset, relatedProbes);
  return `
    <section class="probe-panel">
      <div class="probe-title-row">
        <div>
          <h4>${escapeHTML(asset.name)}</h4>
          <a href="${escapeHTML(probe.url)}" target="_blank" rel="noreferrer">${escapeHTML(probe.url)}</a>
        </div>
        <div class="probe-actions">
          <button type="button" id="runProbeButton">手动补测</button>
          <span class="probe-state ${probe.stateClass}">${probe.statusLabel}</span>
        </div>
      </div>

      <div class="probe-bars" aria-label="最近拨测状态">
        ${probe.bars.length ? probe.bars.map((bar) => `<span class="${bar.ok ? "ok" : "bad"}" title="${escapeHTML(bar.label)}"></span>`).join("") : `<em>暂无拨测数据</em>`}
      </div>
      <p class="probe-note">Auto HTTP probe, timeout ${probe.timeoutSeconds} seconds. Manual probe is available for immediate verification.</p>

      <div class="probe-metrics">
        ${renderProbeMetric("Response", "(Current)", probe.currentLatency ? `${probe.currentLatency} ms` : "-")}
        ${renderProbeMetric("Avg. Response", "(24-hour)", probe.avgLatency ? `${probe.avgLatency} ms` : "-")}
        ${renderProbeMetric("Uptime", "(24-hour)", `${probe.uptime24h}%`)}
        ${renderProbeMetric("Uptime", "(30-day)", `${probe.uptime30d}%`)}
        ${renderProbeMetric("Cert Exp.", probe.certDate, probe.certDaysLeft ? `${probe.certDaysLeft} days` : "-")}
      </div>

      <div class="probe-chart">
        ${renderLatencyChart(probe.series)}
      </div>

      ${relatedProbes.length ? `
        <div class="asset-related-list probe-history">
          ${relatedProbes.slice(0, 12).map((item) => `
            <article class="change-card">
              <div class="section-head">
                <p class="change-title">${escapeHTML(formatDateTime(item.checked_at))}</p>
                <span class="status-badge status-${item.status === "up" ? "done" : "cancelled"}">${escapeHTML(item.status)}</span>
              </div>
              <span class="change-meta">${escapeHTML(item.url)} / HTTP ${item.status_code || "-"} / ${item.latency_ms || 0} ms</span>
              ${item.error ? `<p class="change-summary">${escapeHTML(item.error)}</p>` : ""}
            </article>
          `).join("")}
        </div>
      ` : `<div class="probe-empty">暂无真实拨测记录，后台自动拨测任务会写入第一条记录，也可以点击“手动补测”。</div>`}

      ${relatedInspections.length ? `
        <div class="asset-related-list probe-history">
          ${relatedInspections.map((inspection) => `
            <article class="change-card">
              <div class="section-head">
                <p class="change-title">${escapeHTML(inspection.summary || "巡检记录")}</p>
                <span class="status-badge status-${inspection.result === "ok" ? "done" : inspection.result === "warning" ? "in_progress" : "cancelled"}">${escapeHTML(inspection.result)}</span>
              </div>
              <span class="change-meta">${escapeHTML(inspection.executor)} / ${escapeHTML(inspection.checked_at)}</span>
              ${renderInspectionAttachments(inspection)}
            </article>
          `).join("")}
        </div>
      ` : ""}
    </section>
  `;
}

function renderInspectionAttachments(inspection) {
  const attachments = state.attachments.filter((item) => item.inspection_id === inspection.id);
  const canUpload = state.currentUser && ["admin", "ops"].includes(state.currentUser.role);
  return `
    <div class="inspection-attachments">
      <div class="inspection-attachment-head">
        <span>附件 ${attachments.length}</span>
        ${canUpload ? `
          <label class="attachment-upload-button">
            上传
            <input type="file" data-upload-inspection-attachment="${escapeHTML(inspection.id)}">
          </label>
        ` : ""}
      </div>
      ${attachments.length ? `
        <div class="inspection-attachment-list">
          ${attachments.map((item) => `
            <a href="/api/inspection-attachments/${encodeURIComponent(item.id)}/download" target="_blank" rel="noreferrer">
              <span>${escapeHTML(item.file_name)}</span>
              <small>${escapeHTML(formatFileSize(item.size_bytes))} / ${escapeHTML(item.uploader || "-")} / ${escapeHTML(formatDateTime(item.created_at))}</small>
            </a>
          `).join("")}
        </div>
      ` : `<p class="muted attachment-empty">暂无附件</p>`}
    </div>
  `;
}

function renderProbeMetric(label, hint, value) {
  return `
    <div class="probe-metric">
      <span>${escapeHTML(label)}</span>
      <small>${escapeHTML(hint)}</small>
      <strong>${escapeHTML(value)}</strong>
    </div>
  `;
}

function buildProbeSnapshot(asset, relatedProbes) {
  const ordered = [...relatedProbes].sort((left, right) => new Date(left.checked_at) - new Date(right.checked_at));
  const latest = ordered[ordered.length - 1] || null;
  const series = ordered.map((item) => item.status === "up" ? item.latency_ms : 0);
  const up = latest ? latest.status === "up" : false;
  const successful = ordered.filter((item) => item.status === "up");
  const currentLatency = latest && latest.status === "up" ? latest.latency_ms : 0;
  const avgLatency = successful.length
    ? Math.round(successful.reduce((sum, item) => sum + item.latency_ms, 0) / successful.length)
    : 0;
  const uptime24h = formatUptime(filterProbesSince(ordered, 24 * 60 * 60 * 1000));
  const uptime30d = formatUptime(filterProbesSince(ordered, 30 * 24 * 60 * 60 * 1000));
  const certDaysLeft = latest ? latest.cert_days_remaining : 0;
  const certDate = latest && latest.tls_expires_at ? latest.tls_expires_at.slice(0, 10) : "-";
  return {
    up,
    statusLabel: latest ? (up ? "Up" : "Down") : "未拨测",
    stateClass: latest ? (up ? "up" : "down") : "unknown",
    url: buildProbeURL(asset),
    timeoutSeconds: 10,
    currentLatency,
    avgLatency,
    uptime24h,
    uptime30d,
    certDaysLeft,
    certDate,
    series,
    series: series.length ? series : [0],
    bars: ordered.slice(-32).map((item, index) => ({
      ok: item.status === "up",
      label: `${index + 1}: ${item.status} / ${item.latency_ms || 0} ms`
    }))
  };
}

function filterProbesSince(items, durationMS) {
  if (!items.length) {
    return [];
  }
  const cutoff = Date.now() - durationMS;
  return items.filter((item) => new Date(item.checked_at).getTime() >= cutoff);
}

function formatUptime(items) {
  if (!items.length) {
    return "0";
  }
  const upCount = items.filter((item) => item.status === "up").length;
  return ((upCount / items.length) * 100).toFixed(items.length === upCount ? 0 : 1);
}

function buildProbeURL(asset) {
  const raw = asset.resource_type === "DNS Record" ? asset.name : asset.endpoint || asset.name;
  if (!raw) {
    return "#";
  }
  if (/^https?:\/\//i.test(raw)) {
    return raw;
  }
  return `https://${raw}`;
}

function renderLatencyChart(series) {
  const width = 720;
  const height = 230;
  const padding = {top: 16, right: 18, bottom: 28, left: 48};
  const chartWidth = width - padding.left - padding.right;
  const chartHeight = height - padding.top - padding.bottom;
  const maxValue = Math.max(400, Math.ceil(Math.max(...series, 1) / 100) * 100);
  const points = series.map((value, index) => {
    const x = padding.left + (index / Math.max(1, series.length - 1)) * chartWidth;
    const y = padding.top + (1 - value / maxValue) * chartHeight;
    return `${x.toFixed(1)},${y.toFixed(1)}`;
  }).join(" ");
  const fillPoints = `${padding.left},${padding.top + chartHeight} ${points} ${padding.left + chartWidth},${padding.top + chartHeight}`;
  const yTicks = [0, 0.25, 0.5, 0.75, 1].map((ratio) => {
    const value = Math.round(maxValue * ratio);
    const y = padding.top + (1 - ratio) * chartHeight;
    return {value, y};
  });
  const xTicks = ["-6h", "-5h", "-4h", "-3h", "-2h", "-1h", "now"];
  return `
    <svg viewBox="0 0 ${width} ${height}" role="img" aria-label="拨测时延折线图">
      <rect x="0" y="0" width="${width}" height="${height}" rx="8"></rect>
      ${yTicks.map((tick) => `
        <line class="grid-line" x1="${padding.left}" y1="${tick.y}" x2="${padding.left + chartWidth}" y2="${tick.y}"></line>
        <text class="axis-label" x="${padding.left - 10}" y="${tick.y + 4}" text-anchor="end">${tick.value}</text>
      `).join("")}
      ${xTicks.map((label, index) => {
        const x = padding.left + (index / (xTicks.length - 1)) * chartWidth;
        return `
          <line class="grid-line" x1="${x}" y1="${padding.top}" x2="${x}" y2="${padding.top + chartHeight}"></line>
          <text class="axis-label" x="${x}" y="${height - 8}" text-anchor="middle">${label}</text>
        `;
      }).join("")}
      <polygon class="latency-fill" points="${fillPoints}"></polygon>
      <polyline class="latency-line" points="${points}"></polyline>
      <text class="axis-label" x="8" y="24">Resp. Time (ms)</text>
    </svg>
  `;
}
