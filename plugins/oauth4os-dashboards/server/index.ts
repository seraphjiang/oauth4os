import { PluginInitializerContext, CoreSetup, CoreStart, Plugin, Logger } from '../../../src/core/server';
import { IRouter } from '../../../src/core/server';
import { API_PREFIX } from '../common';

export class Oauth4osPlugin implements Plugin {
  private readonly logger: Logger;

  constructor(initializerContext: PluginInitializerContext) {
    this.logger = initializerContext.logger.get();
  }

  public setup(core: CoreSetup) {
    this.logger.debug('oauth4os: setup');
    const router = core.http.createRouter();
    registerRoutes(router, this.logger);
    return {};
  }

  public start(core: CoreStart) {
    this.logger.debug('oauth4os: started');
    return {};
  }

  public stop() {}
}

function registerRoutes(router: IRouter, logger: Logger) {
  const proxyBase = process.env.OAUTH4OS_PROXY_URL || 'http://localhost:8443';

  // List tokens
  router.get(
    { path: `${API_PREFIX}/tokens`, validate: false },
    async (context, request, response) => {
      try {
        const resp = await fetch(`${proxyBase}/oauth/tokens`);
        const body = await resp.json();
        return response.ok({ body });
      } catch (err) {
        logger.error(`Failed to list tokens: ${err}`);
        return response.customError({ statusCode: 502, body: { message: 'Proxy unreachable' } });
      }
    }
  );

  // Create token
  router.post(
    { path: `${API_PREFIX}/tokens`, validate: false },
    async (context, request, response) => {
      try {
        const resp = await fetch(`${proxyBase}/oauth/token`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/x-www-form-urlencoded' },
          body: new URLSearchParams(request.body as Record<string, string>).toString(),
        });
        const body = await resp.json();
        return response.ok({ body });
      } catch (err) {
        logger.error(`Failed to create token: ${err}`);
        return response.customError({ statusCode: 502, body: { message: 'Proxy unreachable' } });
      }
    }
  );

  // Revoke token
  router.delete(
    { path: `${API_PREFIX}/tokens/{id}`, validate: false },
    async (context, request, response) => {
      try {
        const id = (request.params as { id: string }).id;
        const resp = await fetch(`${proxyBase}/oauth/token/${id}`, { method: 'DELETE' });
        const body = await resp.json();
        return response.ok({ body });
      } catch (err) {
        logger.error(`Failed to revoke token: ${err}`);
        return response.customError({ statusCode: 502, body: { message: 'Proxy unreachable' } });
      }
    }
  );

  // Get single token
  router.get(
    { path: `${API_PREFIX}/tokens/{id}`, validate: false },
    async (context, request, response) => {
      try {
        const id = (request.params as { id: string }).id;
        const resp = await fetch(`${proxyBase}/oauth/token/${id}`);
        const body = await resp.json();
        return response.ok({ body });
      } catch (err) {
        logger.error(`Failed to get token: ${err}`);
        return response.customError({ statusCode: 502, body: { message: 'Proxy unreachable' } });
      }
    }
  );
}

export const plugin = (initializerContext: PluginInitializerContext) =>
  new Oauth4osPlugin(initializerContext);
