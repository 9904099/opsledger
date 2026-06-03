// Developer workbench rendering and approval modal helpers.

function renderDeveloperPage() {
  const envs = developerWorkbenchEnvironments();
  const currentEnv = envs.find((item) => item.code === state.developerEnvironment) || envs[0];
  const envCode = currentEnv ? currentEnv.code : "dev";
  const apps = state.tools.filter((tool) => tool.environment === envCode && tool.tool_type === "business");
  const tools = state.tools.filter((tool) => tool.environment === "global" && tool.tool_type !== "business");
  const websshAssets = state.assets.filter((asset) => asset.environment === envCode && isWebSSHAsset(asset));
  const otherAssets = state.assets.filter((asset) => asset.environment === envCode && asset.category !== "tool" && !isWebSSHAsset(asset));
  const assets = [...websshAssets, ...otherAssets];
  const pendingApprovals = state.approvals.filter((item) => item.status === "pending" && canDecideApproval(item));
  refs.openApprovalModalButton.textContent = pendingApprovals.length ? `审批待办 (${pendingApprovals.length})` : "提交申请";

  refs.developerEnvTabs.innerHTML = envs.map((env) => `
    <button type="button" class="${env.code === envCode ? "active" : ""}" data-dev-env="${escapeHTML(env.code)}">
      <strong>${escapeHTML(env.code)}</strong>
      <small>${escapeHTML(env.name || env.code)}</small>
    </button>
  `).join("");
  refs.developerEnvTabs.querySelectorAll("[data-dev-env]").forEach((button) => {
    button.addEventListener("click", () => {
      state.developerEnvironment = button.dataset.devEnv;
      render();
    });
  });

  refs.developerOverview.innerHTML = `
    ${renderDeveloperMetric("应用入口", String(apps.length), "当前环境")}
    ${renderDeveloperMetric("工具入口", String(tools.length), "全局工具")}
    ${renderDeveloperMetric("可访问资产", String(assets.length), "当前环境")}
    ${renderDeveloperMetric("WebSSH", String(websshAssets.length), "EC2 可申请")}
  `;

  refs.developerApps.innerHTML = apps.length
    ? apps.map((tool) => renderDeveloperEntryCard(tool, envCode, "打开应用")).join("")
    : `<div class="empty">当前环境还没有配置应用入口。</div>`;
  refs.developerTools.innerHTML = tools.length
    ? tools.map((tool) => renderDeveloperEntryCard(tool, envCode, "打开工具")).join("")
    : `<div class="empty">还没有配置全局工具入口。</div>`;

  refs.developerAssets.innerHTML = renderDeveloperWebSSHAssetTree(websshAssets);

  refs.approvalTargetId.innerHTML = [
    ...apps.map((tool) => ["asset", tool.asset_id, `${tool.asset_name} / ${tool.environment}`]),
    ...tools.map((tool) => ["asset", tool.asset_id, `${tool.asset_name} / ${tool.environment}`]),
    ...assets.map((asset) => ["asset", asset.id, `${asset.name} / ${asset.environment}`])
  ].map(([type, id, label]) => `<option value="${escapeHTML(id)}" data-target-type="${escapeHTML(type)}">${escapeHTML(label)}</option>`).join("");
  refs.approvalEnvironment.value = envCode;

  refs.developerPage.querySelectorAll("[data-open-tool]").forEach((button) => {
    button.addEventListener("click", () => window.open(button.dataset.openTool, "_blank", "noopener"));
  });
  refs.developerPage.querySelectorAll("[data-request-tool]").forEach((button) => {
    button.addEventListener("click", () => {
      refs.approvalTargetId.value = button.dataset.requestTool;
      refs.approvalEnvironment.value = button.dataset.requestEnv;
      refs.approvalRequestType.value = button.dataset.requestKind;
      openApprovalModal("request");
    });
  });
  refs.developerPage.querySelectorAll("[data-reveal-dev-credential]").forEach((button) => {
    button.addEventListener("click", async () => {
      const credentialID = button.dataset.revealDevCredential;
      const credential = state.credentials.find((item) => item.id === credentialID);
      const grant = credential ? activeAccessGrant("credential", credential.owner_type, credential.owner_id) || activeAccessGrant("credential", "credential", credential.id) : null;
      if (credential && !grant && credential.access_policy !== "viewable") {
        refs.approvalTargetId.value = credential.owner_id;
        refs.approvalEnvironment.value = credential.environment || envCode;
        refs.approvalRequestType.value = "credential";
        openApprovalModal("request");
        return;
      }
      const result = await apiFetch(`/api/credentials/${credentialID}/reveal`, "POST", {});
      showMessage(`凭证明文：${result.value}`, "success");
      await loadData("", "success");
    });
  });
  refs.developerPage.querySelectorAll("[data-open-webssh]").forEach((button) => {
    button.addEventListener("click", async () => {
      const session = await apiFetch("/api/webssh/open", "POST", {asset_id: button.dataset.openWebssh});
      showMessage("WebSSH 临时会话已创建。", "success");
      if (session.login_url) {
        window.open(session.login_url, "_blank", "noopener");
      }
      await loadData("", "success");
    });
  });
  refs.developerPage.querySelectorAll("[data-select-asset]").forEach((button) => {
    button.addEventListener("click", () => {
      openDeveloperAssetModal(button.dataset.selectAsset);
    });
  });
  refs.developerPage.querySelectorAll("[data-dev-webssh-tree-key]").forEach((button) => {
    button.addEventListener("click", () => {
      const key = button.dataset.devWebsshTreeKey;
      const expanded = button.getAttribute("aria-expanded") === "true";
      if (expanded) {
        state.expandedTreeKeys.delete(key);
        state.collapsedTreeKeys.add(key);
      } else {
        state.expandedTreeKeys.add(key);
        state.collapsedTreeKeys.delete(key);
      }
      renderDeveloperPage();
    });
  });
}

