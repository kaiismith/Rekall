import { describe, it, expect, beforeEach } from 'vitest'
import { useUIStore } from '@/store/uiStore'

describe('uiStore', () => {
  beforeEach(() => {
    useUIStore.setState({ sidebarOpen: true, notifications: [] })
  })

  it('starts with sidebarOpen=true and empty notifications', () => {
    const { sidebarOpen, notifications } = useUIStore.getState()
    expect(sidebarOpen).toBe(true)
    expect(notifications).toHaveLength(0)
  })

  it('toggleSidebar() flips sidebarOpen', () => {
    useUIStore.getState().toggleSidebar()
    expect(useUIStore.getState().sidebarOpen).toBe(false)
    useUIStore.getState().toggleSidebar()
    expect(useUIStore.getState().sidebarOpen).toBe(true)
  })

  it('setSidebarOpen() sets sidebarOpen to the given value', () => {
    useUIStore.getState().setSidebarOpen(false)
    expect(useUIStore.getState().sidebarOpen).toBe(false)
    useUIStore.getState().setSidebarOpen(true)
    expect(useUIStore.getState().sidebarOpen).toBe(true)
  })

  it('addNotification() appends a notification with a generated id', () => {
    useUIStore.getState().addNotification({ type: 'success', message: 'Saved!' })
    const { notifications } = useUIStore.getState()
    expect(notifications).toHaveLength(1)
    expect(notifications[0].type).toBe('success')
    expect(notifications[0].message).toBe('Saved!')
    expect(notifications[0].id).toBeTruthy()
  })

  it('addNotification() assigns unique ids to multiple notifications', () => {
    useUIStore.getState().addNotification({ type: 'info', message: 'A' })
    useUIStore.getState().addNotification({ type: 'error', message: 'B' })
    const { notifications } = useUIStore.getState()
    expect(notifications).toHaveLength(2)
    expect(notifications[0].id).not.toBe(notifications[1].id)
  })

  it('removeNotification() removes the notification with the given id', () => {
    useUIStore.getState().addNotification({ type: 'warning', message: 'Watch out' })
    const id = useUIStore.getState().notifications[0].id
    useUIStore.getState().removeNotification(id)
    expect(useUIStore.getState().notifications).toHaveLength(0)
  })

  it('removeNotification() with unknown id leaves notifications unchanged', () => {
    useUIStore.getState().addNotification({ type: 'info', message: 'Hi' })
    useUIStore.getState().removeNotification('nonexistent-id')
    expect(useUIStore.getState().notifications).toHaveLength(1)
  })

  it('removeNotification() only removes the matching notification', () => {
    useUIStore.getState().addNotification({ type: 'info', message: 'A' })
    useUIStore.getState().addNotification({ type: 'error', message: 'B' })
    const firstId = useUIStore.getState().notifications[0].id
    useUIStore.getState().removeNotification(firstId)
    const { notifications } = useUIStore.getState()
    expect(notifications).toHaveLength(1)
    expect(notifications[0].message).toBe('B')
  })
})
