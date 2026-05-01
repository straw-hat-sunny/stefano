import {
  useCallback,
  useEffect,
  useLayoutEffect,
  useMemo,
  useRef,
  useState,
  type KeyboardEvent as ReactKeyboardEvent,
} from 'react';
import type { ListChildComponentProps } from 'react-window';
import { FixedSizeList } from 'react-window';
import styles from './App.module.css';
import { STEFANO_SYSTEM_PROMPT } from './chatConstants';
import type { ChatMessageResponse, ChatSession, Message } from './types';
import type { ModelOption, ModelsListResponse } from './models';

const SESSION_ROW_HEIGHT = 44;

function id(): string {
  return crypto.randomUUID();
}

function stefanoSystemMessage(): Message {
  return { id: id(), role: 'system', content: STEFANO_SYSTEM_PROMPT };
}

function titleFromMessage(text: string): string {
  const line = text.trim().split('\n')[0] ?? '';
  const t = line.length > 48 ? `${line.slice(0, 45)}…` : line;
  return t || 'New chat';
}

type RowData = {
  sessions: ChatSession[];
  activeId: string | null;
  onSelect: (sessionId: string) => void;
};

function SessionRow({ index, style, data }: ListChildComponentProps<RowData>) {
  const session = data.sessions[index];
  if (!session) return null;
  const active = session.id === data.activeId;
  return (
    <div style={style} className={styles.sessionRow}>
      <button
        type="button"
        className={`${styles.sessionBtn} ${active ? styles.sessionBtnActive : ''}`}
        onClick={() => data.onSelect(session.id)}
      >
        {session.title}
      </button>
    </div>
  );
}

