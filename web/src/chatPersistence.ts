const STORAGE_KEY = 'stefano.chatPersistence';
const CURRENT_VERSION = 1;

export type PersistedChatStateV1 = {
  version: 1;
  chatIds: string[];
  activeChatId: string | null;
};

export type PersistedChatState = PersistedChatStateV1;

function isRecord(v: unknown): v is Record<string, unknown> {
  return !!v && typeof v === 'object' && !Array.isArray(v);
}

export function loadPersisted(): PersistedChatState | null {
  try {
    const raw = localStorage.getItem(STORAGE_KEY);
    if (!raw) return null;
    const parsed: unknown = JSON.parse(raw);
    if (!isRecord(parsed)) return null;
    const version = parsed.version;
    if (version !== CURRENT_VERSION) {
      // Future: migrate older versions here
      return null;
    }
    const chatIds = parsed.chatIds;
    const activeChatId = parsed.activeChatId;
    if (!Array.isArray(chatIds) || chatIds.some((x) => typeof x !== 'string')) {
      return null;
    }
    if (activeChatId !== null && typeof activeChatId !== 'string') {
      return null;
    }
    return {
      version: 1,
      chatIds: chatIds as string[],
      activeChatId: activeChatId as string | null,
    };
  } catch {
    return null;
  }
}

export function savePersisted(state: Omit<PersistedChatStateV1, 'version'>): void {
  try {
    const payload: PersistedChatStateV1 = {
      version: CURRENT_VERSION,
      chatIds: state.chatIds,
      activeChatId: state.activeChatId,
    };
    localStorage.setItem(STORAGE_KEY, JSON.stringify(payload));
  } catch {
    /* ignore quota / private mode */
  }
}
