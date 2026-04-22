import { apiClient } from './api'
import type { ApiResponse } from '@/types/common'
import type {
  Organization,
  OrgMember,
  CreateOrgPayload,
  UpdateOrgPayload,
  InviteUserPayload,
  UpdateMemberRolePayload,
  AcceptInvitationPayload,
  Department,
  DeptMember,
  CreateDeptPayload,
  UpdateDeptPayload,
  AddDeptMemberPayload,
  UpdateDeptMemberRolePayload,
} from '@/types/organization'

const BASE = '/organizations'

export const organizationService = {
  list: async (): Promise<Organization[]> => {
    const { data } = await apiClient.get<ApiResponse<Organization[]>>(BASE)
    return data.data
  },

  create: async (payload: CreateOrgPayload): Promise<Organization> => {
    const { data } = await apiClient.post<ApiResponse<Organization>>(BASE, payload)
    return data.data
  },

  get: async (id: string): Promise<Organization> => {
    const { data } = await apiClient.get<ApiResponse<Organization>>(`${BASE}/${id}`)
    return data.data
  },

  update: async (id: string, payload: UpdateOrgPayload): Promise<Organization> => {
    const { data } = await apiClient.patch<ApiResponse<Organization>>(`${BASE}/${id}`, payload)
    return data.data
  },

  delete: async (id: string): Promise<void> => {
    await apiClient.delete(`${BASE}/${id}`)
  },

  // Members
  listMembers: async (orgId: string): Promise<OrgMember[]> => {
    const { data } = await apiClient.get<ApiResponse<OrgMember[]>>(`${BASE}/${orgId}/members`)
    return data.data
  },

  updateMember: async (orgId: string, userId: string, payload: UpdateMemberRolePayload): Promise<void> => {
    await apiClient.patch(`${BASE}/${orgId}/members/${userId}`, payload)
  },

  removeMember: async (orgId: string, userId: string): Promise<void> => {
    await apiClient.delete(`${BASE}/${orgId}/members/${userId}`)
  },

  // Invitations
  inviteUser: async (orgId: string, payload: InviteUserPayload): Promise<void> => {
    await apiClient.post(`${BASE}/${orgId}/invitations`, payload)
  },

  acceptInvitation: async (payload: AcceptInvitationPayload): Promise<Organization> => {
    const { data } = await apiClient.post<ApiResponse<Organization>>('/invitations/accept', payload)
    return data.data
  },

  // Departments
  listDepartments: async (orgId: string): Promise<Department[]> => {
    const { data } = await apiClient.get<ApiResponse<Department[]>>(`${BASE}/${orgId}/departments`)
    return data.data
  },

  createDepartment: async (orgId: string, payload: CreateDeptPayload): Promise<Department> => {
    const { data } = await apiClient.post<ApiResponse<Department>>(`${BASE}/${orgId}/departments`, payload)
    return data.data
  },

  getDepartment: async (deptId: string): Promise<Department> => {
    const { data } = await apiClient.get<ApiResponse<Department>>(`/departments/${deptId}`)
    return data.data
  },

  updateDepartment: async (deptId: string, payload: UpdateDeptPayload): Promise<Department> => {
    const { data } = await apiClient.patch<ApiResponse<Department>>(`/departments/${deptId}`, payload)
    return data.data
  },

  deleteDepartment: async (deptId: string): Promise<void> => {
    await apiClient.delete(`/departments/${deptId}`)
  },

  // Department members
  listDeptMembers: async (deptId: string): Promise<DeptMember[]> => {
    const { data } = await apiClient.get<ApiResponse<DeptMember[]>>(`/departments/${deptId}/members`)
    return data.data
  },

  addDeptMember: async (deptId: string, payload: AddDeptMemberPayload): Promise<void> => {
    await apiClient.post(`/departments/${deptId}/members`, payload)
  },

  updateDeptMember: async (deptId: string, userId: string, payload: UpdateDeptMemberRolePayload): Promise<void> => {
    await apiClient.patch(`/departments/${deptId}/members/${userId}`, payload)
  },

  removeDeptMember: async (deptId: string, userId: string): Promise<void> => {
    await apiClient.delete(`/departments/${deptId}/members/${userId}`)
  },
}
