// Token types shared between server and public
export interface OAuthToken {
  id: string;
  client_id: string;
  scope: string;
  created_at: string;
  expires_at: string;
  active: boolean;
}

export interface CreateTokenRequest {
  client_id: string;
  client_secret: string;
  scope: string;
  grant_type: 'client_credentials';
}

export interface TokenListResponse {
  tokens: OAuthToken[];
}

export const PLUGIN_ID = 'oauth4os';
export const PLUGIN_NAME = 'OAuth Token Management';
export const API_PREFIX = '/api/oauth4os';
