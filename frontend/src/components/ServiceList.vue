<template>
  <aside class="sidebar">
    <div class="sidebar-header">
      <span>{{ t('sidebar.services', { count: displayServices.length }) }}</span>
      <div class="sidebar-actions">
        <button class="btn-sm btn-secondary" @click="$emit('new')">{{ t('sidebar.newBtn') }}</button>
        <button class="btn-sm btn-primary" @click="$emit('refresh')">{{ t('sidebar.refreshBtn') }}</button>
      </div>
    </div>
    <div class="service-list" v-if="displayServices.length > 0">
      <div
        v-for="svc in displayServices"
        :key="svc.name + '-' + svc.source"
        class="service-item"
        :class="{ active: selectedService === svc.name }"
        @click="$emit('select', svc)"
      >
        <div class="service-item-row">
          <div class="service-item-info">
            <div class="service-item-name">
              {{ svc.displayName || svc.name }}
              <span v-if="svc.source === 'file'" class="source-badge">File</span>
            </div>
            <div class="service-item-meta">
              <span class="status-badge" :class="statusClass(svc.status)">{{ statusLabel(svc.status) }}</span>
              <span v-if="svc.syncState" class="sync-badge" :class="syncClass(svc.syncState)">{{ t('sync.' + svc.syncState) }}</span>
              <span class="start-type">{{ svc.startType }}</span>
            </div>
          </div>
          <button class="btn-sm btn-copy" @click.stop="$emit('copy', svc)" :title="t('sidebar.copyTooltip')">
            &#x2398;
          </button>
        </div>
      </div>
    </div>
    <div v-else class="empty-state">
      <p>{{ t('sidebar.empty') }}</p>
      <p class="hint">{{ t('sidebar.emptyHint') }}</p>
    </div>
  </aside>
</template>

<script>
import { useI18n } from 'vue-i18n'

export default {
  name: 'ServiceList',
  props: {
    displayServices: { type: Array, required: true },
    selectedService: { type: String, default: '' },
  },
  emits: ['new', 'refresh', 'select', 'copy'],
  setup() {
    const { t } = useI18n()

    function statusLabel(status) {
      if (status === 'Orphan') return t('sync.orphaned')
      return status
    }

    function statusClass(status) {
      switch (status) {
        case 'Running': return 'status-running'
        case 'Stopped': return 'status-stopped'
        case 'Not Installed': return 'status-file'
        case 'Orphan': return 'status-orphan'
        default: return 'status-other'
      }
    }

    function syncClass(state) {
      switch (state) {
        case 'synced': return 'sync-synced'
        case 'drifted': return 'sync-drifted'
        default: return 'sync-other'
      }
    }

    return { t, statusLabel, statusClass, syncClass }
  },
}
</script>

<style scoped>
.sidebar {
  width: 280px;
  min-width: 280px;
  background: var(--bg-secondary);
  border-right: 1px solid var(--border);
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.sidebar-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 16px;
  font-weight: 600;
  font-size: 14px;
  border-bottom: 1px solid var(--border);
}

.sidebar-actions {
  display: flex;
  gap: 6px;
}

.service-list {
  flex: 1;
  overflow-y: auto;
  padding: 6px;
}

.service-item {
  padding: 10px 12px;
  border-radius: var(--radius);
  cursor: pointer;
  transition: background 0.15s;
  margin-bottom: 2px;
}
.service-item:hover { background: var(--bg-hover); }
.service-item:hover .btn-copy { opacity: 1; }
.service-item.active { background: var(--bg-active); }

.service-item-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 4px;
}

.service-item-info {
  flex: 1;
  min-width: 0;
}

.btn-copy {
  opacity: 0;
  transition: opacity 0.15s;
  font-size: 14px;
  padding: 2px 6px;
  flex-shrink: 0;
  background: transparent;
  border: 1px solid var(--border);
  border-radius: 4px;
  color: var(--text-muted);
  cursor: pointer;
  line-height: 1;
}
.btn-copy:hover {
  background: var(--bg-hover);
  color: var(--accent);
  border-color: var(--accent);
}

.service-item-name {
  font-weight: 500;
  margin-bottom: 4px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  display: flex;
  align-items: center;
  gap: 6px;
}

.source-badge {
  font-size: 10px;
  font-weight: 600;
  padding: 1px 5px;
  border-radius: 3px;
  background: rgba(33, 150, 243, 0.2);
  color: var(--info, #2196F3);
}

.service-item-meta {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 12px;
  flex-wrap: wrap;
}

.status-badge {
  padding: 1px 6px;
  border-radius: 3px;
  font-size: 11px;
  font-weight: 600;
}
.status-running { background: rgba(76, 175, 80, 0.2); color: var(--success); }
.status-stopped { background: rgba(244, 67, 54, 0.2); color: var(--danger); }
.status-file { background: rgba(158, 158, 158, 0.2); color: var(--text-muted); }
.status-orphan { background: rgba(244, 67, 54, 0.2); color: var(--danger); }
.status-other { background: rgba(255, 152, 0, 0.2); color: var(--warning); }
.start-type { color: var(--text-muted); }

.sync-badge {
  padding: 1px 6px;
  border-radius: 3px;
  font-size: 11px;
  font-weight: 600;
}
.sync-synced { background: rgba(76, 175, 80, 0.2); color: var(--success); }
.sync-drifted { background: rgba(255, 152, 0, 0.2); color: var(--warning); }
.sync-other { background: rgba(158, 158, 158, 0.2); color: var(--text-muted); }

.empty-state {
  padding: 40px 20px;
  text-align: center;
  color: var(--text-muted);
}
.empty-state .hint {
  font-size: 12px;
  margin-top: 6px;
}
</style>
