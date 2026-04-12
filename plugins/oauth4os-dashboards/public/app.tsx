import React, { useState, useEffect, useCallback } from 'react';
import ReactDOM from 'react-dom';
import {
  EuiPage, EuiPageBody, EuiPageHeader, EuiBasicTable,
  EuiButton, EuiModal, EuiModalHeader, EuiModalHeaderTitle,
  EuiModalBody, EuiModalFooter, EuiForm, EuiFormRow,
  EuiFieldText, EuiCallOut, EuiBadge, EuiSpacer,
  EuiConfirmModal, EuiCopy, EuiCode, EuiFlexGroup, EuiFlexItem,
  EuiText, EuiToolTip, EuiButtonIcon,
} from '@elastic/eui';
import { AppMountParameters } from '../../../../src/core/public';
import { OAuthToken, API_PREFIX } from '../../common';

function statusBadge(token: OAuthToken) {
  if (token.revoked) return <EuiBadge color="danger">Revoked</EuiBadge>;
  if (new Date(token.expires_at) < new Date()) return <EuiBadge color="warning">Expired</EuiBadge>;
  return <EuiBadge color="success">Active</EuiBadge>;
}

function timeAgo(date: string) {
  const diff = Date.now() - new Date(date).getTime();
  const mins = Math.floor(diff / 60000);
  const hrs = Math.floor(mins / 60);
  const days = Math.floor(hrs / 24);
  if (days > 0) return `${days}d ago`;
  if (hrs > 0) return `${hrs}h ago`;
  if (mins > 0) return `${mins}m ago`;
  return 'just now';
}

