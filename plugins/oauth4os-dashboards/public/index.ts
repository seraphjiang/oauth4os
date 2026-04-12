import { AppMountParameters, CoreSetup, CoreStart, Plugin } from '../../../src/core/public';
import { PLUGIN_ID, PLUGIN_NAME } from '../common';

export class Oauth4osPlugin implements Plugin {
  public setup(core: CoreSetup) {
    core.application.register({
      id: PLUGIN_ID,
      title: PLUGIN_NAME,
      category: { id: 'management', label: 'Management', order: 5000 },
      async mount(params: AppMountParameters) {
        const { renderApp } = await import('./app');
        return renderApp(params);
      },
    });
    return {};
  }

  public start(core: CoreStart) {
    return {};
  }

  public stop() {}
}

export const plugin = () => new Oauth4osPlugin();
