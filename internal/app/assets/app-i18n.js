const i18nMessages = {
  zh: {
    "app.title": "云平台运维台账",
    "app.subtitle": "按云平台、账号和资源类型管理运维现场。",
    "login.subtitle": "使用本地账号登录后进入运维台账和开发者工作台。",
    "login.username": "用户名",
    "login.password": "密码",
    "login.submit": "登录",
    "login.seedHint": "开发种子账号需显式启用并设置本地密码；公开部署请接入正式身份源。",
    "setup.title": "首次部署初始化",
    "setup.username": "管理员账号",
    "setup.displayName": "显示名称",
    "setup.email": "邮箱",
    "setup.password": "管理员密码",
    "setup.confirmPassword": "确认密码",
    "setup.submit": "完成初始化",
    "setup.hint": "数据库表已自动初始化。这里会创建第一个平台管理员账号，不会生成默认弱口令。",
    "nav.developer": "开发者视角",
    "nav.audit": "审计",
    "nav.refresh": "刷新",
    "nav.logout": "退出",
    "nav.configTitle": "打开配置页",
    "workbench.summaryTitle": "资产总览",
    "workbench.treePath": "项目 -> 云平台 -> 账号 -> Zone / 类型 -> 资产",
    "workbench.searchPlaceholder": "搜索平台 / 云账号 / 资源 / 负责人 / 标签",
    "config.syncPolicy": "同步策略",
    "config.syncMode": "同步模式",
    "config.syncPeriod": "同步周期",
    "config.autoSync": "启用自动同步",
    "unit.year": "年",
    "unit.month": "月",
    "unit.week": "周",
    "unit.day": "日",
    "unit.hour": "时",
    "unit.minute": "分",
    "unit.second": "秒",
    "sync.manual": "手动",
    "sync.scheduled": "定时",
    "message.loggedOut": "已退出登录。",
    "message.loginLoading": "登录中...",
    "message.login": "登录",
    "message.lastUpdated": "最近刷新: "
  },
  en: {
    "app.title": "Cloud Ops Ledger",
    "app.subtitle": "Manage operations by cloud platform, account, and resource type.",
    "login.subtitle": "Sign in with a local account to access the ops and developer workspaces.",
    "login.username": "Username",
    "login.password": "Password",
    "login.submit": "Sign In",
    "login.seedHint": "Development seed users require explicit local passwords. Public deployments should use a real identity provider.",
    "setup.title": "First Deployment Setup",
    "setup.username": "Admin Username",
    "setup.displayName": "Display Name",
    "setup.email": "Email",
    "setup.password": "Admin Password",
    "setup.confirmPassword": "Confirm Password",
    "setup.submit": "Complete Setup",
    "setup.hint": "Database tables are initialized automatically. This creates the first platform administrator without a default weak password.",
    "nav.developer": "Developer",
    "nav.audit": "Audit",
    "nav.refresh": "Refresh",
    "nav.logout": "Logout",
    "nav.configTitle": "Open settings",
    "workbench.summaryTitle": "Asset Overview",
    "workbench.treePath": "Project -> Platform -> Account -> Zone / Type -> Asset",
    "workbench.searchPlaceholder": "Search platform / account / asset / owner / tag",
    "config.syncPolicy": "Sync Policy",
    "config.syncMode": "Sync Mode",
    "config.syncPeriod": "Sync Period",
    "config.autoSync": "Enable auto sync",
    "unit.year": "Y",
    "unit.month": "M",
    "unit.week": "W",
    "unit.day": "D",
    "unit.hour": "h",
    "unit.minute": "m",
    "unit.second": "s",
    "sync.manual": "Manual",
    "sync.scheduled": "Scheduled",
    "message.loggedOut": "Logged out.",
    "message.loginLoading": "Signing in...",
    "message.login": "Sign In",
    "message.lastUpdated": "Last refresh: "
  }
};

