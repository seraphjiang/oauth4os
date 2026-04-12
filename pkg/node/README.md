# oauth4os Node.js SDK

Node.js client for [oauth4os](https://github.com/seraphjiang/oauth4os) — OAuth 2.0 proxy for OpenSearch.

## Install

```bash
npm install ./pkg/node
# Zero dependencies — uses built-in fetch (Node 18+)
```

## Usage

```javascript
const { Client } = require('oauth4os');

// Token is auto-managed
const c = new Client('http://localhost:8443', 'my-client', 'my-secret', {
  scopes: ['read:logs-*'],
});

// Search
const docs = await c.search('logs-*', { query: { match: { level: 'error' } } });

// Index
await c.index('logs-app', { level: 'info', msg: 'deployed' });

// Health
console.log(await c.health()); // { status: 'ok', version: '0.2.0' }

// Token lifecycle
const token = await c.createToken('read:logs-*');
await c.revokeToken(token);

// Dynamic client registration
const { clientID, clientSecret } = await c.register('my-agent', 'read:*');

// Raw request
const result = await c.do('GET', '/logs-*/_count');
```

## Requirements

Node.js 18+ (uses built-in `fetch`). Zero external dependencies.
