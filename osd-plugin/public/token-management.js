// oauth4os OSD Plugin — Token Management UI
// React components for list/create/revoke tokens
// Designed for OpenSearch Dashboards plugin framework

const { useState, useEffect, useCallback } = React;

// --- API layer ---
const API_BASE = window.OAUTH4OS_API || '/oauth';

async function api(path, opts = {}) {
  const res = await fetch(`${API_BASE}${path}`, {
    headers: { 'Content-Type': 'application/json', ...opts.headers },
    ...opts,
  });
  if (!res.ok) {
    const err = await res.json().catch(() => ({ error: res.statusText }));
    throw new Error(err.error_description || err.error || res.statusText);
  }
  if (res.status === 204) return null;
  return res.json();
}

// --- Components ---

function TokenBadge({ token }) {
  const isExpired = new Date(token.expires_at) < new Date();
  const status = token.revoked ? 'revoked' : isExpired ? 'expired' : 'active';
  const colors = { active: '#3fb950', expired: '#f0883e', revoked: '#f85149' };
  return React.createElement('span', {
    className: 'token-badge',
    style: { background: colors[status] + '22', color: colors[status], padding: '2px 8px', borderRadius: '10px', fontSize: '11px', fontWeight: 600 }
  }, status);
}

function ScopeTags({ scopes }) {
  return React.createElement('div', { className: 'scope-tags', style: { display: 'flex', gap: '4px', flexWrap: 'wrap' } },
    (scopes || []).map(s => React.createElement('code', {
      key: s,
      style: { fontSize: '11px', padding: '1px 6px', borderRadius: '4px', background: 'var(--surface, #161b22)', border: '1px solid var(--border, #30363d)' }
    }, s))
  );
}

function TimeAgo({ date }) {
  const d = new Date(date);
  const diff = Date.now() - d.getTime();
  const mins = Math.floor(diff / 60000);
  const hrs = Math.floor(mins / 60);
  const days = Math.floor(hrs / 24);
  const text = days > 0 ? `${days}d ago` : hrs > 0 ? `${hrs}h ago` : mins > 0 ? `${mins}m ago` : 'just now';
  return React.createElement('span', { title: d.toISOString(), style: { color: 'var(--muted, #8b949e)', fontSize: '12px' } }, text);
}

function TokenRow({ token, onRevoke }) {
  const [confirming, setConfirming] = useState(false);
  const isActive = !token.revoked && new Date(token.expires_at) > new Date();

  return React.createElement('tr', { className: 'token-row' },
    React.createElement('td', null,
      React.createElement('code', { style: { fontSize: '12px' } }, token.id.slice(0, 20) + '…')
    ),
    React.createElement('td', null, token.client_id),
    React.createElement('td', null, React.createElement(ScopeTags, { scopes: token.scopes })),
    React.createElement('td', null, React.createElement(TokenBadge, { token })),
    React.createElement('td', null, React.createElement(TimeAgo, { date: token.created_at })),
    React.createElement('td', null,
      isActive && !confirming && React.createElement('button', {
        className: 'btn btn-sm btn-danger',
        onClick: () => setConfirming(true),
        'aria-label': `Revoke token ${token.id}`
      }, 'Revoke'),
      confirming && React.createElement('span', { style: { display: 'flex', gap: '4px', alignItems: 'center' } },
        React.createElement('span', { style: { fontSize: '11px', color: 'var(--muted)' } }, 'Sure?'),
        React.createElement('button', {
          className: 'btn btn-sm btn-danger',
          onClick: () => { onRevoke(token.id); setConfirming(false); },
          'aria-label': 'Confirm revoke'
        }, 'Yes'),
        React.createElement('button', {
          className: 'btn btn-sm',
          onClick: () => setConfirming(false),
          'aria-label': 'Cancel revoke'
        }, 'No')
      )
    )
  );
}

function TokenList({ tokens, onRevoke, loading }) {
  if (loading) return React.createElement('div', { className: 'loading' }, 'Loading tokens…');
  if (!tokens.length) return React.createElement('div', { className: 'empty' }, 'No active tokens. Create one to get started.');

  return React.createElement('table', { className: 'token-table' },
    React.createElement('thead', null,
      React.createElement('tr', null,
        React.createElement('th', null, 'Token ID'),
        React.createElement('th', null, 'Client'),
        React.createElement('th', null, 'Scopes'),
        React.createElement('th', null, 'Status'),
        React.createElement('th', null, 'Created'),
        React.createElement('th', null, 'Actions')
      )
    ),
    React.createElement('tbody', null,
      tokens.map(t => React.createElement(TokenRow, { key: t.id, token: t, onRevoke }))
    )
  );
}

