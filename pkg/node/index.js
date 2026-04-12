/**
 * oauth4os Node.js SDK — OAuth 2.0 proxy client for OpenSearch.
 *
 * @example
 * const { Client } = require('oauth4os');
 * const c = new Client('http://localhost:8443', 'my-client', 'my-secret', { scopes: ['read:logs-*'] });
 * const docs = await c.search('logs-*', { query: { match: { level: 'error' } } });
 */

class Client {
  /**
   * @param {string} baseURL - Proxy URL
   * @param {string} clientID - OAuth client ID
   * @param {string} clientSecret - OAuth client secret
   * @param {object} [opts] - Options
   * @param {string[]} [opts.scopes] - Requested scopes
   */
  constructor(baseURL, clientID, clientSecret, opts = {}) {
    this.baseURL = baseURL.replace(/\/$/, '');
    this.clientID = clientID;
    this.clientSecret = clientSecret;
    this.scopes = (opts.scopes || ['admin']).join(' ');
    this._token = null;
    this._expiry = 0;
  }

  /** Get a valid access token, fetching or refreshing as needed. */
  async token() {
    if (this._token && Date.now() < this._expiry - 30000) {
      return this._token;
    }
    return this._fetchToken();
  }

  async _fetchToken() {
    const body = new URLSearchParams({
      grant_type: 'client_credentials',
      client_id: this.clientID,
      client_secret: this.clientSecret,
      scope: this.scopes,
    });
    const resp = await fetch(`${this.baseURL}/oauth/token`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
      body: body.toString(),
    });
    if (!resp.ok) throw new Error(`Token request failed: ${resp.status}`);
    const data = await resp.json();
    this._token = data.access_token;
    this._expiry = Date.now() + (data.expires_in || 3600) * 1000;
    return this._token;
  }

  /** Execute an authenticated request. */
  async do(method, path, body) {
    const tok = await this.token();
    const opts = {
      method,
      headers: { Authorization: `Bearer ${tok}`, 'Content-Type': 'application/json' },
    };
    if (body) opts.body = JSON.stringify(body);
    const resp = await fetch(`${this.baseURL}${path}`, opts);
    if (!resp.ok) {
      const text = await resp.text();
      throw new Error(`${method} ${path} returned ${resp.status}: ${text}`);
    }
    return resp.json();
  }

  /** Query an OpenSearch index. Returns array of _source docs. */
  async search(index, query) {
    const result = await this.do('POST', `/${index}/_search`, query);
    return (result.hits?.hits || []).map((h) => h._source);
  }

  /** Index a document. */
  async index(index, doc, id) {
    const path = id ? `/${index}/_doc/${id}` : `/${index}/_doc`;
    return this.do('POST', path, doc);
  }

  /** Check proxy health (unauthenticated). */
  async health() {
    const resp = await fetch(`${this.baseURL}/health`);
    return resp.json();
  }

  /** Issue a new scoped token. */
  async createToken(scope) {
    const body = new URLSearchParams({
      grant_type: 'client_credentials',
      client_id: this.clientID,
      scope,
    });
    const resp = await fetch(`${this.baseURL}/oauth/token`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
      body: body.toString(),
    });
    const data = await resp.json();
    return data.access_token;
  }

  /** Revoke a token by ID. */
  async revokeToken(tokenID) {
    const tok = await this.token();
    const resp = await fetch(`${this.baseURL}/oauth/token/${tokenID}`, {
      method: 'DELETE',
      headers: { Authorization: `Bearer ${tok}` },
    });
    if (!resp.ok && resp.status !== 204) throw new Error(`Revoke failed: ${resp.status}`);
  }

  /** Dynamic client registration (RFC 7591). */
  async register(clientName, scope = '') {
    const resp = await fetch(`${this.baseURL}/oauth/register`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ client_name: clientName, scope, grant_types: ['client_credentials'] }),
    });
    if (!resp.ok) throw new Error(`Register failed: ${resp.status}`);
    const data = await resp.json();
    return { clientID: data.client_id, clientSecret: data.client_secret };
  }
}

module.exports = { Client };