function renderDeveloperWebSSHAssetTree(assets) {
  if (!assets.length) {
    return `<div class="empty">当前环境没有可申请 WebSSH 的 EC2 资产。</div>`;
  }
  const groups = groupDeveloperAssetsByProject(assets);
  return groups.map((group) => {
    const treeKey = `developer:webssh:project:${group.project}`;
    const expanded = isTreeKeyExpanded(treeKey, true);
    const authorizedCount = group.assets.filter((asset) => activeAccessGrant("webssh", "asset", asset.id)).length;
    return `
      <section class="developer-asset-tree-project">
        <button type="button" class="developer-asset-tree-head" data-dev-webssh-tree-key="${escapeHTML(treeKey)}" aria-expanded="${expanded}">
          <span class="tree-caret">${expanded ? "-" : "+"}</span>
          <span class="developer-asset-tree-title">
            <span class="tree-node-type">项目</span>
            <strong>${escapeHTML(projectLabel(group.project))}</strong>
          </span>
          <span class="developer-asset-tree-count">
            ${authorizedCount ? `<span class="pill success">${escapeHTML(String(authorizedCount))} 台可登录</span>` : ""}
            <span class="pill">${escapeHTML(String(group.assets.length))} 台 EC2</span>
          </span>
        </button>
        <div class="developer-asset-tree-items ${expanded ? "" : "hidden"}">
          ${group.assets.map((asset) => renderDeveloperWebSSHAssetNode(asset)).join("")}
        </div>
      </section>
    `;
  }).join("");
}

function groupDeveloperAssetsByProject(assets) {
  const groups = new Map();
  assets.forEach((asset) => {
    const project = asset.project_code || "public";
    if (!groups.has(project)) {
      groups.set(project, []);
    }
    groups.get(project).push(asset);
  });
  return Array.from(groups.entries())
    .map(([project, items]) => ({
      project,
      assets: sortDeveloperWebSSHAssets(items)
    }))
    .sort((left, right) => projectLabel(left.project).localeCompare(projectLabel(right.project)));
}

function sortDeveloperWebSSHAssets(assets) {
  return [...assets].sort((left, right) => {
    const leftAuthorized = activeAccessGrant("webssh", "asset", left.id) ? 1 : 0;
    const rightAuthorized = activeAccessGrant("webssh", "asset", right.id) ? 1 : 0;
    if (leftAuthorized !== rightAuthorized) {
      return rightAuthorized - leftAuthorized;
    }
    const leftStatus = left.status === "active" ? 1 : 0;
    const rightStatus = right.status === "active" ? 1 : 0;
    if (leftStatus !== rightStatus) {
      return rightStatus - leftStatus;
    }
    return String(left.name || "").localeCompare(String(right.name || ""));
  });
}