const TokenManagement: React.FC = () => {
  const [tokens, setTokens] = useState<OAuthToken[]>([]);
  const [loading, setLoading] = useState(true);
  const [showCreate, setShowCreate] = useState(false);
  const [revokeTarget, setRevokeTarget] = useState<string | null>(null);
  const [clientId, setClientId] = useState('');
  const [clientSecret, setClientSecret] = useState('');
  const [scope, setScope] = useState('read:logs-*');
  const [error, setError] = useState('');
  const [created, setCreated] = useState<{ access_token: string; refresh_token?: string; expires_in: number; scope: string } | null>(null);

  const fetchTokens = useCallback(async () => {
    setLoading(true);
    try {
      const resp = await fetch(`${API_PREFIX}/tokens`);
      const data = await resp.json();
      setTokens(data.tokens || data || []);
      setError('');
    } catch {
      setError('Failed to load tokens');
    }
    setLoading(false);
  }, []);

  useEffect(() => { fetchTokens(); }, [fetchTokens]);

  const createToken = async () => {
    try {
      const resp = await fetch(`${API_PREFIX}/tokens`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ grant_type: 'client_credentials', client_id: clientId, client_secret: clientSecret, scope }),
      });
      const data = await resp.json();
      if (!resp.ok) throw new Error(data.error_description || data.error);
      setCreated(data);
      setShowCreate(false);
      setClientId('');
      setClientSecret('');
      fetchTokens();
    } catch (e: any) {
      setError(e.message || 'Failed to create token');
    }
  };

  const revokeToken = async (id: string) => {
    try {
      await fetch(`${API_PREFIX}/tokens/${id}`, { method: 'DELETE' });
      setRevokeTarget(null);
      fetchTokens();
    } catch {
      setError('Failed to revoke token');
    }
  };

  const columns = [
    { field: 'id', name: 'Token ID', truncateText: true, width: '200px',
      render: (id: string) => (
        <EuiCopy textToCopy={id}>{(copy) => (
          <EuiToolTip content="Click to copy full ID">
            <EuiCode onClick={copy} style={{ cursor: 'pointer' }}>{id.slice(0, 20)}…</EuiCode>
          </EuiToolTip>
        )}</EuiCopy>
      ),
    },
    { field: 'client_id', name: 'Client' },
    { field: 'scopes', name: 'Scopes',
      render: (scopes: string[]) => (
        <EuiFlexGroup gutterSize="xs" wrap>
          {(scopes || []).map(s => <EuiFlexItem grow={false} key={s}><EuiBadge>{s}</EuiBadge></EuiFlexItem>)}
        </EuiFlexGroup>
      ),
    },
    { name: 'Status', render: (token: OAuthToken) => statusBadge(token) },
    { field: 'created_at', name: 'Created',
      render: (d: string) => <EuiToolTip content={new Date(d).toISOString()}><span>{timeAgo(d)}</span></EuiToolTip>,
    },
    { name: 'Actions',
      render: (token: OAuthToken) => {
        const isActive = !token.revoked && new Date(token.expires_at) > new Date();
        return isActive ? (
          <EuiButtonIcon iconType="trash" color="danger" aria-label={`Revoke token ${token.id}`}
            onClick={() => setRevokeTarget(token.id)} />
        ) : null;
      },
    },
  ];

  return (
    <EuiPage>
      <EuiPageBody>
        <EuiPageHeader
          pageTitle="🔐 OAuth Token Management"
          rightSideItems={[
            <EuiButton iconType="refresh" onClick={fetchTokens} aria-label="Refresh tokens">Refresh</EuiButton>,
            <EuiButton fill iconType="plusInCircle" onClick={() => setShowCreate(true)} aria-label="Create token">Create Token</EuiButton>,
          ]}
        />
        {error && <><EuiCallOut title={error} color="danger" iconType="alert" /><EuiSpacer /></>}

        {created && (
          <>
            <EuiCallOut title="Token created — copy it now, it won't be shown again." color="success" iconType="check">
              <EuiSpacer size="s" />
              <EuiText size="xs"><strong>Access Token</strong></EuiText>
              <EuiFlexGroup gutterSize="s" alignItems="center">
                <EuiFlexItem><EuiCode>{created.access_token}</EuiCode></EuiFlexItem>
                <EuiFlexItem grow={false}>
                  <EuiCopy textToCopy={created.access_token}>{(copy) => (
                    <EuiButtonIcon iconType="copy" onClick={copy} aria-label="Copy access token" />
                  )}</EuiCopy>
                </EuiFlexItem>
              </EuiFlexGroup>
              {created.refresh_token && (
                <>
                  <EuiSpacer size="s" />
                  <EuiText size="xs"><strong>Refresh Token</strong></EuiText>
                  <EuiFlexGroup gutterSize="s" alignItems="center">
                    <EuiFlexItem><EuiCode>{created.refresh_token}</EuiCode></EuiFlexItem>
                    <EuiFlexItem grow={false}>
                      <EuiCopy textToCopy={created.refresh_token}>{(copy) => (
                        <EuiButtonIcon iconType="copy" onClick={copy} aria-label="Copy refresh token" />
                      )}</EuiCopy>
                    </EuiFlexItem>
                  </EuiFlexGroup>
                </>
              )}
              <EuiSpacer size="s" />
              <EuiText size="xs" color="subdued">Expires in {created.expires_in}s · Scope: {created.scope}</EuiText>
              <EuiSpacer size="s" />
              <EuiButton size="s" onClick={() => setCreated(null)}>Dismiss</EuiButton>
            </EuiCallOut>
            <EuiSpacer />
          </>
        )}

        <EuiBasicTable items={tokens} columns={columns} loading={loading} />

        {showCreate && (
          <EuiModal onClose={() => setShowCreate(false)}>
            <EuiModalHeader><EuiModalHeaderTitle>Create OAuth Token</EuiModalHeaderTitle></EuiModalHeader>
            <EuiModalBody>
              <EuiForm>
                <EuiFormRow label="Client ID">
                  <EuiFieldText value={clientId} onChange={(e) => setClientId(e.target.value)} aria-label="Client ID" />
                </EuiFormRow>
                <EuiFormRow label="Client Secret">
                  <EuiFieldText type="password" value={clientSecret} onChange={(e) => setClientSecret(e.target.value)} aria-label="Client secret" />
                </EuiFormRow>
                <EuiFormRow label="Scope" helpText="Space-separated scopes, e.g. read:logs-* write:dashboards">
                  <EuiFieldText value={scope} onChange={(e) => setScope(e.target.value)} aria-label="Token scopes" />
                </EuiFormRow>
              </EuiForm>
            </EuiModalBody>
            <EuiModalFooter>
              <EuiButton onClick={() => setShowCreate(false)}>Cancel</EuiButton>
              <EuiButton fill onClick={createToken} aria-label="Submit create token">Create</EuiButton>
            </EuiModalFooter>
          </EuiModal>
        )}

        {revokeTarget && (
          <EuiConfirmModal
            title="Revoke token?"
            onCancel={() => setRevokeTarget(null)}
            onConfirm={() => revokeToken(revokeTarget)}
            cancelButtonText="Cancel"
            confirmButtonText="Revoke"
            buttonColor="danger"
          >
            <p>This will immediately invalidate the token. Any clients using it will lose access.</p>
          </EuiConfirmModal>
        )}
      </EuiPageBody>
    </EuiPage>
  );
};

export function renderApp({ element }: AppMountParameters) {
  ReactDOM.render(<TokenManagement />, element);
  return () => ReactDOM.unmountComponentAtNode(element);
}
