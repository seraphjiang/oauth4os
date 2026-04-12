import React, { useState, useEffect } from 'react';
import ReactDOM from 'react-dom';
import {
  EuiPage, EuiPageBody, EuiPageHeader, EuiBasicTable,
  EuiButton, EuiModal, EuiModalHeader, EuiModalHeaderTitle,
  EuiModalBody, EuiModalFooter, EuiForm, EuiFormRow,
  EuiFieldText, EuiCallOut, EuiBadge, EuiSpacer,
} from '@elastic/eui';
import { AppMountParameters } from '../../../../src/core/public';
import { OAuthToken, API_PREFIX } from '../../common';

const TokenManagement: React.FC = () => {
  const [tokens, setTokens] = useState<OAuthToken[]>([]);
  const [loading, setLoading] = useState(true);
  const [showCreate, setShowCreate] = useState(false);
  const [clientId, setClientId] = useState('');
  const [clientSecret, setClientSecret] = useState('');
  const [scope, setScope] = useState('read:logs-*');
  const [error, setError] = useState('');

  const fetchTokens = async () => {
    setLoading(true);
    try {
      const resp = await fetch(`${API_PREFIX}/tokens`);
      const data = await resp.json();
      setTokens(data.tokens || data || []);
    } catch (e) {
      setError('Failed to load tokens');
    }
    setLoading(false);
  };

  useEffect(() => { fetchTokens(); }, []);

  const createToken = async () => {
    try {
      await fetch(`${API_PREFIX}/tokens`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          grant_type: 'client_credentials',
          client_id: clientId,
          client_secret: clientSecret,
          scope,
        }),
      });
      setShowCreate(false);
      setClientId('');
      setClientSecret('');
      fetchTokens();
    } catch (e) {
      setError('Failed to create token');
    }
  };

  const revokeToken = async (id: string) => {
    try {
      await fetch(`${API_PREFIX}/tokens/${id}`, { method: 'DELETE' });
      fetchTokens();
    } catch (e) {
      setError('Failed to revoke token');
    }
  };

  const columns = [
    { field: 'id', name: 'ID', truncateText: true, width: '180px' },
    { field: 'client_id', name: 'Client', truncateText: true },
    { field: 'scope', name: 'Scope' },
    { field: 'created_at', name: 'Created' },
    { field: 'expires_at', name: 'Expires' },
    {
      field: 'active', name: 'Status',
      render: (active: boolean) => (
        <EuiBadge color={active ? 'success' : 'danger'}>
          {active ? 'Active' : 'Revoked'}
        </EuiBadge>
      ),
    },
    {
      name: 'Actions',
      render: (token: OAuthToken) =>
        token.active ? (
          <EuiButton size="s" color="danger" onClick={() => revokeToken(token.id)}>
            Revoke
          </EuiButton>
        ) : null,
    },
  ];

  return (
    <EuiPage>
      <EuiPageBody>
        <EuiPageHeader
          pageTitle="OAuth Token Management"
          rightSideItems={[
            <EuiButton fill onClick={() => setShowCreate(true)}>Create Token</EuiButton>,
          ]}
        />
        {error && (
          <>
            <EuiCallOut title={error} color="danger" iconType="alert" />
            <EuiSpacer />
          </>
        )}
        <EuiBasicTable items={tokens} columns={columns} loading={loading} />

        {showCreate && (
          <EuiModal onClose={() => setShowCreate(false)}>
            <EuiModalHeader>
              <EuiModalHeaderTitle>Create OAuth Token</EuiModalHeaderTitle>
            </EuiModalHeader>
            <EuiModalBody>
              <EuiForm>
                <EuiFormRow label="Client ID">
                  <EuiFieldText value={clientId} onChange={(e) => setClientId(e.target.value)} />
                </EuiFormRow>
                <EuiFormRow label="Client Secret">
                  <EuiFieldText type="password" value={clientSecret} onChange={(e) => setClientSecret(e.target.value)} />
                </EuiFormRow>
                <EuiFormRow label="Scope">
                  <EuiFieldText value={scope} onChange={(e) => setScope(e.target.value)} />
                </EuiFormRow>
              </EuiForm>
            </EuiModalBody>
            <EuiModalFooter>
              <EuiButton onClick={() => setShowCreate(false)}>Cancel</EuiButton>
              <EuiButton fill onClick={createToken}>Create</EuiButton>
            </EuiModalFooter>
          </EuiModal>
        )}
      </EuiPageBody>
    </EuiPage>
  );
};

export function renderApp({ element }: AppMountParameters) {
  ReactDOM.render(<TokenManagement />, element);
  return () => ReactDOM.unmountComponentAtNode(element);
}
