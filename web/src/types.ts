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

export type ChatMessageResponse = {
  message: Message;
};
