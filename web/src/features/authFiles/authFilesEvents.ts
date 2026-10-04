export const AUTH_FILES_CHANGED_EVENT = 'auth-files-changed';

export const notifyAuthFilesChanged = () => {
  if (typeof window !== 'undefined') {
    window.dispatchEvent(new Event(AUTH_FILES_CHANGED_EVENT));
  }
};
