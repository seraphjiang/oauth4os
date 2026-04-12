const vscode = require('vscode');

let proxyUrl = '';
let currentToken = '';

function activate(context) {
  const config = () => vscode.workspace.getConfiguration('oauth4os');
  proxyUrl = config().get('proxyUrl') || 'http://localhost:8443';

  // Tree data providers
  const tokenProvider = new TokenTreeProvider();
  const scopeProvider = new ScopeTreeProvider();
  const logProvider = new LogTreeProvider();

  vscode.window.registerTreeDataProvider('oauth4os.tokens', tokenProvider);
  vscode.window.registerTreeDataProvider('oauth4os.scopes', scopeProvider);
  vscode.window.registerTreeDataProvider('oauth4os.logs', logProvider);

  // Commands
  context.subscriptions.push(
    vscode.commands.registerCommand('oauth4os.connect', async () => {
      const url = await vscode.window.showInputBox({
        prompt: 'oauth4os proxy URL',
        value: proxyUrl,
        placeHolder: 'http://localhost:8443',
      });
      if (url) {
        proxyUrl = url;
        await config().update('proxyUrl', url, true);
        try {
          const health = await apiGet('/health');
          vscode.window.showInformationMessage(`Connected to oauth4os v${health.version} (uptime: ${health.uptime_seconds}s)`);
          tokenProvider.refresh();
        } catch (e) {
          vscode.window.showErrorMessage(`Connection failed: ${e.message}`);
        }
      }
    }),

    vscode.commands.registerCommand('oauth4os.createToken', async () => {
      const clientId = await vscode.window.showInputBox({
        prompt: 'Client ID',
        value: config().get('clientId') || '',
      });
      if (!clientId) return;
      const secret = await vscode.window.showInputBox({ prompt: 'Client Secret', password: true });
      if (!secret) return;
      const scope = await vscode.window.showInputBox({
        prompt: 'Scope',
        value: config().get('defaultScope') || 'read:logs-*',
      });
      if (!scope) return;

      try {
        const body = new URLSearchParams({
          grant_type: 'client_credentials', client_id: clientId, client_secret: secret, scope,
        });
        const data = await apiPost('/oauth/token', body.toString(), 'application/x-www-form-urlencoded');
        currentToken = data.access_token;
        await vscode.env.clipboard.writeText(data.access_token);
        vscode.window.showInformationMessage(`Token created (expires in ${data.expires_in}s). Copied to clipboard.`);
        tokenProvider.refresh();
      } catch (e) {
        vscode.window.showErrorMessage(`Create failed: ${e.message}`);
      }
    }),

    vscode.commands.registerCommand('oauth4os.revokeToken', async (item) => {
      const id = item?.id || await vscode.window.showInputBox({ prompt: 'Token ID to revoke' });
      if (!id) return;
      const confirm = await vscode.window.showWarningMessage(`Revoke token ${id.slice(0, 20)}…?`, 'Revoke', 'Cancel');
      if (confirm !== 'Revoke') return;
      try {
        await apiFetch(`/oauth/token/${id}`, { method: 'DELETE' });
        vscode.window.showInformationMessage('Token revoked.');
        tokenProvider.refresh();
      } catch (e) {
        vscode.window.showErrorMessage(`Revoke failed: ${e.message}`);
      }
    }),

    vscode.commands.registerCommand('oauth4os.refreshTokens', () => tokenProvider.refresh()),

    vscode.commands.registerCommand('oauth4os.inspectToken', async (item) => {
      const id = item?.id || await vscode.window.showInputBox({ prompt: 'Token ID' });
      if (!id) return;
      try {
        const token = await apiGet(`/oauth/token/${id}`);
        const doc = await vscode.workspace.openTextDocument({
          content: JSON.stringify(token, null, 2), language: 'json',
        });
        vscode.window.showTextDocument(doc);
      } catch (e) {
        vscode.window.showErrorMessage(`Inspect failed: ${e.message}`);
      }
    }),

    vscode.commands.registerCommand('oauth4os.copyToken', async (item) => {
      if (item?.id) {
        await vscode.env.clipboard.writeText(item.id);
        vscode.window.showInformationMessage('Token ID copied.');
      }
    }),

    vscode.commands.registerCommand('oauth4os.queryOpenSearch', async () => {
      if (!currentToken) {
        vscode.window.showWarningMessage('No active token. Create one first.');
        return;
      }
      const query = await vscode.window.showInputBox({
        prompt: 'OpenSearch query (e.g., logs-*/_search)',
        placeHolder: 'logs-*/_search',
      });
      if (!query) return;
      try {
        const path = query.startsWith('/') ? query : `/${query}`;
        const data = await apiFetch(path, {
          method: 'GET',
          headers: { 'Authorization': `Bearer ${currentToken}` },
        });
        const doc = await vscode.workspace.openTextDocument({
          content: JSON.stringify(data, null, 2), language: 'json',
        });
        vscode.window.showTextDocument(doc);
      } catch (e) {
        vscode.window.showErrorMessage(`Query failed: ${e.message}`);
      }
    }),
  );

  // Auto-connect on activation
  apiGet('/health').then(h => {
    vscode.window.setStatusBarMessage(`oauth4os v${h.version}`, 5000);
    tokenProvider.refresh();
  }).catch(() => {});
}

