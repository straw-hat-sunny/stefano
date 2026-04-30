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
import type { ChatSession, Message } from './types';
import { MODELS, type ModelId } from './models';

const SESSION_ROW_HEIGHT = 44;
const MOCK_REPLY_MS = 720;

function id(): string {
  return crypto.randomUUID();
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
    { id: initialSessionId, title: 'New chat', messages: [] },
  ]);
  const [activeId, setActiveId] = useState<string | null>(initialSessionId);
  const [model, setModel] = useState<ModelId>(MODELS[0].id);
  const [draft, setDraft] = useState('');
  const [sidebarOpen, setSidebarOpen] = useState(false);
  const [isMobile, setIsMobile] = useState(false);

  const listWrapRef = useRef<HTMLDivElement>(null);
  const [listHeight, setListHeight] = useState(320);
  const messagesEndRef = useRef<HTMLDivElement>(null);
  const replyTimeouts = useRef<ReturnType<typeof setTimeout>[]>([]);

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
  }, [activeSession?.messages]);

  useEffect(() => {
    if (!sidebarOpen || !isMobile) return;
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') setSidebarOpen(false);
    };
    window.addEventListener('keydown', onKey);
    return () => window.removeEventListener('keydown', onKey);
  }, [sidebarOpen, isMobile]);

  useEffect(() => {
    return () => {
      replyTimeouts.current.forEach(clearTimeout);
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
    const next: ChatSession = { id: id(), title: 'New chat', messages: [] };
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
    if (!trimmed || !activeId) return;

    const sessionId = activeId;
    const userMsg: Message = { id: id(), role: 'user', content: trimmed };
    setDraft('');

    setSessions((prev) =>
      prev.map((s) => {
        if (s.id !== sessionId) return s;
        const nextMessages = [...s.messages, userMsg];
        const nextTitle =
          s.title === 'New chat' && s.messages.length === 0
            ? titleFromMessage(trimmed)
            : s.title;
        return { ...s, title: nextTitle, messages: nextMessages };
      })
    );

    const t = setTimeout(() => {
      appendAssistantReply(sessionId, {
        id: id(),
        role: 'assistant',
        content: `(${MODELS.find((m) => m.id === model)?.label ?? model}) This is a demo reply. Hook up your backend to stream real responses.`,
      });
    }, MOCK_REPLY_MS);
    replyTimeouts.current.push(t);
  }, [draft, activeId, model, appendAssistantReply]);

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
          {activeSession && activeSession.messages.length === 0 ? (
            <p className={styles.emptyState}>
              Start the conversation below. Messages stay in this session until you open another
              chat or start a new one.
            </p>
          ) : null}
          {activeSession?.messages.map((m) => (
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
            />
            <button
              type="button"
              className={styles.send}
              onClick={handleSend}
              disabled={!draft.trim() || !activeId}
            >
              Send
            </button>
            <div className={styles.modelRow}>
              <label htmlFor="model-select" className={styles.modelLabel}>
                Model
              </label>
              <select
                id="model-select"
                className={styles.modelSelect}
                value={model}
                onChange={(e) => setModel(e.target.value as ModelId)}
                aria-label="Model"
              >
                {MODELS.map((m) => (
                  <option key={m.id} value={m.id}>
                    {m.label}
                  </option>
                ))}
              </select>
            </div>
          </div>
        </footer>
      </div>
    </div>
  );
}
