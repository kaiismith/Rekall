/** Public representation of an organization. */
export interface Organization {
  id: string
  name: string
  slug: string
  owner_id: string
  created_at: string
  updated_at: string
}

/** Public representation of an org membership. */
export interface OrgMember {
  user_id: string
  org_id: string
  role: 'owner' | 'admin' | 'member'
  joined_at: string
}

/** Body for POST /organizations. */
export interface CreateOrgPayload {
  name: string
  /** Set by platform admins creating on behalf of another user. */
  owner_email?: string
}

/** Body for PATCH /organizations/:id. */
export interface UpdateOrgPayload {
  name: string
}

/** Body for POST /organizations/:id/invitations. */
export interface InviteUserPayload {
  email: string
  role?: 'admin' | 'member'
}

/** Body for PATCH /organizations/:id/members/:userID. */
export interface UpdateMemberRolePayload {
  role: 'admin' | 'member'
}

/** Body for POST /invitations/accept. */
export interface AcceptInvitationPayload {
  token: string
}

/** Public representation of a department. */
export interface Department {
  id: string
  org_id: string
  name: string
  description: string
  created_by: string
  created_at: string
  updated_at: string
}

/** Public representation of a department membership. */
export interface DeptMember {
  user_id: string
  department_id: string
  role: 'head' | 'member'
  joined_at: string
}

/** Body for POST /organizations/:id/departments. */
export interface CreateDeptPayload {
  name: string
  description?: string
}

/** Body for PATCH /departments/:id. */
export interface UpdateDeptPayload {
  name: string
  description?: string
}

/** Body for POST /departments/:id/members. */
export interface AddDeptMemberPayload {
  user_id: string
  role?: 'head' | 'member'
}

/** Body for PATCH /departments/:id/members/:userID. */
export interface UpdateDeptMemberRolePayload {
  role: 'head' | 'member'
}
