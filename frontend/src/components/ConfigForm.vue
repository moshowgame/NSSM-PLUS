<template>
  <main class="main-content">
    <div v-if="syncDiff" class="form-section sync-diff">
      <h2 class="section-title">{{ t('sync.diffTitle') }}</h2>
      <p class="sync-diff-hint">{{ t('sync.diffHint') }}</p>
      <div class="sync-diff-table">
        <div class="sync-diff-row sync-diff-head">
          <span class="sync-diff-field"></span>
          <span class="sync-diff-val">{{ t('sync.fileValue') }}</span>
          <span class="sync-diff-val">{{ t('sync.actualValue') }}</span>
        </div>
        <div v-for="f in syncDiff.fields" :key="f.field" class="sync-diff-row">
          <span class="sync-diff-field">{{ fieldLabel(f.field) }}</span>
          <span class="sync-diff-val">{{ displayVal(f.fileValue) }}</span>
          <span class="sync-diff-val">{{ displayVal(f.actualValue) }}</span>
        </div>
      </div>
    </div>

    <div class="form-section">
      <h2 class="section-title">{{ t('form.serviceConfig') }}</h2>
      <div class="form-grid">
        <div class="form-group">
          <label>{{ t('form.serviceName') }}</label>
          <input v-model="config.serviceName" :placeholder="t('form.serviceNamePlaceholder')" />
        </div>
        <div class="form-group">
          <label>{{ t('form.displayName') }}</label>
          <input v-model="config.displayName" :placeholder="t('form.displayNamePlaceholder')" @focus="autoFillFromServiceName('displayName')" />
        </div>
        <div class="form-group full-width">
          <label>{{ t('form.description') }}</label>
          <textarea v-model="config.description" rows="2" :placeholder="t('form.descriptionPlaceholder')" @focus="autoFillFromServiceName('description')"></textarea>
        </div>
      </div>
    </div>

    <div class="form-section">
      <h2 class="section-title">{{ t('form.application') }}</h2>
      <div class="form-grid">
        <div class="form-group full-width">
          <label>{{ t('form.appPath') }}</label>
          <div class="input-with-btn">
            <textarea v-model="config.appPath" rows="2" :placeholder="t('form.appPathPlaceholder')"></textarea>
            <button class="btn-sm btn-secondary btn-browse" @click="$emit('browse-app')">{{ t('form.browse') }}</button>
          </div>
          <span class="field-hint">{{ t('form.appPathHint') }}</span>
        </div>
        <div class="form-group full-width">
          <label>{{ t('form.arguments') }}</label>
          <textarea v-model="config.arguments" rows="4" :placeholder="t('form.argumentsPlaceholder')"></textarea>
          <span class="field-hint">{{ t('form.argumentsHint') }}</span>
        </div>
        <div class="form-group full-width">
          <label>{{ t('form.workDir') }}</label>
          <div class="input-with-btn">
            <input v-model="config.workDir" :placeholder="t('form.workDirPlaceholder')" />
            <button class="btn-sm btn-secondary btn-browse" @click="$emit('browse-dir')">{{ t('form.browse') }}</button>
          </div>
        </div>
      </div>
    </div>

    <div class="form-section">
      <h2 class="section-title">{{ t('form.startup') }}</h2>
      <div class="form-grid">
        <div class="form-group">
          <label>{{ t('form.account') }}</label>
          <input v-model="config.account" :placeholder="t('form.accountPlaceholder')" />
        </div>
        <div class="form-group">
          <label>{{ t('form.password') }}</label>
          <input v-model="config.password" type="password" :placeholder="t('form.passwordPlaceholder')" />
        </div>
        <div class="form-group">
          <label>{{ t('form.startType') }}</label>
          <select v-model="config.startType">
            <option value="auto">{{ t('form.startAuto') }}</option>
            <option value="demand">{{ t('form.startManual') }}</option>
            <option value="disabled">{{ t('form.startDisabled') }}</option>
          </select>
        </div>
      </div>
    </div>

    <div class="form-section">
      <h2 class="section-title">{{ t('form.logging') }}</h2>
      <div class="form-grid">
        <div class="form-group">
          <label>{{ t('form.logStdout') }}</label>
          <input v-model="config.logStdout" :placeholder="t('form.logStdoutPlaceholder')" />
        </div>
        <div class="form-group">
          <label>{{ t('form.logStderr') }}</label>
          <input v-model="config.logStderr" :placeholder="t('form.logStderrPlaceholder')" />
        </div>
        <div class="form-group checkbox-group">
          <label>
            <input type="checkbox" v-model="config.rotateLog" />
            {{ t('form.rotateLog') }}
          </label>
        </div>
      </div>
    </div>

    <div class="form-section">
      <h2 class="section-title">{{ t('form.recovery') }}</h2>
      <div class="form-grid">
        <div class="form-group">
          <label>{{ t('form.restartDelay') }}</label>
          <input v-model.number="config.restartDelay" type="number" min="0" />
        </div>
        <div class="form-group">
          <label>{{ t('form.restartTimeout') }}</label>
          <input v-model.number="config.restartTimeout" type="number" min="0" />
        </div>
      </div>
    </div>

    <div class="form-section">
      <h2 class="section-title">{{ t('form.environment') }}</h2>
      <div class="kv-editor">
        <div v-for="(val, key, idx) in config.environment" :key="key" class="kv-row">
          <input v-model="envKeys[idx]" class="kv-key" :placeholder="t('form.envKey')" @change="updateEnvKey(idx, envKeys[idx])" />
          <span class="kv-sep">=</span>
          <input v-model="config.environment[key]" class="kv-val" :placeholder="t('form.envValue')" />
          <button class="btn-sm btn-danger" @click="removeEnvKey(key)">&times;</button>
        </div>
        <div class="kv-row kv-add">
          <input v-model="newEnvKey" class="kv-key" :placeholder="t('form.envKey')" @keydown.enter="addEnvKey" />
          <span class="kv-sep">=</span>
          <input v-model="newEnvValue" class="kv-val" :placeholder="t('form.envValue')" @keydown.enter="addEnvKey" />
          <button class="btn-sm btn-primary" @click="addEnvKey">+</button>
        </div>
      </div>
    </div>

    <div class="form-section">
      <h2 class="section-title">{{ t('form.dependencies') }}</h2>
      <div class="dep-editor">
        <div v-for="(dep, idx) in config.dependencies" :key="idx" class="dep-row">
          <input v-model="config.dependencies[idx]" class="dep-input" :placeholder="t('form.depName')" />
          <button class="btn-sm btn-danger" @click="removeDependency(idx)">&times;</button>
        </div>
        <div class="dep-row dep-add">
          <input v-model="newDependency" class="dep-input" :placeholder="t('form.depName')" @keydown.enter="addDependency" />
          <button class="btn-sm btn-primary" @click="addDependency">+</button>
        </div>
      </div>
    </div>
  </main>
