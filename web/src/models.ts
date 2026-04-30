export const MODELS = [
  { id: 'gpt-4.1', label: 'GPT-4.1' },
  { id: 'gpt-4o-mini', label: 'GPT-4o mini' },
  { id: 'claude-sonnet', label: 'Claude Sonnet' },
] as const;

export type ModelId = (typeof MODELS)[number]['id'];
