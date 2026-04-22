import { describe, it, expect, vi, beforeEach } from 'vitest'
import { organizationService } from '@/services/organizationService'

vi.mock('@/services/api', () => ({
  apiClient: {
    get: vi.fn(),
    post: vi.fn(),
    patch: vi.fn(),
    delete: vi.fn(),
  },
}))

import { apiClient } from '@/services/api'

const mockOrg = { id: 'org-1', name: 'Acme', created_at: '2026-01-01T00:00:00Z' }
const mockMember = { org_id: 'org-1', user_id: 'user-1', role: 'member', joined_at: '2026-01-01T00:00:00Z' }
const mockDept = { id: 'dept-1', org_id: 'org-1', name: 'Engineering' }
const mockDeptMember = { department_id: 'dept-1', user_id: 'user-1', role: 'member', joined_at: '2026-01-01T00:00:00Z' }

describe('organizationService', () => {
  beforeEach(() => vi.clearAllMocks())

  // ── Core CRUD ───────────────────────────────────────────────────────────────

  it('list() calls GET /organizations', async () => {
    vi.mocked(apiClient.get).mockResolvedValue({ data: { data: [mockOrg] } })
    const result = await organizationService.list()
    expect(apiClient.get).toHaveBeenCalledWith('/organizations')
    expect(result).toEqual([mockOrg])
  })

  it('create() calls POST /organizations', async () => {
    vi.mocked(apiClient.post).mockResolvedValue({ data: { data: mockOrg } })
    const result = await organizationService.create({ name: 'Acme' })
    expect(apiClient.post).toHaveBeenCalledWith('/organizations', { name: 'Acme' })
    expect(result).toEqual(mockOrg)
  })

  it('get() calls GET /organizations/:id', async () => {
    vi.mocked(apiClient.get).mockResolvedValue({ data: { data: mockOrg } })
    const result = await organizationService.get('org-1')
    expect(apiClient.get).toHaveBeenCalledWith('/organizations/org-1')
    expect(result).toEqual(mockOrg)
  })

  it('update() calls PATCH /organizations/:id', async () => {
    const updated = { ...mockOrg, name: 'Acme Corp' }
    vi.mocked(apiClient.patch).mockResolvedValue({ data: { data: updated } })
    const result = await organizationService.update('org-1', { name: 'Acme Corp' })
    expect(apiClient.patch).toHaveBeenCalledWith('/organizations/org-1', { name: 'Acme Corp' })
    expect(result).toEqual(updated)
  })

  it('delete() calls DELETE /organizations/:id', async () => {
    vi.mocked(apiClient.delete).mockResolvedValue({})
    await organizationService.delete('org-1')
    expect(apiClient.delete).toHaveBeenCalledWith('/organizations/org-1')
  })

  // ── Members ─────────────────────────────────────────────────────────────────

  it('listMembers() calls GET /organizations/:id/members', async () => {
    vi.mocked(apiClient.get).mockResolvedValue({ data: { data: [mockMember] } })
    const result = await organizationService.listMembers('org-1')
    expect(apiClient.get).toHaveBeenCalledWith('/organizations/org-1/members')
    expect(result).toEqual([mockMember])
  })

  it('updateMember() calls PATCH /organizations/:orgId/members/:userId', async () => {
    vi.mocked(apiClient.patch).mockResolvedValue({})
    await organizationService.updateMember('org-1', 'user-1', { role: 'admin' })
    expect(apiClient.patch).toHaveBeenCalledWith('/organizations/org-1/members/user-1', { role: 'admin' })
  })

  it('removeMember() calls DELETE /organizations/:orgId/members/:userId', async () => {
    vi.mocked(apiClient.delete).mockResolvedValue({})
    await organizationService.removeMember('org-1', 'user-1')
    expect(apiClient.delete).toHaveBeenCalledWith('/organizations/org-1/members/user-1')
  })

  // ── Invitations ──────────────────────────────────────────────────────────────

  it('inviteUser() calls POST /organizations/:id/invitations', async () => {
    vi.mocked(apiClient.post).mockResolvedValue({})
    await organizationService.inviteUser('org-1', { email: 'a@b.com' })
    expect(apiClient.post).toHaveBeenCalledWith('/organizations/org-1/invitations', { email: 'a@b.com' })
  })

  it('acceptInvitation() calls POST /invitations/accept', async () => {
    vi.mocked(apiClient.post).mockResolvedValue({ data: { data: mockOrg } })
    const result = await organizationService.acceptInvitation({ token: 'tok-1' })
    expect(apiClient.post).toHaveBeenCalledWith('/invitations/accept', { token: 'tok-1' })
    expect(result).toEqual(mockOrg)
  })

  // ── Departments ──────────────────────────────────────────────────────────────

  it('listDepartments() calls GET /organizations/:id/departments', async () => {
    vi.mocked(apiClient.get).mockResolvedValue({ data: { data: [mockDept] } })
    const result = await organizationService.listDepartments('org-1')
    expect(apiClient.get).toHaveBeenCalledWith('/organizations/org-1/departments')
    expect(result).toEqual([mockDept])
  })

  it('createDepartment() calls POST /organizations/:id/departments', async () => {
    vi.mocked(apiClient.post).mockResolvedValue({ data: { data: mockDept } })
    const result = await organizationService.createDepartment('org-1', { name: 'Engineering' })
    expect(apiClient.post).toHaveBeenCalledWith('/organizations/org-1/departments', { name: 'Engineering' })
    expect(result).toEqual(mockDept)
  })

  it('getDepartment() calls GET /departments/:id', async () => {
    vi.mocked(apiClient.get).mockResolvedValue({ data: { data: mockDept } })
    const result = await organizationService.getDepartment('dept-1')
    expect(apiClient.get).toHaveBeenCalledWith('/departments/dept-1')
    expect(result).toEqual(mockDept)
  })

  it('updateDepartment() calls PATCH /departments/:id', async () => {
    const updated = { ...mockDept, name: 'Eng' }
    vi.mocked(apiClient.patch).mockResolvedValue({ data: { data: updated } })
    const result = await organizationService.updateDepartment('dept-1', { name: 'Eng' })
    expect(apiClient.patch).toHaveBeenCalledWith('/departments/dept-1', { name: 'Eng' })
    expect(result).toEqual(updated)
  })

  it('deleteDepartment() calls DELETE /departments/:id', async () => {
    vi.mocked(apiClient.delete).mockResolvedValue({})
    await organizationService.deleteDepartment('dept-1')
    expect(apiClient.delete).toHaveBeenCalledWith('/departments/dept-1')
  })

  // ── Department members ───────────────────────────────────────────────────────

  it('listDeptMembers() calls GET /departments/:id/members', async () => {
    vi.mocked(apiClient.get).mockResolvedValue({ data: { data: [mockDeptMember] } })
    const result = await organizationService.listDeptMembers('dept-1')
    expect(apiClient.get).toHaveBeenCalledWith('/departments/dept-1/members')
    expect(result).toEqual([mockDeptMember])
  })

  it('addDeptMember() calls POST /departments/:id/members', async () => {
    vi.mocked(apiClient.post).mockResolvedValue({})
    await organizationService.addDeptMember('dept-1', { user_id: 'user-1' })
    expect(apiClient.post).toHaveBeenCalledWith('/departments/dept-1/members', { user_id: 'user-1' })
  })

  it('updateDeptMember() calls PATCH /departments/:id/members/:userId', async () => {
    vi.mocked(apiClient.patch).mockResolvedValue({})
    await organizationService.updateDeptMember('dept-1', 'user-1', { role: 'head' })
    expect(apiClient.patch).toHaveBeenCalledWith('/departments/dept-1/members/user-1', { role: 'head' })
  })

  it('removeDeptMember() calls DELETE /departments/:id/members/:userId', async () => {
    vi.mocked(apiClient.delete).mockResolvedValue({})
    await organizationService.removeDeptMember('dept-1', 'user-1')
    expect(apiClient.delete).toHaveBeenCalledWith('/departments/dept-1/members/user-1')
  })
})