</template>

<script>
import { ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'

export default {
  name: 'ConfigForm',
  props: {
    config: { type: Object, required: true },
    syncDiff: { type: Object, default: null },
  },
  emits: ['browse-app', 'browse-dir'],
  setup(props) {
    const { t } = useI18n()

    const envKeys = ref([])
    const newEnvKey = ref('')
    const newEnvValue = ref('')
    const newDependency = ref('')

    const FIELD_LABEL_KEYS = {
      displayName: 'form.displayName',
      description: 'form.description',
      appPath: 'form.appPath',
      arguments: 'form.arguments',
      workDir: 'form.workDir',
      startType: 'form.startType',
      account: 'form.account',
      environment: 'form.environment',
      dependencies: 'form.dependencies',
      logStdout: 'form.logStdout',
      logStderr: 'form.logStderr',
      rotateLog: 'form.rotateLog',
      restartDelay: 'form.restartDelay',
      restartTimeout: 'form.restartTimeout',
      error: 'sync.readError',
    }

    function fieldLabel(field) {
      return t(FIELD_LABEL_KEYS[field] || field)
    }

    function displayVal(v) {
      if (v === '' || v == null) return t('sync.emptyValue')
      return v
    }

    function syncEnvKeys() {
      envKeys.value = Object.keys(props.config.environment || {})
    }

    watch(() => props.config.environment, () => syncEnvKeys(), { deep: true })
    syncEnvKeys()

    function addEnvKey() {
      if (!newEnvKey.value) return
      if (!props.config.environment) props.config.environment = {}
      props.config.environment[newEnvKey.value] = newEnvValue.value
      newEnvKey.value = ''
      newEnvValue.value = ''
      syncEnvKeys()
    }

    function removeEnvKey(key) {
      delete props.config.environment[key]
      syncEnvKeys()
    }

    function updateEnvKey(idx, newKey) {
      const oldKey = envKeys.value[idx]
      if (oldKey === newKey) return
      const val = props.config.environment[oldKey]
      delete props.config.environment[oldKey]
      props.config.environment[newKey] = val
      syncEnvKeys()
    }

    function addDependency() {
      if (!newDependency.value) return
      if (!props.config.dependencies) props.config.dependencies = []
      props.config.dependencies.push(newDependency.value)
      newDependency.value = ''
    }

    function removeDependency(idx) {
      props.config.dependencies.splice(idx, 1)
    }

    function autoFillFromServiceName(field) {
      if (props.config.serviceName && !props.config[field]) {
        props.config[field] = props.config.serviceName
      }
    }

    return {
      t, envKeys, newEnvKey, newEnvValue, newDependency,
      addEnvKey, removeEnvKey, updateEnvKey,
      addDependency, removeDependency,
      autoFillFromServiceName,
      fieldLabel, displayVal,
    }
  },
}
</script>

<style scoped>
.main-content {
  flex: 1;
  overflow-y: auto;
  padding: 20px;
}

.form-section {
  margin-bottom: 20px;
  background: var(--bg-secondary);
  border-radius: var(--radius);
  padding: 16px 20px;
  border: 1px solid var(--border);
}

.section-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--accent);
  margin-bottom: 12px;
  padding-bottom: 8px;
  border-bottom: 1px solid var(--border);
}