function CreateTokenForm({ onCreated }) {
  const [clientId, setClientId] = useState('');
  const [clientSecret, setClientSecret] = useState('');
  const [scope, setScope] = useState('');
  const [error, setError] = useState(null);
  const [result, setResult] = useState(null);
  const [submitting, setSubmitting] = useState(false);

  const handleSubmit = async (e) => {
    e.preventDefault();
    setError(null);
    setResult(null);
    setSubmitting(true);
    try {
      const body = new URLSearchParams({
        grant_type: 'client_credentials',
        client_id: clientId,
        client_secret: clientSecret,
        scope: scope,
      });
      const res = await fetch(`${API_BASE}/token`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
        body,
      });
      const data = await res.json();
      if (!res.ok) throw new Error(data.error_description || data.error);
      setResult(data);
      onCreated();
    } catch (err) {
      setError(err.message);
    } finally {
      setSubmitting(false);
    }
  };

  return React.createElement('form', { className: 'create-form', onSubmit: handleSubmit },
    React.createElement('div', { className: 'form-grid' },
      React.createElement('div', null,
        React.createElement('label', { className: 'form-label' }, 'Client ID'),
        React.createElement('input', {
          type: 'text', value: clientId, onChange: e => setClientId(e.target.value),
          placeholder: 'my-agent', required: true, className: 'form-input', 'aria-label': 'Client ID'
        })
      ),
      React.createElement('div', null,
        React.createElement('label', { className: 'form-label' }, 'Client Secret'),
        React.createElement('input', {
          type: 'password', value: clientSecret, onChange: e => setClientSecret(e.target.value),
          placeholder: '••••••••', required: true, className: 'form-input', 'aria-label': 'Client secret'
        })
      ),
      React.createElement('div', null,
        React.createElement('label', { className: 'form-label' }, 'Scopes'),
        React.createElement('input', {
          type: 'text', value: scope, onChange: e => setScope(e.target.value),
          placeholder: 'read:logs-* write:dashboards', className: 'form-input', 'aria-label': 'Token scopes'
        })
      ),
      React.createElement('div', { style: { display: 'flex', alignItems: 'flex-end' } },
        React.createElement('button', {
          type: 'submit', className: 'btn btn-primary', disabled: submitting,
          'aria-label': 'Create token'
        }, submitting ? 'Creating…' : '🔑 Create Token')
      )
    ),
    error && React.createElement('div', { className: 'alert alert-error', role: 'alert' }, '❌ ', error),
    result && React.createElement('div', { className: 'alert alert-success' },
      React.createElement('p', { style: { fontWeight: 600, marginBottom: '8px' } }, '✅ Token created — copy it now, it won\'t be shown again.'),
      React.createElement('div', { className: 'token-result' },
        React.createElement('label', { style: { fontSize: '11px', color: 'var(--muted)' } }, 'Access Token'),
        React.createElement('div', { className: 'copy-row' },
          React.createElement('code', { className: 'token-value' }, result.access_token),
          React.createElement('button', {
            type: 'button', className: 'btn btn-sm',
            'aria-label': 'Copy access token',
            onClick: () => navigator.clipboard.writeText(result.access_token)
          }, '📋')
        ),
        result.refresh_token && React.createElement(React.Fragment, null,
          React.createElement('label', { style: { fontSize: '11px', color: 'var(--muted)', marginTop: '8px', display: 'block' } }, 'Refresh Token'),
          React.createElement('div', { className: 'copy-row' },
            React.createElement('code', { className: 'token-value' }, result.refresh_token),
            React.createElement('button', {
              type: 'button', className: 'btn btn-sm',
              'aria-label': 'Copy refresh token',
              onClick: () => navigator.clipboard.writeText(result.refresh_token)
            }, '📋')
          )
        ),
        React.createElement('p', { style: { fontSize: '11px', color: 'var(--muted)', marginTop: '8px' } },
          `Expires in ${result.expires_in}s · Scope: ${result.scope}`)
      )
    )
  );
}

// --- Main App ---

function TokenManagement() {
  const [tokens, setTokens] = useState([]);
  const [loading, setLoading] = useState(true);
  const [tab, setTab] = useState('list');
  const [error, setError] = useState(null);

  const loadTokens = useCallback(async () => {
    try {
      setLoading(true);
      const data = await api('/tokens');
      setTokens(data || []);
      setError(null);
    } catch (err) {
      setError(err.message);
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => { loadTokens(); }, [loadTokens]);

  const handleRevoke = async (id) => {
    try {
      await api(`/token/${id}`, { method: 'DELETE' });
      loadTokens();
    } catch (err) {
      setError(err.message);
    }
  };

  return React.createElement('div', { className: 'token-mgmt' },
    React.createElement('div', { className: 'tm-header' },
      React.createElement('h2', null, '🔐 Token Management'),
      React.createElement('div', { className: 'tm-tabs' },
        React.createElement('button', {
          className: `tm-tab ${tab === 'list' ? 'active' : ''}`,
          onClick: () => setTab('list'), 'aria-label': 'List tokens'
        }, `📋 Tokens (${tokens.length})`),
        React.createElement('button', {
          className: `tm-tab ${tab === 'create' ? 'active' : ''}`,
          onClick: () => setTab('create'), 'aria-label': 'Create token'
        }, '🔑 Create'),
      ),
      React.createElement('button', {
        className: 'btn btn-sm', onClick: loadTokens, 'aria-label': 'Refresh token list'
      }, '🔄')
    ),
    error && React.createElement('div', { className: 'alert alert-error', role: 'alert' }, error),
    tab === 'list' && React.createElement(TokenList, { tokens, onRevoke: handleRevoke, loading }),
    tab === 'create' && React.createElement(CreateTokenForm, { onCreated: () => { loadTokens(); setTab('list'); } })
  );
}

// Export for OSD plugin or standalone use
window.OAuth4osTokenManagement = TokenManagement;
