// Common types used across the application

export type ID = string;

export interface BaseEntity {
  id: ID;
  created_at: string;
  updated_at: string;
}

export interface PaginatedResponse<T> {
  data: T[];
  total: number;
  page: number;
  page_size: number;
}

export interface ApiError {
  error: string;
  details?: string;
}