function renderDeveloperWebSSHAssetNode(asset) {
  const grant = activeAccessGrant("webssh", "asset", asset.id);
  return `
    <article class="developer-asset-tree-node ${grant ? "is-authorized" : ""}">
      <div class="developer-asset-node-main">
        <button type="button" class="developer-asset-name" data-select-asset="${escapeHTML(asset.id)}">${escapeHTML(asset.name || asset.id)}</button>
        <span class="change-meta">${escapeHTML(asset.cloud_account_name || asset.platform_name || asset.platform_code || "-")} / ${escapeHTML(asset.region || "-")} / ${escapeHTML(asset.owner || "-")}</span>
      </div>
      <div class="developer-asset-node-meta">
        <span class="status-badge status-${escapeHTML(asset.status || "-")}">${escapeHTML(asset.status || "-")}</span>
        ${renderDeveloperAssetPrimaryAction(asset)}
      </div>
    </article>
  `;
}

function renderDeveloperAssetPrimaryAction(asset) {
  if (!isWebSSHAsset(asset)) {
    return `<button type="button" data-request-tool="${escapeHTML(asset.id)}" data-request-env="${escapeHTML(asset.environment)}" data-request-kind="credential">申请访问</button>`;
  }
  const grant = activeAccessGrant("webssh", "asset", asset.id);
  if (grant) {
    return `<button type="button" class="primary" data-open-webssh="${escapeHTML(asset.id)}">登录 WebSSH</button><span class="pill">授权至 ${escapeHTML(formatDateTime(grant.expires_at))}</span>`;
  }
  return `<button type="button" data-request-tool="${escapeHTML(asset.id)}" data-request-env="${escapeHTML(asset.environment)}" data-request-kind="webssh">申请 WebSSH</button>`;
}

function activeAccessGrant(action, targetType, targetId) {
  const now = Date.now();
  return (state.accessGrants || []).find((grant) => {
    const expiresAt = new Date(grant.expires_at).getTime();
    return grant.status === "active"
      && grant.action === action
      && grant.target_type === targetType
      && grant.target_id === targetId
      && !Number.isNaN(expiresAt)
      && expiresAt > now;
  }) || null;
}

function renderDeveloperEntryCard(tool, envCode, openLabel) {
  const credential = credentialForOwner("asset", tool.asset_id);
  const credentialGrant = credential ? activeAccessGrant("credential", credential.owner_type, credential.owner_id) || activeAccessGrant("credential", "credential", credential.id) : null;
  const credentialButton = credential
    ? `<button type="button" data-reveal-dev-credential="${escapeHTML(credential.id)}">${credentialGrant || credential.access_policy === "viewable" ? "查看凭证" : "申请凭证"}</button>`
    : `<button type="button" data-request-tool="${escapeHTML(tool.asset_id)}" data-request-env="${escapeHTML(tool.environment === "global" ? envCode : tool.environment)}" data-request-kind="credential">${tool.approval_required ? "申请访问" : "获取凭证"}</button>`;
  return `
    <article class="developer-tool-card">
      <div class="section-head">
        <div>
          <h3>${escapeHTML(tool.asset_name)}</h3>
          <p class="muted">${escapeHTML(tool.tool_type)} / ${escapeHTML(renderToolScope(tool.environment))} / ${escapeHTML(tool.owner || "-")}</p>
        </div>
        <span class="status-badge status-${escapeHTML(tool.status)}">${escapeHTML(tool.status)}</span>
      </div>
      <a href="${escapeHTML(tool.endpoint)}" target="_blank" rel="noreferrer">${escapeHTML(tool.endpoint)}</a>
      <div class="developer-action-row">
        <button type="button" data-open-tool="${escapeHTML(tool.endpoint)}">${escapeHTML(openLabel)}</button>
        ${credentialButton}
      </div>
    </article>
  `;
}

function credentialForOwner(ownerType, ownerId) {
  return (state.credentials || []).find((credential) => credential.owner_type === ownerType && credential.owner_id === ownerId) || null;
}

