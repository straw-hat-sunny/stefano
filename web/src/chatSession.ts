import type { ChatSession, Message } from './types';

function newClientMessageId(): string {
  return crypto.randomUUID();
}

export function titleFromMessage(text: string): string {
  const line = text.trim().split('\n')[0] ?? '';
  const t = line.length > 48 ? `${line.slice(0, 45)}…` : line;
  return t || 'New chat';
}

/** Sidebar title: first user message in API order, else "New chat". */
export function sessionTitleFromApiRows(
  rows: { user: string; content: string }[]
): string {
  const firstUser = rows.find((m) => m.user === 'user');
  if (!firstUser) return 'New chat';
  return titleFromMessage(firstUser.content);
}

export function apiMessagesToMessages(
  rows: { user: string; content: string }[]
): Message[] {
  return rows.map((m) => ({
    id: newClientMessageId(),
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
    (m): m is { user: string; content: string } =>
      !!m &&
      typeof m === 'object' &&
      typeof (m as { user?: unknown }).user === 'string' &&
      typeof (m as { content?: unknown }).content === 'string'
  );
  const title = sessionTitleFromApiRows(rawRows);
  const messages = apiMessagesToMessages(rawRows);
  return { id: sid, title, messages };
}

export { newClientMessageId as newMessageId };