// --- API helpers ---

async function apiFetch(path, opts = {}) {
  const url = `${proxyUrl}${path}`;
  const res = await fetch(url, { ...opts, headers: { 'Content-Type': 'application/json', ...opts.headers } });
  if (!res.ok) {
    const err = await res.json().catch(() => ({}));
    throw new Error(err.error_description || err.error || res.statusText);
  }
  if (res.status === 204) return null;
  return res.json();
}

async function apiGet(path) { return apiFetch(path); }

async function apiPost(path, body, contentType) {
  return apiFetch(path, { method: 'POST', body, headers: { 'Content-Type': contentType || 'application/json' } });
}

// --- Tree providers ---

class TokenTreeProvider {
  constructor() { this._onDidChange = new vscode.EventEmitter(); this.onDidChangeTreeData = this._onDidChange.event; }
  refresh() { this._onDidChange.fire(); }
  getTreeItem(el) { return el; }
  async getChildren() {
    try {
      const tokens = await apiGet('/oauth/tokens');
      if (!tokens || !tokens.length) return [new vscode.TreeItem('No active tokens')];
      return (tokens || []).map(t => {
        const expired = new Date(t.expires_at) < new Date();
        const status = t.revoked ? '🔴' : expired ? '🟡' : '🟢';
        const item = new vscode.TreeItem(`${status} ${t.id.slice(0, 16)}… — ${(t.scopes || []).join(', ')}`, vscode.TreeItemCollapsibleState.None);
        item.id = t.id;
        item.tooltip = `Client: ${t.client_id}\nScopes: ${(t.scopes || []).join(', ')}\nCreated: ${t.created_at}\nExpires: ${t.expires_at}`;
        item.contextValue = t.revoked || expired ? 'token-inactive' : 'token-active';
        item.command = { command: 'oauth4os.inspectToken', title: 'Inspect', arguments: [{ id: t.id }] };
        return item;
      });
    } catch {
      return [new vscode.TreeItem('⚠️ Connect to proxy first')];
    }
  }
}

class ScopeTreeProvider {
  getTreeItem(el) { return el; }
  async getChildren() {
    try {
      const health = await apiGet('/health');
      return [
        new vscode.TreeItem(`Proxy: ${proxyUrl}`),
        new vscode.TreeItem(`Version: ${health.version}`),
        new vscode.TreeItem(`Uptime: ${health.uptime_seconds}s`),
      ];
    } catch {
      return [new vscode.TreeItem('Not connected')];
    }
  }
}

class LogTreeProvider {
  getTreeItem(el) { return el; }
  async getChildren() {
    try {
      const metrics = await fetch(`${proxyUrl}/metrics`).then(r => r.text());
      const lines = metrics.split('\n').filter(l => l && !l.startsWith('#'));
      return lines.slice(0, 10).map(l => new vscode.TreeItem(l));
    } catch {
      return [new vscode.TreeItem('Not connected')];
    }
  }
}

function deactivate() {}

module.exports = { activate, deactivate };