function renderDeveloperMetric(label, value, hint) {
  return `
    <article class="developer-metric">
      <span>${escapeHTML(label)}</span>
      <strong>${escapeHTML(value)}</strong>
      <small>${escapeHTML(hint)}</small>
    </article>
  `;
}

function openApprovalModal(mode = "request") {
  state.approvalModalMode = mode;
  refs.approvalModalTitle.textContent = mode === "inbox" ? "审批待办" : "提交审批申请";
  refs.approvalModalHint.textContent = mode === "inbox" ? "处理待审批消息，或补充新的访问申请。" : "填写用途、权限和有效期后提交。";
  if (state.currentUser) {
    refs.approvalRequester.value = state.currentUser.username;
  }
  refs.approvalModal.classList.remove("hidden");
  refs.approvalReason.focus();
}

function closeApprovalModal() {
  refs.approvalModal.classList.add("hidden");
  state.approvalModalMode = "request";
}

function openDeveloperAssetModal(assetID) {
  const asset = state.assets.find((item) => item.id === assetID);
  if (!asset) {
    showMessage("资产不存在或当前无权查看。", "error");
    return;
  }
  state.developerSelectedAssetId = assetID;
  renderDeveloperAssetModal(asset);
  refs.developerAssetModal.classList.remove("hidden");
}

function closeDeveloperAssetModal() {
  refs.developerAssetModal.classList.add("hidden");
}

function renderDeveloperAssetModal(asset) {
  const relatedChanges = state.changes.filter((item) => item.asset_id === asset.id).slice(0, 4);
  const relatedProbes = state.probes.filter((item) => item.asset_id === asset.id).slice(0, 5);
  const latestProbe = relatedProbes[0];
  const specs = Object.entries(asset.specs || {}).slice(0, 12);
  const detailItems = [
    ["平台", asset.platform_name || asset.platform_code || "-"],
    ["云账号", asset.cloud_account_name || "-"],
    ["账号 ID", asset.account_id || "-"],
    ["环境", asset.environment || "-"],
    ["资源类型", asset.resource_type || "-"],
    ["地域", asset.region || "-"],
    ["负责人", asset.owner || "-"],
    ["状态", asset.status || "-"],
    ["重要级别", asset.criticality || "-"],
    ["入口", asset.endpoint || "-"],
    ["最近拨测", latestProbe ? `${latestProbe.status} / ${latestProbe.latency_ms || 0} ms / ${formatDateTime(latestProbe.checked_at)}` : "未拨测"],
    ["最近检查", asset.last_checked_at || "-"]
  ];
  refs.developerAssetModalTitle.textContent = asset.name || "资产详情";
  refs.developerAssetModalHint.textContent = `${asset.resource_type || "-"} / ${asset.environment || "-"} / ${asset.status || "-"}`;
  refs.developerAssetModalBody.innerHTML = `
    ${renderInspectionCards(renderDeveloperAssetCards(asset, latestProbe), "asset-inspection-grid")}
    <section class="asset-spec-section">
      <h4>基础信息</h4>
      <div class="asset-spec-grid">
        ${detailItems.map(([label, value]) => `
          <div class="asset-spec-item">
            <span class="asset-spec-label">${escapeHTML(label)}</span>
            <span class="asset-spec-value">${escapeHTML(value)}</span>
          </div>
        `).join("")}
      </div>
    </section>
    <section class="asset-spec-section">
      <h4>规格细项</h4>
      ${specs.length ? `
        <div class="asset-spec-grid">
          ${specs.map(([label, value]) => `
            <div class="asset-spec-item">
              <span class="asset-spec-label">${escapeHTML(formatSpecLabel(label))}</span>
              <span class="asset-spec-value">${escapeHTML(value)}</span>
            </div>
          `).join("")}
        </div>
      ` : `<div class="empty">当前资产暂无结构化规格细项。</div>`}
    </section>
    <section class="asset-spec-section">
      <h4>最近变更</h4>
      ${relatedChanges.length ? `
        <div class="asset-related-list">
          ${relatedChanges.map((change) => `
            <article class="change-card">
              <div class="section-head">
                <p class="change-title">${escapeHTML(change.title)}</p>
                <span class="status-badge status-${escapeHTML(change.status)}">${escapeHTML(change.status)}</span>
              </div>
              <span class="change-meta">${escapeHTML(change.executor || "-")} / ${escapeHTML(change.window || "-")}</span>
              <p class="change-summary">${escapeHTML(change.summary || "-")}</p>
            </article>
          `).join("")}
        </div>
      ` : `<div class="empty">暂无关联变更。</div>`}
    </section>
  `;
}