const i18nTextTranslations = {
  "云平台运维台账": "Cloud Ops Ledger",
  "使用本地账号登录后进入运维台账和开发者工作台。": "Sign in with a local account to access the ops and developer workspaces.",
  "按云平台、账号和资源类型管理运维现场。": "Manage operations by cloud platform, account, and resource type.",
  "开发种子账号需显式启用并设置本地密码；公开部署请接入正式身份源。": "Development seed users require explicit local passwords. Public deployments should use a real identity provider.",
  "用户名": "Username",
  "密码": "Password",
  "登录": "Sign In",
  "首次部署初始化": "First Deployment Setup",
  "管理员账号": "Admin Username",
  "显示名称": "Display Name",
  "管理员密码": "Admin Password",
  "确认密码": "Confirm Password",
  "完成初始化": "Complete Setup",
  "数据库表已自动初始化。这里会创建第一个平台管理员账号，不会生成默认弱口令。": "Database tables are initialized automatically. This creates the first platform administrator without a default weak password.",
  "首次部署时先完成数据库检查和管理员账号初始化。": "Complete database checks and administrator initialization before first use.",
  "使用本地账号登录后进入运维台账和开发者工作台。": "Sign in with a local account to access the ops and developer workspaces.",
  "数据库已初始化": "Database initialized",
  "正在检查数据库初始化状态": "Checking database initialization status",
  "初始化中...": "Initializing...",
  "初始化完成。": "Setup completed.",
  "初始化完成，请使用管理员账号登录。": "Setup completed. Sign in with the administrator account.",
  "登录中...": "Signing in...",
  "退出": "Logout",
  "刷新": "Refresh",
  "审计": "Audit",
  "开发者视角": "Developer",
  "打开配置页": "Open settings",
  "资产总览": "Asset Overview",
  "项目 -> 云平台 -> 账号 -> Zone / 类型 -> 资产": "Project -> Platform -> Account -> Zone / Type -> Asset",
  "搜索平台 / 云账号 / 资源 / 负责人 / 标签": "Search platform / account / asset / owner / tag",
  "全部状态": "All statuses",
  "全部项目": "All projects",
  "全部云账号": "All accounts",
  "全部资源类型": "All resource types",
  "全部动作": "All actions",
  "全部结果": "All outcomes",
  "资产树": "Asset Tree",
  "按账号：云平台 -> 账号 -> 项目 -> 类型；按项目：项目 -> 云平台 -> 账号 -> 类型": "By account: Platform -> Account -> Project -> Type; by project: Project -> Platform -> Account -> Type",
  "资产树分类方式": "Asset tree grouping",
  "按账号": "By Account",
  "按项目": "By Project",
  "资产详情": "Asset Details",
  "查看规格、同步、巡检与变更概况": "View specs, sync, inspection, and change status",
  "请选择左侧云账号或资产查看详情。": "Select an account or asset on the left to view details.",
  "请选择左侧云账号查看账号详情。": "Select an account on the left to view account details.",
  "没有匹配的资产记录。": "No matching assets.",
  "平台": "Platform",
  "账号": "Account",
  "项目": "Project",
  "类型": "Type",
  "域名": "Domain",
  "公共资源": "Public",
  "总资产": "Total Assets",
  "运行中": "Running",
  "维护中": "Maintenance",
  "高重要级别": "High Criticality",
  "拨测异常": "Probe Alerts",
  "未处理告警": "Open Alerts",
  "工具资产": "Tool Assets",
  "待审批": "Pending Approvals",
  "待执行变更": "Planned Changes",
  "执行中变更": "Changes In Progress",
  "云账号数": "Accounts",
  "配置中心": "Settings",
  "云账号、凭证、同步、资产维护与运维记录集中配置。": "Manage cloud accounts, credentials, sync, assets, and ops records.",
  "配置页导航": "Settings navigation",
  "云账号": "Cloud Accounts",
  "工具凭证": "Tools & Credentials",
  "权限流程": "Permissions & Flows",
  "巡检": "Inspections",
  "同步任务": "Sync Jobs",
  "标签分账": "Tags & Chargeback",
  "资产变更": "Assets & Changes",
  "关闭": "Close",
  "云账号管理": "Cloud Account Management",
  "平台 -> 云账号 -> 资产。AWS 填 AK/SK；Cloudflare 填 API Token。": "Platform -> Account -> Assets. Use AK/SK for AWS; API Token for Cloudflare.",
  "收起": "Collapse",
  "展开": "Expand",
  "新建": "New",
  "基础信息": "Basic Info",
  "云账号名称": "Account Name",
  "账号 ID / 标识": "Account ID / Identifier",
  "默认 Region": "Default Region",
  "环境": "Environment",
  "负责人": "Owner",
  "重要级别": "Criticality",
  "凭证": "Credential",
  "已配置": "Configured",
  "缺失": "Missing",
  "异常": "Abnormal",
  "正常": "Normal",
  "本月费用": "Current Month Cost",
  "高危资产": "High-risk Assets",
  "自动汇总": "Automatic Summary",
  "配置完整性": "Configuration Completeness",
  "完整": "Complete",
  "大类": "Category",
  "账号 ID": "Account ID",
  "来源": "Source",
  "外部 ID": "External ID",
  "概览": "Overview",
  "规格": "Specs",
  "个资产": "assets",
  "AWS 使用 Access Key / Secret；Cloudflare 在 AK / API Token 中填写 API Token，Secret 可留空。": "Use Access Key / Secret for AWS. For Cloudflare, put the API Token in AK / API Token and leave Secret empty.",
  "Cloudflare 可留空": "Optional for Cloudflare",
  "同步策略": "Sync Policy",
  "同步模式": "Sync Mode",
  "同步周期": "Sync Period",
  "启用自动同步": "Enable auto sync",
  "年": "Y",
  "月": "M",
  "周": "W",
  "日": "D",
  "时": "h",
  "分": "m",
  "秒": "s",
  "保存云账号": "Save Account",
  "立即同步": "Sync Now",
  "同步费用": "Sync Cost",
  "新增资产": "New Asset",
  "编辑资产": "Edit Asset",
  "支持手工维护非云发现资产。": "Supports manually maintained non-cloud-discovered assets.",
  "导出": "Export",
  "导入": "Import",
  "平台代码": "Platform Code",
  "平台名称": "Platform Name",
  "项目维度": "Project",
  "分类": "Category",
  "资源类型": "Resource Type",
  "地域 / 可用区": "Region / AZ",
  "资源名称": "Resource Name",
  "入口 / 地址": "Entry / Address",
  "状态": "Status",
  "最近巡检": "Last Inspection",
  "标签": "Tags",
  "备注": "Notes",
  "保存资产": "Save Asset",
  "删除资产": "Delete Asset",
  "批量编辑": "Bulk Edit",
  "按当前资产树筛选条件批量更新资产治理字段；留空字段不会修改。": "Bulk-update governance fields by current asset-tree filters. Empty fields are not changed.",
  "不修改": "No change",
  "不修改；逗号分隔": "No change; comma-separated",
  "批量更新当前筛选资产": "Bulk Update Filtered Assets",
  "新增工具资产": "New Tool Asset",
  "编辑工具资产": "Edit Tool Asset",
  "工具资产列表": "Tool Asset List",
  "点击工具可回填编辑，并同步出现在开发者工作台。": "Click a tool to fill the edit form. It also appears in the developer workspace.",
  "工具由运维配置，同时作为 tool 类资产进入资产树和开发者工作台。": "Tools are configured by ops and also appear as tool assets in the tree and developer workspace.",
  "归属": "Scope",
  "全局工具": "Global Tool",
  "工具类型": "Tool Type",
  "工具名称": "Tool Name",
  "入口 URL": "Entry URL",
  "登录策略": "Login Policy",
  "凭证策略": "Credential Policy",
  "敏感操作需要审批": "Sensitive actions require approval",
  "支持 WebSSH": "Supports WebSSH",
  "工具资产只维护入口和凭证策略；WebSSH 请在 AWS EC2 资产上申请。": "Tool assets maintain only entry points and credential policy. Request WebSSH on AWS EC2 assets.",
  "webssh（历史选项，WebSSH 请用 EC2 资产）": "webssh (legacy option; use EC2 assets for WebSSH)",
  "说明": "Description",
  "保存工具资产": "Save Tool Asset",
  "凭证管理": "Credential Management",
  "给工具、资产或云账号维护受控凭证。保存后只展示脱敏值，查看和复制都会审计。": "Maintain controlled credentials for tools, assets, or cloud accounts. Only masked values are shown after saving; reveal and copy are audited.",
  "归属类型": "Owner Type",
  "归属对象": "Owner Object",
  "凭证类型": "Credential Type",
  "键名": "Key Name",
  "访问策略": "Access Policy",
  "凭证明文": "Credential Plaintext",
  "保存后不会回填明文": "Plaintext is not returned after saving",
  "保存凭证": "Save Credential",
  "凭证列表": "Credential List",
  "只显示脱敏值，查看和复制都会写审计。": "Only masked values are shown. Reveal and copy actions are audited.",
  "云账号列表": "Cloud Account List",
  "点击云账号仅回填配置；详情在资产树点击云账号查看。": "Click an account to fill the config form. View details from the asset tree.",
  "资产。AWS 填 AK/SK；Cloudflare 填 API Token。": "Assets. Use AK/SK for AWS and API Token for Cloudflare.",
  "标签治理": "Tag Governance",
  "基于台账资产和云资源 tag 值识别 Project / Environment 缺口。": "Detect Project / Environment gaps from ledger assets and cloud resource tags.",
  "项目分账": "Project Chargeback",
  "只读估算：按云账号内非 stale 资产项目占比分摊账号费用。": "Read-only estimate: allocate account cost by non-stale asset project share within each cloud account.",
  "项目估算口径：按每个云账号内项目资产数量占比分摊。账号详情中的服务维度费用来自 AWS Cost Explorer 真实账单；项目精确分账后续接 Cost Allocation Tag / CUR。": "Project estimate method: allocate by project asset count within each cloud account. Service-level cost in account details comes from real AWS Cost Explorer bills; precise project chargeback will use Cost Allocation Tags / CUR later.",
  "用户列表": "Users",
  "本地用户与角色权限底座。": "Local users and role-permission foundation.",
  "用户权限": "User Permissions",
  "编辑用户": "Edit User",
  "新增变更": "New Change",
  "编辑变更": "Edit Change",
  "开发者工作台": "Developer Workspace",
  "按环境查看工具入口、账号密码、临时凭证和 EC2 WebSSH。": "View app entries, tools, credentials, temporary access, and EC2 WebSSH by environment.",
  "提交申请": "Submit Request",
  "返回运维台账": "Back to Ops Ledger",
  "应用入口": "Application Entries",
  "当前环境的业务应用入口，默认展开。": "Business app entries for the current environment. Expanded by default.",
  "工具入口": "Tool Entries",
  "全局运维工具入口，默认展开。": "Global ops tools. Expanded by default.",
  "资产列表": "Asset List",
  "资产": "Asset",
  "WebSSH 目标为 AWS EC2，敏感环境进入审批。": "WebSSH targets AWS EC2. Sensitive environments require approval.",
  "审批待办": "Approval Inbox",
  "处理待审批消息，或补充新的访问申请。": "Handle pending approvals or submit a new access request.",
  "提交审批申请": "Submit Approval Request",
  "填写用途、权限和有效期后提交。": "Fill in purpose, permission, and validity before submitting.",
  "审批与凭证": "Approval & Credentials",
  "查看、复制、登录、签发都会进入审计。": "Reveal, copy, login, and issuance actions are audited.",
  "申请人": "Requester",
  "目标": "Target",
  "权限级别": "Permission Level",
  "有效期分钟": "Duration (minutes)",
  "用途": "Purpose",
  "说明排障、发布或验证用途": "Describe troubleshooting, release, or validation purpose",
  "有审批权限的角色可在这里处理待办。": "Roles with approval permission can process tasks here.",
  "只读查看资产基础信息、规格和最近状态。": "Read-only asset details, specs, and recent status.",
  "审计工作台": "Audit Workspace",
  "按操作时间追踪登录、权限、审批、凭证、WebSSH、同步和告警处理留痕。": "Track login, permission, approval, credential, WebSSH, sync, and alert actions by time.",
  "最近 50 条随首页加载，接口最多返回 100 条": "Latest 50 records load with the page. API returns up to 100.",
  "审计日志": "Audit Log",
  "每条记录包含操作时间、操作人、动作、对象、结果、来源和摘要。": "Each record includes time, actor, action, object, result, source, and summary.",
  "搜索操作人 / 动作 / 目标 / 摘要 / IP": "Search actor / action / target / summary / IP",
  "审计范围": "Audit Scope",
  "当前已覆盖关键安全动作。": "Key security actions are covered.",
  "身份": "Identity",
  "登录、退出、越权拒绝": "Login, logout, denied access",
  "审批": "Approval",
  "申请提交、审批处理、短期授权": "Request submission, approval handling, temporary grants",
  "查看明文、复制凭证、保存凭证": "Reveal plaintext, copy credentials, save credentials",
  "访问": "Access",
  "WebSSH 打开、会话授权校验": "WebSSH open and session authorization checks",
  "运维": "Operations",
  "云同步、费用同步、告警处理、附件上传": "Cloud sync, cost sync, alert handling, attachment upload",
  "操作时间": "Time",
  "操作人": "Actor",
  "动作": "Action",
  "对象": "Object",
  "结果": "Result",
  "来源": "Source",
  "摘要": "Summary",
  "权限策略": "Permission Policies",
  "需要审批": "Approval Required",
  "保存权限": "Save Permission",
  "新建权限": "New Permission",
  "直接允许": "Direct Allow",
  "暂无权限策略。": "No permission policies.",
  "权限管理": "Permission Management",
  "配置用户、角色、角色权限和审批流程。": "Configure users, roles, role permissions, and approval flows.",
  "显示名": "Display Name",
  "邮箱": "Email",
  "手机号": "Phone",
  "角色": "Role",
  "团队": "Team",
  "留空则不修改": "Leave empty to keep unchanged",
  "保存用户": "Save User",
  "角色配置": "Role Configuration",
  "定义角色编码、名称和层级。": "Define role code, name, and level.",
  "角色编码": "Role Code",
  "角色名称": "Role Name",
  "层级": "Level",
  "保存角色": "Save Role",
  "新建角色": "New Role",
  "角色权限": "Role Permissions",
  "按角色、环境、资源范围和动作配置权限。": "Configure permissions by role, environment, resource scope, and action.",
  "项目范围": "Project Scope",
  "资源范围": "Resource Scope",
  "动作": "Action",
  "审批流程": "Approval Flows",
  "拖动步骤调整顺序，保存后作为审批编排定义。": "Drag steps to reorder them, then save as an approval flow definition.",
  "流程名称": "Flow Name",
  "生产凭证访问审批": "Production Credential Access Approval",
  "范围": "Scope",
  "新增步骤": "Add Step",
  "保存流程": "Save Flow",
  "新建流程": "New Flow",
  "暂无角色定义。": "No roles.",
  "暂无审批流程。": "No approval flows.",
  "审批角色": "Approver Role",
  "显示名称": "Display Name",
  "超时分钟": "Timeout Minutes",
  "删除": "Delete",
  "暂无流程步骤，先新增一个审批步骤。": "No flow steps. Add an approval step first.",
  "当前还没有云账号记录。": "No cloud accounts yet.",
  "未配置凭证": "Credential not configured",
  "配置索引：详情、费用、巡检和资产分布请在资产树点击云账号查看。": "Config index: click a cloud account in the asset tree to view details, cost, inspections, and asset distribution.",
  "当前还没有工具资产。": "No tool assets yet.",
  "审批": "Approval",
  "当前还没有凭证项。": "No credential items yet.",
  "凭证已复制，并已写入审计。": "Credential copied and audited.",
  "缺 Project": "Missing Project",
  "缺 Environment": "Missing Environment",
  "空标签值": "Empty Tag Value",
  "值不合规": "Invalid Tag Value",
  "环境 unknown": "Environment Unknown",
  "未发现 Project/ProjectCode/Biz/System/App 标签，项目归属只能依赖台账推断。": "No Project/ProjectCode/Biz/System/App tag was found. Project ownership can only be inferred from the ledger.",
  "未发现 Environment/Env/Stage/Profile 标签，环境归属只能依赖台账或名称推断。": "No Environment/Env/Stage/Profile tag was found. Environment can only be inferred from the ledger or name.",
  "混合账号资源未能从标签或名称判断环境，需要补 Environment 标签。": "Mixed-account resource environment cannot be inferred from tags or name. Add an Environment tag.",
  "查看": "Reveal",
  "复制": "Copy",
  "当前资产标签治理项为空。重新同步 AWS 后可在这里复核真实 tag 值。": "No tag-governance findings yet. Resync AWS to review real tag values here.",
  "无标签": "No Tags",
  "项目数": "Projects",
  "纳入资产": "Included Assets",
  "本月项目估算": "Current Project Estimate",
  "预计项目估算": "Forecast Project Estimate",
  "费用账号": "Billing Accounts",
  "当前没有可分摊的云账号费用。请先同步 AWS 费用和资产标签。": "No cloud account cost is available for chargeback. Sync AWS cost and asset tags first.",
  "当前还没有用户。": "No users yet.",
  "请求失败": "Request failed",
  "已恢复登录状态。": "Session restored.",
  "已退出登录。": "Logged out.",
  "已刷新台账数据。": "Ledger data refreshed.",
  "当前角色不能进入配置中心。": "Current role cannot access settings.",
  "当前角色不能进入审计工作台。": "Current role cannot access the audit workspace.",
  "当前角色不能进入开发者工作台。": "Current role cannot access the developer workspace.",
  "请先选择云账号。": "Select a cloud account first.",
  "云账号已更新。": "Cloud account updated.",
  "云账号已创建。": "Cloud account created.",
  "导入文件不是有效 JSON。": "Import file is not valid JSON.",
  "工具资产已更新。": "Tool asset updated.",
  "工具资产已创建。": "Tool asset created.",
  "凭证已保存，明文未回填。": "Credential saved. Plaintext was not returned.",
  "用户已更新。": "User updated.",
  "用户已创建。": "User created.",
  "角色已更新。": "Role updated.",
  "角色已创建。": "Role created.",
  "权限策略已更新。": "Permission policy updated.",
  "权限策略已创建。": "Permission policy created.",
  "审批流程已更新。": "Approval flow updated.",
  "审批流程已创建。": "Approval flow created.",
  "审批申请已提交。": "Approval request submitted.",
  "请先选择一个资产。": "Select an asset first.",
  "巡检记录已创建。": "Inspection record created.",
  "资产已更新。": "Asset updated.",
  "资产已创建。": "Asset created.",
  "当前筛选条件下没有可更新资产。": "No assets can be updated under the current filters.",
  "请至少填写一个批量更新字段。": "Fill at least one bulk update field.",
  "删除资产会同步删除关联变更，继续吗？": "Deleting this asset also deletes related changes. Continue?",
  "资产已删除。": "Asset deleted.",
  "变更已更新。": "Change updated.",
  "变更已创建。": "Change created.",
  "确认删除这条变更记录吗？": "Delete this change record?",
  "变更已删除。": "Change deleted.",
  "登录成功。": "Signed in.",
  "未配置邮箱": "Email not configured",
  "未登录": "Never logged in",
  "开发者": "Developer",
  "开发负责人": "Development Lead",
  "运维工程师": "Ops Engineer",
  "平台管理员": "Platform Admin",
  "只读观察员": "Read-only Viewer",
  "审计员": "Auditor",
  "管理系统配置、权限和高风险审批": "Manage system settings, permissions, and high-risk approvals",
  "维护云账号、资产、工具和审批": "Maintain cloud accounts, assets, tools, and approvals",
  "处理开发和测试环境的团队审批": "Handle team approvals for development and testing environments",
  "使用环境入口、申请凭证和 WebSSH": "Use environment entries, request credentials, and WebSSH",
  "查看台账和运行状态": "View ledger and runtime status",
  "查看审批、审计和变更留痕": "View approval, audit, and change records",
  "开发 WebSSH 审批": "Development WebSSH Approval",
  "开发环境 WebSSH 由开发负责人审批。": "Development WebSSH access is approved by the development lead.",
  "生产环境凭证查看先由运维审批，高风险再由管理员确认。": "Production credential reveal is first approved by ops, then confirmed by an admin for high-risk access.",
  "全部环境": "All Environments",
  "当前没有告警记录。": "No alert records.",
  "告警记录": "Alert Records",
  "自动拨测和巡检规则生成，恢复后可自动关闭；未恢复的告警由运维处理留痕。": "Generated by automatic probes and inspection rules. Recovered alerts close automatically; unresolved alerts are handled by ops with records.",
  "告警已处理。": "Alert resolved.",
  "巡检由系统自动拨测、同步检查和资产健康规则生成，配置页不再手工录入。": "Inspections are generated by automatic probes, sync checks, and asset health rules. Manual entry is disabled here.",
  "巡检由后台自动拨测、同步检查和资产规则生成；这里仅保留当前口径说明。": "Inspections are generated by background probes, sync checks, and asset rules. This area only keeps the current policy note.",
  "巡检资产": "Inspected Asset",
  "巡检结果": "Inspection Result",
  "巡检时间": "Inspection Time",
  "巡检摘要": "Inspection Summary",
  "自动生成，无需保存": "Generated automatically; no save required",
  "告警": "Alert",
  "无摘要": "No summary",
  "查看资产": "View Asset",
  "处理告警": "Resolve Alert",
  "处理结论": "Resolution",
  "已确认并处理": "Confirmed and resolved",
  "当前还没有同步记录。": "No sync records yet.",
  "等待调度": "Waiting for schedule",
  "即将同步": "Sync soon",
  "同步时间待确认": "Sync schedule pending",
  "未启用自动同步": "Auto sync disabled",
  "当前筛选下没有审计事件。": "No audit events match the current filters.",
  "查看所属云账号": "View Account",
  "发起变更": "Create Change",
  "最近变更": "Recent Changes",
  "暂无关联变更。": "No related changes.",
  "关联资产": "Related Asset",
  "标题": "Title",
  "类别": "Category",
  "执行人": "Executor",
  "风险级别": "Risk Level",
  "执行窗口": "Execution Window",
  "回滚方案": "Rollback Plan",
  "执行摘要": "Execution Summary",
  "保存变更": "Save Change",
  "删除变更": "Delete Change",
  "记录执行窗口、风险级别和回滚方案。": "Record execution window, risk level, and rollback plan.",
  "已回填该变更到配置页表单。": "Filled this change into the settings form.",
  "当前资产没有归属云账号。": "This asset has no linked cloud account.",
  "上一条": "Previous",
  "下一条": "Next",
  "所属云账号": "Account",
  "最近同步": "Latest Sync",
  "最近巡检": "Latest Inspection",
  "最近检查": "Latest Check",
  "关联变更": "Related Changes",
  "同类资源": "Peer Assets",
  "未同步": "Not Synced",
  "无同步记录": "No sync records",
  "未巡检": "Not Inspected",
  "无拨测记录": "No probe records",
  "可在下方查看详情": "See details below",
  "暂无变更": "No changes",
  "未配置": "Not Configured",
  "自动同步": "Auto Sync",
  "启用": "Enabled",
  "未启用": "Disabled",
  "同步计划": "Sync Plan",
  "下次自动同步": "Next Auto Sync",
  "同步状态": "Sync Status",
  "费用同步": "Cost Sync",
  "编辑配置": "Edit Config",
  "同步资产": "Sync Assets",
  "上月费用": "Last Month Cost",
  "上月整月": "Last Full Month",
  "上月同进度": "Last Month To Date",
  "本月当前": "Current Month",
  "本月当前使用": "Current Month Usage",
  "本月预计": "Forecast This Month",
  "预计本月": "Forecast This Month",
  "同进度差额": "Same-progress Delta",
  "资产类型分布": "Asset Type Distribution",
  "当前云账号没有资产。": "This account has no assets.",
  "自动巡检异常": "Automatic Inspection Alerts",
  "自动巡检": "Automatic Inspection",
  "当前账号没有自动巡检异常。": "This account has no automatic inspection alerts.",
  "最近同步留痕": "Recent Sync Records",
  "当前账号还没有同步记录。": "This account has no sync records.",
  "AWS Cost Explorer 真实费用": "Real AWS Cost Explorer Cost",
  "服务维度与每日快照来自 AWS 账单接口，不按资产数量估值。": "Service costs and daily snapshots come from AWS billing APIs, not asset-count estimates.",
  "暂无日快照": "No daily snapshot",
  "本月服务费用": "Current Month Service Cost",
  "请先同步费用，或确认账号已开通 Cost Explorer。": "Sync cost first, or confirm Cost Explorer is enabled.",
  "最近每日记录": "Recent Daily Records",
  "展示最近的云账号同步留痕。": "Show recent cloud account sync records.",
  "暂无每日费用快照。": "No daily cost snapshots.",
  "同步中...": "Syncing...",
  "拨测中...": "Probing...",
  "手动补测": "Manual Probe",
  "规格细项": "Spec Details",
  "当前资产暂无结构化规格细项。": "This asset has no structured spec details.",
  "当前资产还没有关联变更记录。": "This asset has no related change records.",
  "当前资产所属云账号还没有同步记录。": "This asset account has no sync records.",
  "最近拨测状态": "Recent Probe Status",
  "最近拨测": "Latest Probe",
  "暂无拨测数据": "No probe data",
  "暂无真实拨测记录，后台自动拨测任务会写入第一条记录，也可以点击“手动补测”。": "No real probe records yet. Automatic background probes will create the first record; you can also click Manual Probe.",
  "巡检记录": "Inspection Record",
  "附件": "Attachments",
  "上传": "Upload",
  "巡检附件已上传。": "Inspection attachment uploaded.",
  "暂无附件": "No attachments",
  "未拨测": "Not Probed",
  "拨测时延折线图": "Probe latency chart",
  "应用入口": "Application Entries",
  "当前环境": "Current Environment",
  "工具入口": "Tool Entries",
  "可访问资产": "Accessible Assets",
  "EC2 可申请": "EC2 Requestable",
  "打开应用": "Open App",
  "打开工具": "Open Tool",
  "当前环境还没有配置应用入口。": "No application entries are configured for this environment.",
  "还没有配置全局工具入口。": "No global tool entries are configured.",
  "当前环境没有可申请 WebSSH 的 EC2 资产。": "No EC2 assets are available for WebSSH requests in this environment.",
  "台可登录": "authorized",
  "台 EC2": "EC2",
  "申请访问": "Request Access",
  "登录 WebSSH": "Open WebSSH",
  "申请 WebSSH": "Request WebSSH",
  "查看凭证": "Reveal Credential",
  "申请凭证": "Request Credential",
  "获取凭证": "Get Credential",
  "资产不存在或当前无权查看。": "Asset does not exist or you do not have permission to view it.",
  "基础信息": "Basic Info",
  "地域": "Region",
  "入口": "Entry",
  "资产状态": "Asset Status",
  "拨测状态": "Probe Status",
  "响应耗时": "Response Time",
  "流程待确认": "Flow Pending",
  "旧申请": "Legacy Request",
  "批准": "Approve",
  "拒绝": "Reject",
  "暂无审批申请。": "No approval requests.",
  "同意本次临时访问": "Approve this temporary access",
  "拒绝本次临时访问": "Reject this temporary access",
  "审批状态已更新。": "Approval status updated.",
  "WebSSH 临时会话已创建。": "WebSSH temporary session created.",
  "开发环境": "Development",
  "生产环境": "Production",
  "测试环境": "Testing",
  "预发环境": "Staging",
  "到期未知": "Expiry unknown",
  "未归属 Zone": "Unassigned Zone",
  "未命名记录": "Unnamed record"
};

