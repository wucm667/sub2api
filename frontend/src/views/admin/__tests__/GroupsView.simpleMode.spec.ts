import { describe, expect, it, vi, beforeEach } from 'vitest'
import { computed, defineComponent, nextTick } from 'vue'
import { mount, flushPromises } from '@vue/test-utils'

const {
  isSimpleModeRef,
  listGroupsMock,
  getModelsListCandidatesMock,
  getUsageSummaryMock,
  getCapacitySummaryMock,
  createGroupMock,
  updateGroupMock,
  deleteGroupMock
} = vi.hoisted(() => ({
  isSimpleModeRef: { value: true },
  listGroupsMock: vi.fn(),
  getModelsListCandidatesMock: vi.fn(),
  getUsageSummaryMock: vi.fn(),
  getCapacitySummaryMock: vi.fn(),
  createGroupMock: vi.fn(),
  updateGroupMock: vi.fn(),
  deleteGroupMock: vi.fn()
}))

vi.mock('@/stores/auth', () => ({
  useAuthStore: () => ({
    get isSimpleMode() {
      return isSimpleModeRef.value
    }
  })
}))

vi.mock('@/stores/app', () => ({
  useAppStore: () => ({
    showError: vi.fn(),
    showSuccess: vi.fn()
  })
}))

vi.mock('@/stores/onboarding', () => ({
  useOnboardingStore: () => ({
    isCurrentStep: vi.fn().mockReturnValue(false),
    nextStep: vi.fn()
  })
}))

vi.mock('@/api/admin', () => ({
  adminAPI: {
    groups: {
      list: listGroupsMock,
      getModelsListCandidates: getModelsListCandidatesMock,
      getUsageSummary: getUsageSummaryMock,
      getCapacitySummary: getCapacitySummaryMock,
      create: createGroupMock,
      update: updateGroupMock,
      delete: deleteGroupMock
    }
  }
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

vi.mock('vue-draggable-plus', () => ({
  VueDraggable: defineComponent({
    template: '<div><slot /></div>'
  })
}))

import GroupsView from '../GroupsView.vue'

const AppLayoutStub = defineComponent({
  template: '<div><slot /></div>'
})

const TablePageLayoutStub = defineComponent({
  template: '<div><slot name="filters" /><slot name="table" /><slot name="pagination" /></div>'
})

const DataTableStub = defineComponent({
  props: {
    columns: {
      type: Array,
      default: () => []
    }
  },
  setup(props) {
    const columnKeys = computed(() => (props.columns as Array<{ key: string }>).map((c) => c.key).join(','))
    return { columnKeys }
  },
  template: '<div data-testid="columns">{{ columnKeys }}</div>'
})

const BaseDialogStub = defineComponent({
  props: {
    show: {
      type: Boolean,
      default: false
    }
  },
  template: '<div v-if="show"><slot /><slot name="footer" /></div>'
})

const SelectStub = defineComponent({
  props: {
    modelValue: {
      type: [String, Number, Boolean, null],
      default: ''
    },
    options: {
      type: Array,
      default: () => []
    }
  },
  emits: ['update:modelValue'],
  template: '<select :value="modelValue" @change="$emit(\'update:modelValue\', $event.target.value)"><option v-for="option in options" :key="option.value" :value="option.value">{{ option.label }}</option></select>'
})

function mountGroupsView() {
  return mount(GroupsView, {
    global: {
      stubs: {
        AppLayout: AppLayoutStub,
        TablePageLayout: TablePageLayoutStub,
        DataTable: DataTableStub,
        Pagination: true,
        BaseDialog: BaseDialogStub,
        ConfirmDialog: true,
        EmptyState: true,
        Select: SelectStub,
        PlatformIcon: true,
        Icon: true,
        GroupRateMultipliersModal: true,
        GroupRPMOverridesModal: true,
        GroupCapacityBadge: true
      }
    }
  })
}

describe('GroupsView simple mode', () => {
  beforeEach(() => {
    isSimpleModeRef.value = true
    listGroupsMock.mockReset().mockResolvedValue({ items: [], total: 0, pages: 0 })
    getModelsListCandidatesMock.mockReset().mockResolvedValue([])
    getUsageSummaryMock.mockReset().mockResolvedValue([])
    getCapacitySummaryMock.mockReset().mockResolvedValue([])
    createGroupMock.mockReset().mockResolvedValue({})
    updateGroupMock.mockReset().mockResolvedValue({})
    deleteGroupMock.mockReset().mockResolvedValue({})
  })

  it('renders the minimal group field set in simple mode', async () => {
    const wrapper = mountGroupsView()
    await flushPromises()

    expect(wrapper.get('[data-testid="columns"]').text()).toContain('name,platform')
    expect(wrapper.get('[data-testid="columns"]').text()).not.toContain('rate_multiplier')
    expect(getUsageSummaryMock).not.toHaveBeenCalled()
    expect(getCapacitySummaryMock).not.toHaveBeenCalled()
    expect(getModelsListCandidatesMock).not.toHaveBeenCalled()

    await wrapper.get('[data-tour="groups-create-btn"]').trigger('click')
    await nextTick()

    expect(wrapper.text()).toContain('admin.groups.form.name')
    expect(wrapper.text()).toContain('admin.groups.form.description')
    expect(wrapper.text()).toContain('admin.groups.form.platform')
    expect(wrapper.text()).not.toContain('admin.groups.form.rateMultiplier')
    expect(wrapper.text()).not.toContain('admin.groups.subscription.type')
    expect(wrapper.text()).not.toContain('admin.groups.imagePricing.title')
  })

  it('renders the full commercial field set in standard mode', async () => {
    isSimpleModeRef.value = false
    const wrapper = mountGroupsView()
    await flushPromises()

    expect(wrapper.get('[data-testid="columns"]').text()).toContain('rate_multiplier')
    expect(getUsageSummaryMock).toHaveBeenCalledTimes(1)
    expect(getCapacitySummaryMock).toHaveBeenCalledTimes(1)
    expect(getModelsListCandidatesMock).toHaveBeenCalledTimes(1)

    await wrapper.get('[data-tour="groups-create-btn"]').trigger('click')
    await nextTick()

    expect(wrapper.text()).toContain('admin.groups.form.rateMultiplier')
    expect(wrapper.text()).toContain('admin.groups.subscription.type')
  })
})
