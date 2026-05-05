export type Role = 'user' | 'assistant';

export interface Message {
  id: string;
  role: Role;
  content: string;
}

export interface ChatSession {
  id: string;
  title: string;
  messages: Message[];
}

/** Backend POST /api/chat — message shape (field is `user`, not `role`). */
export type ChatApiMessageDTO = {
  user: string;
  content: string;
};

/** Backend POST /api/chat response */
export type CreateChatApiResponse = {
  id: string;
  messages: ChatApiMessageDTO[];
};

/** Backend POST /api/chat/{chat_id} response */
export type ProcessChatApiResponse = {
  message: ChatApiMessageDTO;
};