const i18nReverseTextTranslations = Object.fromEntries(
  Object.entries(i18nTextTranslations).map(([source, translated]) => [translated, source])
);

const i18nPatternTranslations = [
  [/^最近刷新: (.*)$/u, "Last refresh: $1"],
  [/^关联变更(.*)$/u, "Related Changes$1"],
  [/^巡检记录(.*)$/u, "Inspection Records$1"],
  [/^同步记录(.*)$/u, "Sync Records$1"],
  [/^当前筛选命中 (\d+) 条资产$/u, "Current filter matches $1 assets"],
  [/^有 (\d+) 条审批待办，点击处理$/u, "$1 pending approval(s). Click to process."],
  [/^审批待办 \((\d+)\)$/u, "Approval Inbox ($1)"],
  [/^层级 (\d+) \/ (.*)$/u, "Level $1 / $2"],
  [/^最近登录 (.*)$/u, "Last login $1"],
  [/^资产 (\d+) \/ 账号 (\d+) \/ 占比 (.*)$/u, "Assets $1 / Accounts $2 / Weight $3"],
  [/^(\d+) 个资产$/u, "$1 assets"],
  [/^本月当前使用 (.*)$/u, "Current month usage $1"],
  [/^(.*) \/ (\d+)天$/u, "$1 / $2 days"],
  [/^次数 (\d+)$/u, "Count $1"],
  [/^首次 (.*)$/u, "First $1"],
  [/^处理 (.*)$/u, "Resolved $1"],
  [/^授权至 (.*)$/u, "Authorized until $1"],
  [/^第 (\d+) 步$/u, "Step $1"],
  [/^(.*) \/ 第 (\d+) 步$/u, "$1 / Step $2"],
  [/^(\d+) 分钟$/u, "$1 minutes"],
  [/^附件 (\d+)$/u, "Attachments $1"],
  [/^标签 (.*) 没有值。$/u, "Tag $1 has no value."],
  [/^标签 (.*) 包含不建议的字符。$/u, "Tag $1 contains discouraged characters."],
  [/^将批量更新当前筛选出的 (\d+) 条资产，继续吗？$/u, "Bulk-update $1 currently filtered assets. Continue?"],
  [/^费用同步完成：上月整月 (.*)，上月同进度 (.*)，本月 (.*)，预计 (.*)。$/u, "Cost sync complete: last month $1, last month to date $2, current month $3, forecast $4."],
  [/^费用同步完成：上月整月 (.*)，上月同进度 (.*)，本月 (.*)，预计 (.*)，警告 (\d+) 条。$/u, "Cost sync complete: last month $1, last month to date $2, current month $3, forecast $4, warnings $5."],
  [/^发现 (\d+) 条，新增 (\d+) 条，更新 (\d+) 条$/u, "Discovered $1, created $2, updated $3"],
  [/^发现 (\d+) 条，新增 (\d+) 条，更新 (\d+) 条，警告 (\d+) 条。$/u, "Discovered $1, created $2, updated $3, warnings $4."],
  [/^发现 (\d+) 条，新增 (\d+) 条，更新 (\d+) 条，标记 stale (\d+) 条$/u, "Discovered $1, created $2, updated $3, marked stale $4"],
  [/^发现 (\d+) 条，新增 (\d+) 条，更新 (\d+) 条，标记 stale (\d+) 条，警告 (\d+) 条。$/u, "Discovered $1, created $2, updated $3, marked stale $4, warnings $5."],
  [/^导入完成：新增 (\d+) 条，跳过 (\d+) 条。$/u, "Import complete: created $1, skipped $2."],
  [/^批量更新完成：更新 (\d+) 条，跳过 (\d+) 条。$/u, "Bulk update complete: updated $1, skipped $2."],
  [/^已导出 (\d+) 条资产。$/u, "Exported $1 assets."],
  [/^拨测完成：(.*) \/ (.*) ms。$/u, "Probe complete: $1 / $2 ms."],
  [/^凭证明文：(.*)$/u, "Credential plaintext: $1"],
  [/^已回填云账号 (.*) 的配置表单。$/u, "Filled the config form for account $1."],
  [/^(.*) 的配置表单。$/u, "$1 config form."],
  [/^已切换到资产 (.*) 的编辑表单。$/u, "Switched to the edit form for asset $1."],
  [/^已为资产 (.*) 预填变更表单。$/u, "Prepared a change form for asset $1."],
  [/^关联资产：(.*)$/u, "Related asset: $1"],
  [/^最新日快照 (.*)$/u, "Latest daily snapshot $1"]
];