function renderDeveloperAssetCards(asset, latestProbe) {
  return [
    ["资产状态", asset.status || "-", asset.status === "active" ? "ok" : "warn"],
    ["重要级别", asset.criticality || "-", asset.criticality === "high" ? "warn" : "ok"],
    ["拨测状态", latestProbe ? latestProbe.status : "未拨测", latestProbe && latestProbe.status !== "up" ? "danger" : latestProbe ? "ok" : "warn"],
    ["响应耗时", latestProbe && latestProbe.latency_ms ? `${latestProbe.latency_ms} ms` : "-", latestProbe && latestProbe.latency_ms > 3000 ? "warn" : "ok"]
  ];
}

function canDecideApprovals() {
  return ["admin", "ops", "lead"].includes(roleOfCurrentUser());
}

function pendingApprovalTask(item) {
  return (item.tasks || []).find((task) => task.status === "pending") || null;
}

function canDecideApproval(item) {
  if (!canDecideApprovals() || item.status !== "pending") {
    return false;
  }
  const task = pendingApprovalTask(item);
  if (!task) {
    return true;
  }
  const role = roleOfCurrentUser();
  return role === "admin" || role === task.approver_role;
}

function renderApprovalStep(item) {
  const task = pendingApprovalTask(item);
  if (!task) {
    return item.flow_id ? "流程待确认" : "旧申请";
  }
  return `${task.approver_label || task.approver_role} / 第 ${task.step_order} 步`;
}

function renderToolScope(environment) {
  return environment === "global" ? "全局工具" : (environment || "-");
}

function isWebSSHAsset(asset) {
  return String(asset.platform_code || "").toLowerCase() === "aws"
    && String(asset.resource_type || "").toLowerCase() === "ec2";
}

function renderApprovals() {
  const items = approvalsForCurrentModal();
  refs.approvalList.innerHTML = items.length ? items.map((item) => `
    <article class="change-card">
      <div class="section-head">
        <p class="change-title">${escapeHTML(item.request_type)} / ${escapeHTML(item.target_name || item.target_id || "-")}</p>
        <span class="status-badge status-${item.status === "approved" ? "done" : item.status === "rejected" ? "cancelled" : "planned"}">${escapeHTML(item.status)}</span>
      </div>
      <span class="change-meta">${escapeHTML(item.requester)} / ${escapeHTML(item.environment)} / ${escapeHTML(String(item.duration_minutes))} 分钟 / ${escapeHTML(renderApprovalStep(item))}</span>
      <p class="change-summary">${escapeHTML(item.reason || "-")}</p>
      ${canDecideApproval(item) ? `
        <div class="developer-action-row">
          <button type="button" data-approval-id="${escapeHTML(item.id)}" data-approval-decision="approved">批准</button>
          <button type="button" data-approval-id="${escapeHTML(item.id)}" data-approval-decision="rejected">拒绝</button>
        </div>
      ` : ""}
    </article>
  `).join("") : `<div class="empty">暂无审批申请。</div>`;
  refs.approvalList.querySelectorAll("[data-approval-id]").forEach((button) => {
    button.addEventListener("click", async () => {
      await apiFetch(`/api/approvals/${button.dataset.approvalId}/decision`, "POST", {
        status: button.dataset.approvalDecision,
        decision_summary: button.dataset.approvalDecision === "approved" ? "同意本次临时访问" : "拒绝本次临时访问"
      });
      await loadData("审批状态已更新。", "success");
    });
  });
}

function approvalsForCurrentModal() {
  if (state.approvalModalMode === "inbox") {
    return state.approvals.filter((item) => item.status === "pending" && canDecideApproval(item));
  }
  return [...state.approvals].sort((left, right) => {
    const leftPending = left.status === "pending" ? 1 : 0;
    const rightPending = right.status === "pending" ? 1 : 0;
    if (leftPending !== rightPending) {
      return rightPending - leftPending;
    }
    return String(right.created_at || "").localeCompare(String(left.created_at || ""));
  }).slice(0, 10);
}
