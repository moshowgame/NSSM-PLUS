<template>
  <footer class="action-bar">
    <div class="action-left">
      <button class="btn-secondary" @click="$emit('new')">
        <span class="icon">+</span> {{ t('action.newConfig') }}
      </button>
      <button class="btn-secondary" @click="$emit('import')">
        <span class="icon">&#x1F4C2;</span> {{ t('action.importConfig') }}
      </button>
      <button class="btn-secondary" @click="$emit('export')">
        <span class="icon">&#x1F4BE;</span> {{ t('action.exportConfig') }}
      </button>
      <button class="btn-primary" @click="$emit('save-service')" :disabled="!isEditing || !config.serviceName">
        <span class="icon">&#x1F4C4;</span> {{ t('action.saveService') }}
      </button>
    </div>
    <div class="action-right">
      <button class="btn-primary" @click="$emit('install')" :disabled="!config.serviceName || !config.appPath || (isEditing && source === 'installed')">
        {{ t('action.install') }}
      </button>
      <button class="btn-warning" @click="$emit('reconfigure')" :disabled="!isEditing || source !== 'installed'">
        {{ t('action.reconfigure') }}
      </button>
      <button class="btn-success" @click="$emit('start')" :disabled="!isEditing || source !== 'installed'">
        {{ t('action.start') }}
      </button>
      <button class="btn-warning" @click="$emit('stop')" :disabled="!isEditing || source !== 'installed'">
        {{ t('action.stop') }}
      </button>
      <button class="btn-secondary" @click="$emit('restart')" :disabled="!isEditing || source !== 'installed'">
        {{ t('action.restart') }}
      </button>
      <button class="btn-secondary" @click="$emit('check')" :disabled="!config.serviceName">
        {{ t('action.check') }}
      </button>
      <button class="btn-danger" @click="$emit('uninstall')" :disabled="!isEditing || source !== 'installed'">
        {{ t('action.uninstall') }}
      </button>
      <button class="btn-danger" @click="$emit('delete')" :disabled="!isEditing || source === 'orphan'">
        {{ t('action.delete') }}
      </button>
    </div>
  </footer>
</template>

<script>
import { useI18n } from 'vue-i18n'

export default {
  name: 'ActionBar',
  props: {
    config: { type: Object, required: true },
    isEditing: { type: Boolean, default: false },
    source: { type: String, default: '' },
  },
  emits: ['new', 'import', 'export', 'save-service', 'install', 'reconfigure', 'start', 'stop', 'restart', 'check', 'uninstall', 'delete'],
  setup() {
    const { t } = useI18n()
    return { t }
  },
}
</script>

<style scoped>
.action-bar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 20px;
  background: var(--bg-secondary);
  border-top: 1px solid var(--border);
  flex-shrink: 0;
}

.action-left, .action-right {
  display: flex;
  gap: 8px;
}

button:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

.icon {
  font-size: 15px;
}
</style>