function currentLanguage() {
  return localStorage.getItem("opsledger_lang") === "en" ? "en" : "zh";
}

function t(key) {
  const lang = currentLanguage();
  return (i18nMessages[lang] && i18nMessages[lang][key]) || i18nMessages.zh[key] || key;
}

function applyI18n() {
  document.documentElement.lang = currentLanguage() === "en" ? "en" : "zh-CN";
  document.querySelectorAll("[data-i18n]").forEach((node) => {
    node.textContent = t(node.dataset.i18n);
  });
  document.querySelectorAll("[data-i18n-placeholder]").forEach((node) => {
    node.setAttribute("placeholder", t(node.dataset.i18nPlaceholder));
  });
  document.querySelectorAll("[data-i18n-title]").forEach((node) => {
    node.setAttribute("title", t(node.dataset.i18nTitle));
  });
  if (refs.languageToggleButton) {
    refs.languageToggleButton.textContent = currentLanguage() === "en" ? "ZH" : "EN";
    refs.languageToggleButton.title = currentLanguage() === "en" ? "Switch to Chinese" : "Switch to English";
  }
  updateSyncControlLabels();
  translateRenderedText();
  translateRenderedAttributes();
  translateRenderedFormValues();
}

function toggleLanguage() {
  localStorage.setItem("opsledger_lang", currentLanguage() === "en" ? "zh" : "en");
  applyI18n();
  render();
}