.form-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 12px 16px;
}

.form-group {
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.form-group.full-width { grid-column: 1 / -1; }

.form-group label {
  font-size: 12px;
  color: var(--text-secondary);
  font-weight: 500;
}

.field-hint {
  font-size: 11px;
  color: var(--text-muted);
  margin-top: 2px;
}

.form-group input,
.form-group select,
.form-group textarea {
  width: 100%;
}

.input-with-btn {
  display: flex;
  gap: 8px;
  align-items: stretch;
}
.input-with-btn textarea,
.input-with-btn input {
  flex: 1;
  min-width: 0;
}
.btn-browse {
  align-self: flex-end;
  flex-shrink: 0;
  white-space: nowrap;
}

.checkbox-group label {
  display: flex;
  align-items: center;
  gap: 8px;
  cursor: pointer;
  padding-top: 6px;
}
.checkbox-group input[type="checkbox"] {
  width: 16px;
  height: 16px;
  accent-color: var(--accent);
}

.kv-editor {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.kv-row {
  display: flex;
  align-items: center;
  gap: 6px;
}
.kv-key {
  flex: 2;
  min-width: 0;
}
.kv-sep {
  color: var(--text-muted);
  font-weight: 600;
  flex-shrink: 0;
}
.kv-val {
  flex: 3;
  min-width: 0;
}

.dep-editor {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.dep-row {
  display: flex;
  align-items: center;
  gap: 6px;
}
.dep-input {
  flex: 1;
  min-width: 0;
}

/* Sync diff panel */
.sync-diff {
  border-color: var(--warning);
}

.sync-diff-hint {
  font-size: 12px;
  color: var(--text-muted);
  margin-bottom: 10px;
}

.sync-diff-table {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.sync-diff-row {
  display: grid;
  grid-template-columns: 140px 1fr 1fr;
  gap: 12px;
  align-items: start;
  font-size: 12px;
  padding: 6px 8px;
  border-radius: 4px;
}
.sync-diff-row:not(.sync-diff-head) {
  background: var(--bg-primary);
}

.sync-diff-head {
  color: var(--text-muted);
  font-weight: 600;
  padding-bottom: 2px;
}

.sync-diff-field {
  color: var(--text-secondary);
  font-weight: 500;
  word-break: break-all;
}

.sync-diff-val {
  color: var(--text-primary);
  font-family: Consolas, monospace;
  word-break: break-all;
}
</style>
