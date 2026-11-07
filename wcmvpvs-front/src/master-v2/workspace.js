import { inject, onScopeDispose } from 'vue';

export const masterWorkspaceKey = Symbol('master-workspace');

export function useMasterWorkspace() {
  const workspace = inject(masterWorkspaceKey);
  return {
    ...workspace,
    useDirty(source) {
      onScopeDispose(workspace.registerDirty(source));
    },
  };
}