function translateUIMessage(value) {
  return currentLanguage() === "en" ? translateUIText(value) : value;
}

function translateUIText(value) {
  const text = String(value ?? "");
  if (!text) {
    return text;
  }
  const exact = i18nTextTranslations[text];
  if (exact) {
    return exact;
  }
  for (const [pattern, replacement] of i18nPatternTranslations) {
    if (pattern.test(text)) {
      return text.replace(pattern, replacement);
    }
  }
  return text;
}

function restoreTranslatedUIText(value) {
  const text = String(value ?? "");
  return i18nReverseTextTranslations[text] || text;
}

function restoreI18nFormValues(root = document) {
  root.querySelectorAll("[data-i18n-original-value]").forEach((node) => {
    node.value = node.dataset.i18nOriginalValue;
    delete node.dataset.i18nOriginalValue;
  });
}

function translateRenderedText() {
  const root = document.body;
  if (!root) {
    return;
  }
  const walker = document.createTreeWalker(root, NodeFilter.SHOW_TEXT, {
    acceptNode(node) {
      if (!node.nodeValue || !node.nodeValue.trim()) {
        return NodeFilter.FILTER_REJECT;
      }
      return shouldTranslateTextNode(node) ? NodeFilter.FILTER_ACCEPT : NodeFilter.FILTER_REJECT;
    }
  });
  const nodes = [];
  while (walker.nextNode()) {
    nodes.push(walker.currentNode);
  }
  nodes.forEach((node) => {
    if (currentLanguage() === "zh") {
      if (node.__opsledgerZhText) {
        node.nodeValue = node.__opsledgerZhText;
      }
      return;
    }
    const source = node.__opsledgerZhText || node.nodeValue;
    const translated = translateTextPreservingWhitespace(source);
    if (translated !== source) {
      node.__opsledgerZhText = source;
      node.nodeValue = translated;
    }
  });
}

