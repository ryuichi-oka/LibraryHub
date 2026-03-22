export type UserStatus = "ACTIVE" | "INACTIVE";

export type UpdatedUser = {
  id: string;
  employee_id: string;
  email: string;
  status: UserStatus;
  updated_at: string;
};

export type UpdateUserStatusApiUser = {
  id: string;
  status: UserStatus;
  updated_at: string;
};

export type UpdateUserStatusResponse = {
  user: UpdateUserStatusApiUser;
};

export type AdminUser = {
  id: string;
  employee_id: string;
  email: string;
  role: string;
  status: UserStatus;
};

export type AdminUsersResponse = {
  users: AdminUser[];
};

export type ErrorResponse = {
  error?: string;
};
