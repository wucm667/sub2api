import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { flushPromises, mount } from '@vue/test-utils'

import AccountsView from '../AccountsView.vue'

const {
  listAccounts,
  listWithEtag,
  getBatchTodayStats,
  getAllProxies,
  getAllGroups
} = vi.hoisted(() => ({
  listAccounts: vi.fn(),
  listWithEtag: vi.fn(),
  getBatchTodayStats: vi.fn(),
  getAllProxies: vi.fn(),
  getAllGroups: vi.fn()
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    accounts: {
      list: listAccounts,
      listWithEtag,
      getBatchTodayStats,
      delete: vi.fn(),
      batchClearError: vi.fn(),
      batchRefresh: vi.fn(),
      toggleSchedulable: vi.fn()
    },
    proxies: {
      getAll: getAllProxies
    },
    groups: {
      getAll: getAllGroups
    }
  }
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError: vi.fn(),
    showSuccess: vi.fn(),
    showInfo: vi.fn()
  })
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => ({
    token: 'test-token'
  })
}))

vi.mock('vue-i18n', async () => {
  const actual = await vi.importActual<typeof import('vue-i18n')>('vue-i18n')
  return {
    ...actual,
    useI18n: () => ({
      t: (key: string) => key
    })
  }
})

const DataTableStub = {
  props: ['columns', 'data'],
  template: '<div data-test="data-table"></div>'
}

const CreateAccountModalStub = {
  name: 'CreateAccountModal',
  props: ['show', 'proxies', 'groups'],
  template: '<div data-test="create-account-modal"></div>'
}

function mountView() {
  return mount(AccountsView, {
    global: {
      stubs: {
        AppLayout: { template: '<div><slot /></div>' },
        TablePageLayout: {
          template: '<div><slot name="filters" /><slot name="table" /><slot name="pagination" /></div>'
        },
        DataTable: DataTableStub,
        HelpTooltip: true,
        Pagination: true,
        ConfirmDialog: true,
        AccountTableActions: { template: '<div><slot name="beforeCreate" /><slot name="after" /></div>' },
        AccountTableFilters: { template: '<div></div>' },
        AccountBulkActionsBar: true,
        AccountActionMenu: true,
        ImportDataModal: true,
        ReAuthAccountModal: true,
        AccountTestModal: true,
        AccountStatsModal: true,
        ScheduledTestsPanel: true,
        SyncFromCrsModal: true,
        TempUnschedStatusModal: true,
        ErrorPassthroughRulesModal: true,
        TLSFingerprintProfilesModal: true,
        CreateAccountModal: CreateAccountModalStub,
        EditAccountModal: true,
        BulkEditAccountModal: true,
        PlatformTypeBadge: true,
        AccountCapacityCell: true,
        AccountStatusIndicator: true,
        AccountTodayStatsCell: true,
        AccountGroupsCell: true,
        AccountUsageCell: true,
        Icon: true
      }
    }
  })
}

const openaiGroup = {
  id: 1,
  name: 'openai-default',
  platform: 'openai',
  status: 'active'
}

const accountProxy = {
  id: 7,
  name: 'proxy-a',
  url: 'http://proxy.example',
  status: 'active'
}

describe('admin AccountsView helper data loading', () => {
  beforeEach(() => {
    localStorage.clear()
    vi.spyOn(console, 'error').mockImplementation(() => undefined)

    listAccounts.mockReset()
    listWithEtag.mockReset()
    getBatchTodayStats.mockReset()
    getAllProxies.mockReset()
    getAllGroups.mockReset()

    listAccounts.mockResolvedValue({
      items: [],
      total: 0,
      page: 1,
      page_size: 20,
      pages: 0
    })
    listWithEtag.mockResolvedValue({
      notModified: true,
      etag: null,
      data: null
    })
    getBatchTodayStats.mockResolvedValue({ stats: {} })
    getAllProxies.mockResolvedValue([])
    getAllGroups.mockResolvedValue([])
  })

  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('keeps groups available for account creation when proxy loading fails', async () => {
    const proxyError = new Error('proxy unavailable')
    getAllProxies.mockRejectedValue(proxyError)
    getAllGroups.mockResolvedValue([openaiGroup])

    const wrapper = mountView()
    await flushPromises()

    const modal = wrapper.getComponent(CreateAccountModalStub)
    expect(modal.props('groups')).toEqual([openaiGroup])
    expect(console.error).toHaveBeenCalledWith('Failed to load proxies:', proxyError)
  })

  it('keeps proxies available when group loading fails', async () => {
    const groupError = new Error('groups unavailable')
    getAllProxies.mockResolvedValue([accountProxy])
    getAllGroups.mockRejectedValue(groupError)

    const wrapper = mountView()
    await flushPromises()

    const modal = wrapper.getComponent(CreateAccountModalStub)
    expect(modal.props('proxies')).toEqual([accountProxy])
    expect(console.error).toHaveBeenCalledWith('Failed to load groups:', groupError)
  })

  it('passes both proxies and groups through when both helper requests succeed', async () => {
    getAllProxies.mockResolvedValue([accountProxy])
    getAllGroups.mockResolvedValue([openaiGroup])

    const wrapper = mountView()
    await flushPromises()

    const modal = wrapper.getComponent(CreateAccountModalStub)
    expect(modal.props('proxies')).toEqual([accountProxy])
    expect(modal.props('groups')).toEqual([openaiGroup])
    expect(console.error).not.toHaveBeenCalled()
  })
})
