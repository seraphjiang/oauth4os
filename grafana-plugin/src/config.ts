import React, { ChangeEvent } from 'react';
import { DataSourcePluginOptionsEditorProps } from '@grafana/data';
import { InlineField, Input, SecretInput } from '@grafana/ui';

interface JsonData { proxyUrl: string; }
interface SecureJsonData { accessToken: string; }

type Props = DataSourcePluginOptionsEditorProps<JsonData, SecureJsonData>;

export function ConfigEditor({ options, onOptionsChange }: Props) {
  const onProxyUrlChange = (e: ChangeEvent<HTMLInputElement>) => {
    onOptionsChange({ ...options, jsonData: { ...options.jsonData, proxyUrl: e.target.value } });
  };

  const onTokenChange = (e: ChangeEvent<HTMLInputElement>) => {
    onOptionsChange({ ...options, secureJsonData: { ...options.secureJsonData, accessToken: e.target.value } });
  };

  const onTokenReset = () => {
    onOptionsChange({ ...options, secureJsonData: { ...options.secureJsonData, accessToken: '' } });
  };

  return (
    <>
      <InlineField label="oauth4os Proxy URL" labelWidth={20} tooltip="e.g., http://localhost:8443">
        <Input width={40} value={options.jsonData.proxyUrl || ''} onChange={onProxyUrlChange} placeholder="http://localhost:8443" />
      </InlineField>
      <InlineField label="Access Token" labelWidth={20} tooltip="OAuth token with read scope">
        <SecretInput
          width={40}
          isConfigured={!!options.secureJsonFields?.accessToken}
          value={options.secureJsonData?.accessToken || ''}
          onChange={onTokenChange}
          onReset={onTokenReset}
          placeholder="tok_..."
        />
      </InlineField>
    </>
  );
}
