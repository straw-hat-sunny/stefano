import type { ChatSession, Message } from './types';

function newClientMessageId(): string {
  if (typeof globalThis.crypto?.randomUUID === 'function') {
    return globalThis.crypto.randomUUID();
  }

  if (typeof globalThis.crypto?.getRandomValues === 'function') {
    const bytes = new Uint8Array(16);
    globalThis.crypto.getRandomValues(bytes);
    bytes[6] = (bytes[6] & 0x0f) | 0x40;
    bytes[8] = (bytes[8] & 0x3f) | 0x80;
    const hex = Array.from(bytes, (b) => b.toString(16).padStart(2, '0')).join(
      ''
    );
    return `${hex.slice(0, 8)}-${hex.slice(8, 12)}-${hex.slice(
      12,
      16
    )}-${hex.slice(16, 20)}-${hex.slice(20)}`;
  }

  // Non-cryptographic fallback for older/insecure contexts; IDs are UI-local only.
  const rand = () =>
    Math.floor((1 + Math.random()) * 0x10000)
      .toString(16)
      .slice(1);
  return `${rand()}${rand()}-${rand()}-4${rand().slice(1)}-${(
    (8 + Math.random() * 4) |
    0
  ).toString(16)}${rand().slice(1)}-${rand()}${rand()}${rand()}`;
}

export function titleFromMessage(text: string): string {
  const line = text.trim().split('\n')[0] ?? '';
  const t = line.length > 48 ? `${line.slice(0, 45)}…` : line;
  return t || 'New chat';
}

/** Sidebar title: first user message in API order, else "New chat". */
export function sessionTitleFromApiRows(
  rows: { id: string; user: string; content: string }[]
): string {
  const firstUser = rows.find((m) => m.user === 'user');
  if (!firstUser) return 'New chat';
  return titleFromMessage(firstUser.content);
}

export function apiMessagesToMessages(
  rows: { id: string; user: string; content: string }[]
): Message[] {
  return rows.map((m) => ({
    id: m.id || newClientMessageId(),
    role: m.user === 'assistant' ? 'assistant' : 'user',
    content: m.content,
  }));
}

/** POST /api/chat and GET /api/chat/{id} share the same JSON shape. */
export function chatApiResponseToSession(data: unknown): ChatSession | null {
  if (!data || typeof data !== 'object') return null;
  const rec = data as Record<string, unknown>;
  const sid = rec.id;
  const msgs = rec.messages;
  if (typeof sid !== 'string' || !Array.isArray(msgs)) return null;
  const rawRows = msgs.filter(
    (m): m is { id: string; user: string; content: string } =>
      !!m &&
      typeof m === 'object' &&
      typeof (m as { id?: unknown }).id === 'string' &&
      typeof (m as { user?: unknown }).user === 'string' &&
      typeof (m as { content?: unknown }).content === 'string'
  );
  const title = sessionTitleFromApiRows(rawRows);
  const messages = apiMessagesToMessages(rawRows);
  return { id: sid, title, messages };
}

export { newClientMessageId as newMessageId };
