export type ModelOption = {
  id: string;
  label: string;
};

export type ModelsListResponse = {
  models: ModelOption[];
  selectedId: string;
};