function translateRenderedAttributes() {
  const attrs = ["placeholder", "title", "aria-label"];
  document.querySelectorAll("input, textarea, button, [title], [aria-label]").forEach((node) => {
    attrs.forEach((attr) => {
      if (!node.hasAttribute(attr)) {
        return;
      }
      const originalAttr = `data-i18n-original-${attr}`;
      if (currentLanguage() === "zh") {
        if (node.hasAttribute(originalAttr)) {
          node.setAttribute(attr, node.getAttribute(originalAttr));
        }
        return;
      }
      const source = node.getAttribute(originalAttr) || node.getAttribute(attr) || "";
      const translated = translateUIText(source);
      if (translated !== source) {
        node.setAttribute(originalAttr, source);
        node.setAttribute(attr, translated);
      }
    });
  });
}

function translateRenderedFormValues() {
  const selector = "input:not([type='hidden']):not([type='password']), textarea";
  document.querySelectorAll(selector).forEach((node) => {
    if (!shouldTranslateFormValue(node)) {
      return;
    }
    if (currentLanguage() === "zh") {
      if (node.dataset.i18nOriginalValue !== undefined) {
        node.value = node.dataset.i18nOriginalValue;
        delete node.dataset.i18nOriginalValue;
      }
      return;
    }
    const source = node.dataset.i18nOriginalValue || node.value || "";
    const translated = translateUIText(source);
    if (translated !== source) {
      node.dataset.i18nOriginalValue = source;
      node.value = translated;
    }
  });
}

function translateTextPreservingWhitespace(value) {
  const match = String(value).match(/^(\s*)(.*?)(\s*)$/su);
  if (!match) {
    return translateUIText(value);
  }
  const translated = translateUIText(match[2]);
  return `${match[1]}${translated}${match[3]}`;
}

function shouldTranslateTextNode(node) {
  const parent = node.parentElement;
  if (!parent) {
    return false;
  }
  if (parent.closest("script, style, textarea, input, pre, code, .mono, [data-no-i18n]")) {
    return false;
  }
  if (parent.closest(".tree-item-title, .developer-asset-name")) {
    return false;
  }
  return true;
}

function shouldTranslateFormValue(node) {
  if (!node || !node.value || node.closest("[data-no-i18n]")) {
    return false;
  }
  const translatableIds = new Set([
    "roleName",
    "roleDescription",
    "approvalFlowName",
    "approvalFlowDescription"
  ]);
  if (translatableIds.has(node.id)) {
    return true;
  }
  if (node.dataset.stepField === "approver_label") {
    return true;
  }
  return false;
}