export default function App() {
  const initialSessionId = useMemo(() => id(), []);
  const [sessions, setSessions] = useState<ChatSession[]>(() => [
    { id: initialSessionId, title: 'New chat', messages: [stefanoSystemMessage()] },
  ]);
  const [activeId, setActiveId] = useState<string | null>(initialSessionId);
  const [models, setModels] = useState<ModelOption[]>([]);
  const [model, setModel] = useState('');
  const [modelsLoadError, setModelsLoadError] = useState<string | null>(null);
  const [draft, setDraft] = useState('');
  const [sidebarOpen, setSidebarOpen] = useState(false);
  const [isMobile, setIsMobile] = useState(false);
  const [pendingSessionId, setPendingSessionId] = useState<string | null>(null);
  const [sendError, setSendError] = useState<string | null>(null);

  const listWrapRef = useRef<HTMLDivElement>(null);
  const [listHeight, setListHeight] = useState(320);
  const messagesEndRef = useRef<HTMLDivElement>(null);

  const activeSession = useMemo(
    () => sessions.find((s) => s.id === activeId) ?? null,
    [sessions, activeId]
  );

  useEffect(() => {
    const mq = window.matchMedia('(max-width: 767px)');
    const update = () => setIsMobile(mq.matches);
    update();
    mq.addEventListener('change', update);
    return () => mq.removeEventListener('change', update);
  }, []);

  useLayoutEffect(() => {
    const el = listWrapRef.current;
    if (!el) return;
    const ro = new ResizeObserver(() => {
      setListHeight(Math.max(120, el.clientHeight));
    });
    ro.observe(el);
    setListHeight(Math.max(120, el.clientHeight));
    return () => ro.disconnect();
  }, []);

  useEffect(() => {
    messagesEndRef.current?.scrollIntoView({ behavior: 'smooth', block: 'end' });
  }, [activeSession?.messages, pendingSessionId]);

  useEffect(() => {
    if (!sidebarOpen || !isMobile) return;
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') setSidebarOpen(false);
    };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [sidebarOpen, isMobile]);

  useEffect(() => {
    let cancelled = false;
    (async () => {
      try {
        const res = await fetch('/api/models');
        if (!res.ok) {
          throw new Error(`HTTP ${res.status}`);
        }
        const data = (await res.json()) as ModelsListResponse;
        if (cancelled) return;
        if (!Array.isArray(data.models) || data.models.length === 0) {
          setModelsLoadError('No models available.');
          setModels([]);
          setModel('');
          return;
        }
        setModels(data.models);
        setModelsLoadError(null);
        const sel = data.models.some((m) => m.id === data.selectedId)
          ? data.selectedId
          : data.models[0].id;
        setModel(sel);
      } catch {
        if (!cancelled) {
          setModelsLoadError('Could not load models.');
          setModels([]);
          setModel('');
        }
      }
    })();
    return () => {
      cancelled = true;
    };
  }, []);

  const closeSidebarMobile = useCallback(() => {
    if (isMobile) setSidebarOpen(false);
  }, [isMobile]);

  const handleSelectSession = useCallback(
    (sessionId: string) => {
      setActiveId(sessionId);
      closeSidebarMobile();
    },
    [closeSidebarMobile]
  );

  const handleNewChat = useCallback(() => {
    const next: ChatSession = { id: id(), title: 'New chat', messages: [stefanoSystemMessage()] };
    setSessions((prev) => [next, ...prev]);
    setActiveId(next.id);
    closeSidebarMobile();
  }, [closeSidebarMobile]);

  const appendAssistantReply = useCallback((sessionId: string, reply: Message) => {
    setSessions((prev) =>
      prev.map((s) =>
        s.id === sessionId ? { ...s, messages: [...s.messages, reply] } : s
      )
    );
  }, []);

  const handleSend = useCallback(() => {
    const trimmed = draft.trim();
    if (!trimmed || !activeId || pendingSessionId !== null) return;

    const sessionId = activeId;
    const session = sessions.find((s) => s.id === sessionId);
    const priorMessages = session?.messages ?? [];
    const apiMessages = [
      ...priorMessages.map((m) => ({ role: m.role, content: m.content })),
      { role: 'user' as const, content: trimmed },
    ];
    const userMsg: Message = { id: id(), role: 'user', content: trimmed };
    setDraft('');
    setSendError(null);
    setPendingSessionId(sessionId);

    setSessions((prev) =>
      prev.map((s) => {
        if (s.id !== sessionId) return s;
        const nextMessages = [...s.messages, userMsg];
        const userTurnsBefore = s.messages.filter((m) => m.role === 'user').length;
        const nextTitle =
          s.title === 'New chat' && userTurnsBefore === 0
            ? titleFromMessage(trimmed)
            : s.title;
        return { ...s, title: nextTitle, messages: nextMessages };
      })
    );

    void (async () => {
      try {
        const res = await fetch('/api/chat/message', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({
            content: trimmed,
            modelId: model,
            messages: apiMessages,
          }),
        });
        if (!res.ok) {
          let msg = `Request failed (${res.status})`;
          try {
            const errBody = (await res.json()) as { error?: string };
            if (typeof errBody.error === 'string' && errBody.error) {
              msg = errBody.error;
            }
          } catch {
            /* ignore */
          }
          setSendError(msg);
          return;
        }
        const data = (await res.json()) as ChatMessageResponse;
        const m = data.message;
        if (
          !m ||
          typeof m.id !== 'string' ||
          m.role !== 'assistant' ||
          typeof m.content !== 'string'
        ) {
          setSendError('Invalid response from server.');
          return;
        }
        appendAssistantReply(sessionId, {
          id: m.id,
          role: 'assistant',
          content: m.content,
        });
      } catch {
        setSendError('Could not reach the server.');
      } finally {
        setPendingSessionId((cur) => (cur === sessionId ? null : cur));
      }
    })();
  }, [draft, activeId, pendingSessionId, model, sessions, appendAssistantReply]);

  const handleModelChange = useCallback(async (nextId: string) => {
    const prev = model;
    setModel(nextId);
    try {
      const res = await fetch('/api/model', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ id: nextId }),
      });
      if (!res.ok) {
        setModel(prev);
        return;
      }
      const body = (await res.json()) as ModelOption;
      if (body.id && body.label) {
        setModels((m) => {
          const i = m.findIndex((x) => x.id === body.id);
          if (i >= 0) {
            const copy = [...m];
            copy[i] = { id: body.id, label: body.label };
            return copy;
          }
          return m;
        });
      }
    } catch {
      setModel(prev);
    }
  }, [model]);

  const onKeyDownDraft = (e: ReactKeyboardEvent<HTMLTextAreaElement>) => {
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault();
      handleSend();
    }
  };

  const rowData: RowData = useMemo(
    () => ({
      sessions,
      activeId,
      onSelect: handleSelectSession,
    }),
    [sessions, activeId, handleSelectSession]
  );

  const visibleMessages = useMemo(() => {
    if (!activeSession) return [];
    return activeSession.messages.filter((m) => m.role !== 'system');
  }, [activeSession]);

  const drawerOpenClass = isMobile && sidebarOpen ? styles.drawerOpen : '';

  return (
    <div className={styles.app}>
      {isMobile && sidebarOpen ? (
        <button
          type="button"
          className={styles.backdrop}
          aria-label="Close conversation list"
          onClick={() => setSidebarOpen(false)}
        />
      ) : null}

      <aside
        id="chat-sidebar"
        className={`${styles.sidebar} ${drawerOpenClass}`}
        aria-hidden={isMobile && !sidebarOpen ? true : undefined}
        inert={isMobile && !sidebarOpen ? true : undefined}
      >
        <div className={styles.sidebarInner}>
          <button type="button" className={styles.newChat} onClick={handleNewChat}>
            New Chat
          </button>
          <h2 className={styles.historyHeading}>HISTORY</h2>
          <div ref={listWrapRef} className={styles.historyList}>
            {sessions.length > 0 && listHeight > 0 ? (
              <FixedSizeList
                height={listHeight}
                width="100%"
                itemCount={sessions.length}
                itemSize={SESSION_ROW_HEIGHT}
                itemData={rowData}
                overscanCount={8}
              >
                {SessionRow}
              </FixedSizeList>
            ) : null}
          </div>
        </div>
      </aside>

      <div className={styles.main}>
        <header className={styles.toolbar}>
          <button
            type="button"
            className={styles.menuToggle}
            aria-label={sidebarOpen ? 'Close conversation list' : 'Open conversation list'}
            aria-expanded={sidebarOpen}
            aria-controls="chat-sidebar"
            onClick={() => setSidebarOpen((o) => !o)}
          >
            <svg width="20" height="20" viewBox="0 0 24 24" aria-hidden focusable="false">
              <path
                fill="currentColor"
                d="M4 6h16v2H4V6zm0 5h16v2H4v-2zm0 5h16v2H4v-2z"
              />
            </svg>
          </button>
          <span className={styles.toolbarTitle}>
            {activeSession?.title ?? 'Conversation'}
          </span>
        </header>

        <main className={styles.messages} role="log" aria-live="polite" aria-relevant="additions">
          {activeSession && visibleMessages.length === 0 ? (
            <p className={styles.emptyState}>
              Start the conversation below. Messages stay in this session until you open another
              chat or start a new one.
            </p>
          ) : null}
          {visibleMessages.map((m) => (
            <div
              key={m.id}
              className={`${styles.bubbleRow} ${
                m.role === 'user' ? styles.bubbleRowUser : styles.bubbleRowAssistant
              }`}
            >
              <div
                className={`${styles.bubble} ${
                  m.role === 'user' ? styles.bubbleUser : styles.bubbleAssistant
                }`}
              >
                {m.content}
              </div>
            </div>
          ))}
          {activeSession && pendingSessionId === activeSession.id ? (
            <div className={`${styles.bubbleRow} ${styles.bubbleRowAssistant}`}>
              <div
                className={`${styles.bubble} ${styles.bubbleAssistant} ${styles.bubbleThinking}`}
              >
                ...
              </div>
            </div>
          ) : null}
          <div ref={messagesEndRef} />
        </main>

        <footer className={styles.footer}>
          <div className={styles.footerGrid}>
            <textarea
              id="chat-input"
              className={styles.input}
              placeholder="Message…"
              rows={3}
              value={draft}
              onChange={(e) => setDraft(e.target.value)}
              onKeyDown={onKeyDownDraft}
              aria-label="Chat message"
              disabled={pendingSessionId !== null}
            />
            <button
              type="button"
              className={styles.send}
              onClick={handleSend}
              disabled={!draft.trim() || !activeId || pendingSessionId !== null}
            >
              Send
            </button>
            {sendError ? (
              <span className={styles.sendError} role="alert">
                {sendError}
              </span>
            ) : null}
            <div className={styles.modelRow}>
              <label htmlFor="model-select" className={styles.modelLabel}>
                Model
              </label>
              <select
                id="model-select"
                className={styles.modelSelect}
                value={model}
                onChange={(e) => void handleModelChange(e.target.value)}
                disabled={models.length === 0}
                aria-label="Model"
                aria-invalid={modelsLoadError ? true : undefined}
              >
                {models.map((m) => (
                  <option key={m.id} value={m.id}>
                    {m.label}
                  </option>
                ))}
              </select>
              {modelsLoadError ? (
                <span className={styles.modelError} role="alert">
                  {modelsLoadError}
                </span>
              ) : null}
            </div>
          </div>
        </footer>
      </div>
    </div>
  );
}
