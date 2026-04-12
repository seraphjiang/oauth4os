import React, { ChangeEvent } from 'react';
import { QueryEditorProps } from '@grafana/data';
import { InlineField, Input, Select, CodeEditor } from '@grafana/ui';
import { DataSource, OAuth4osQuery, OAuth4osOptions } from './datasource';

type Props = QueryEditorProps<DataSource, OAuth4osQuery, OAuth4osOptions>;

const queryTypes = [
  { label: 'Search', value: 'search' as const },
  { label: 'Count', value: 'count' as const },
  { label: 'Cat Indices', value: 'cat' as const },
];

export function QueryEditor({ query, onChange, onRunQuery }: Props) {
  const onIndexChange = (e: ChangeEvent<HTMLInputElement>) => {
    onChange({ ...query, index: e.target.value });
  };

  const onQueryTypeChange = (v: any) => {
    onChange({ ...query, queryType: v.value });
    onRunQuery();
  };

  const onQueryBodyChange = (value: string) => {
    onChange({ ...query, queryBody: value });
  };

  return (
    <>
      <InlineField label="Query Type" labelWidth={12}>
        <Select options={queryTypes} value={query.queryType || 'search'} onChange={onQueryTypeChange} width={20} />
      </InlineField>
      {query.queryType !== 'cat' && (
        <>
          <InlineField label="Index" labelWidth={12} tooltip="e.g., logs-* or logs-2026.04">
            <Input width={30} value={query.index || ''} onChange={onIndexChange} placeholder="logs-*" onBlur={onRunQuery} />
          </InlineField>
          <InlineField label="Query Body" labelWidth={12}>
            <CodeEditor
              language="json"
              height={120}
              width="100%"
              value={query.queryBody || '{"query":{"match_all":{}}}'}
              onBlur={onQueryBodyChange}
              showMiniMap={false}
            />
          </InlineField>
        </>
      )}
    </>
  );
}
