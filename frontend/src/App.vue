<template>
  <div class="app-container">
    <!-- Header -->
    <header class="app-header">
      <div class="header-left">
        <h1 class="app-title">NSSM Plus</h1>
        <span class="app-subtitle">Windows Non-Stucking Service Manager Plus - by Moshow</span>
      </div>
      <div class="header-actions">
        <template v-if="configFilePath">
          <span class="config-file-label" :title="configFilePath">
            &#x1F4C4; {{ configFilePath.split(/[\\/]/).pop() }}
          </span>
          <button class="btn-sm btn-secondary" @click="openInExplorer" title="Open in File Explorer">
            &#x1F4C2;
          </button>
        </template>
      </div>
    </header>

    <div class="app-body">
      <!-- Sidebar: Service List -->
      <aside class="sidebar">
        <div class="sidebar-header">
          <span>Services ({{ displayServices.length }})</span>
          <button class="btn-sm btn-primary" @click="refreshServices">Refresh</button>
        </div>
        <div class="service-list" v-if="displayServices.length > 0">
          <div
            v-for="svc in displayServices"
            :key="svc.name + '-' + svc.source"
            class="service-item"
            :class="{ active: selectedService === svc.name }"
            @click="selectService(svc)"
          >
            <div class="service-item-name">
              {{ svc.displayName || svc.name }}
              <span v-if="svc.source === 'file'" class="source-badge">File</span>
            </div>
            <div class="service-item-meta">
              <span class="status-badge" :class="statusClass(svc.status)">{{ svc.status }}</span>
              <span class="start-type">{{ svc.startType }}</span>
            </div>
          </div>
        </div>
        <div v-else class="empty-state">
          <p>No services managed by NSSM Plus</p>
          <p class="hint">Install a new service or load a config file</p>
        </div>
      </aside>

      <!-- Main Content: Configuration Form -->
      <main class="main-content">
        <div class="form-section">
          <h2 class="section-title">Service Configuration</h2>

          <div class="form-grid">
            <!-- Basic Settings -->
            <div class="form-group">
              <label>Service Name *</label>
              <input v-model="config.serviceName" placeholder="MyService" />
            </div>
            <div class="form-group">
              <label>Display Name</label>
              <input v-model="config.displayName" placeholder="My Service" />
            </div>
            <div class="form-group full-width">
              <label>Description</label>
              <textarea v-model="config.description" rows="2" placeholder="Service description"></textarea>
            </div>
          </div>
        </div>

        <div class="form-section">
          <h2 class="section-title">Application</h2>
          <div class="form-grid">
            <div class="form-group two-thirds">
              <label>Application Path *</label>
              <input v-model="config.appPath" placeholder="C:\path\to\app.exe" />
            </div>
            <div class="form-group">
              <label>Working Directory</label>
              <input v-model="config.workDir" placeholder="(default: app directory)" />
            </div>
            <div class="form-group full-width">
              <label>Arguments</label>
              <input v-model="config.arguments" placeholder="--port 8080 --config config.yaml" />
            </div>
          </div>
        </div>

        <div class="form-section">
          <h2 class="section-title">Startup</h2>
          <div class="form-grid">
            <div class="form-group">
              <label>Account</label>
              <input v-model="config.account" placeholder="LocalSystem (default)" />
            </div>
            <div class="form-group">
              <label>Password</label>
              <input v-model="config.password" type="password" placeholder="Leave empty for LocalSystem" />
            </div>
            <div class="form-group">
              <label>Start Type</label>
              <select v-model="config.startType">
                <option value="auto">Automatic</option>
                <option value="demand">Manual</option>
                <option value="disabled">Disabled</option>
              </select>
            </div>
          </div>
        </div>

        <div class="form-section">
          <h2 class="section-title">Logging</h2>
          <div class="form-grid">
            <div class="form-group">
              <label>Stdout Log Path</label>
              <input v-model="config.logStdout" placeholder="C:\logs\stdout.log" />
            </div>
            <div class="form-group">
              <label>Stderr Log Path</label>
              <input v-model="config.logStderr" placeholder="C:\logs\stderr.log" />
            </div>
            <div class="form-group checkbox-group">
              <label>
                <input type="checkbox" v-model="config.rotateLog" />
                Rotate log files
              </label>
            </div>
          </div>
        </div>

        <div class="form-section">
          <h2 class="section-title">Recovery</h2>
          <div class="form-grid">
            <div class="form-group">
              <label>Restart Delay (sec, 0=off)</label>
              <input v-model.number="config.restartDelay" type="number" min="0" />
            </div>
            <div class="form-group">
              <label>Restart Timeout (sec)</label>
              <input v-model.number="config.restartTimeout" type="number" min="0" />
            </div>
          </div>
        </div>
      </main>
    </div>

    <!-- Footer: Action Bar -->
    <footer class="action-bar">
      <div class="action-left">
        <button class="btn-secondary" @click="newConfig">
          <span class="icon">+</span> Service
        </button>
        <button class="btn-secondary" @click="loadConfig">
          <span class="icon">&#x1F4C2;</span> Open Config
        </button>
        <button class="btn-secondary" @click="saveConfig" :disabled="services.length === 0">
          <span class="icon">&#x1F4BE;</span> Save Config
        </button>
      </div>
      <div class="action-right">
        <button class="btn-primary" @click="installService" :disabled="!config.serviceName || !config.appPath">
          {{ isEditing ? 'Apply Changes' : 'Install Service' }}
        </button>
        <button class="btn-success" @click="startService" :disabled="!isEditing || selectedServiceSource !== 'installed'">
          Start
        </button>
        <button class="btn-warning" @click="stopService" :disabled="!isEditing || selectedServiceSource !== 'installed'">
          Stop
        </button>
        <button class="btn-secondary" @click="restartService" :disabled="!isEditing || selectedServiceSource !== 'installed'">
          Restart
        </button>
        <button class="btn-danger" @click="removeService" :disabled="!isEditing || selectedServiceSource !== 'installed'">
          Remove
        </button>
      </div>
    </footer>

    <!-- Toast Notification -->
    <div v-if="toast.show" class="toast" :class="'toast-' + toast.type">
      {{ toast.message }}
    </div>

    <!-- Modal Overlay -->
    <div v-if="modal.show" class="modal-overlay" @click.self="modal.show = false">
      <div class="modal">
        <h3>{{ modal.title }}</h3>
        <p>{{ modal.message }}</p>
        <div class="modal-actions">
          <button class="btn-secondary" @click="modal.show = false">Cancel</button>
          <button :class="modal.confirmClass || 'btn-danger'" @click="modal.onConfirm">Confirm</button>
        </div>
      </div>
    </div>

  </div>
</template>

<script>
import { ref, reactive, computed, onMounted } from 'vue'

export default {
  name: 'App',
  setup() {
    // State
    const services = ref([])           // installed services from Windows SCM
    const loadedServices = ref([])     // services loaded from config file
    const configFilePath = ref('')     // currently loaded config file path
    const selectedService = ref('')
    const selectedServiceSource = ref('') // 'installed' or 'file'
    const isEditing = ref(false)

    // Merged service list for sidebar display
    const displayServices = computed(() => {
      const result = []
      // Installed services first
      for (const svc of services.value) {
        result.push({
          name: svc.name,
          displayName: svc.displayName,
          status: svc.status,
          startType: svc.startType,
          source: 'installed',
        })
      }
      // Loaded but not-yet-installed services
      for (const ls of loadedServices.value) {
        if (!services.value.some(s => s.name === ls.serviceName)) {
          result.push({
            name: ls.serviceName,
            displayName: ls.displayName || ls.serviceName,
            status: 'Not Installed',
            startType: ls.startType || '-',
            source: 'file',
          })
        }
      }
      return result
    })

    const defaultConfig = () => ({
      serviceName: '',
      displayName: '',
      description: '',
      appPath: '',
      arguments: '',
      workDir: '',
      startType: 'auto',
      account: '',
      password: '',
      environment: {},
      logStdout: '',
      logStderr: '',
      rotateLog: false,
      restartDelay: 0,
      restartTimeout: 30,
      dependencies: [],
    })

    const config = reactive(defaultConfig())

    const toast = reactive({ show: false, message: '', type: 'info' })
    const modal = reactive({
      show: false,
      title: '',
      message: '',
      confirmClass: '',
      onConfirm: () => {},
    })

    // Wails binding helper
    function call(method, ...args) {
      if (window.go) {
        return window.go.main.App[method](...args)
      }
      return Promise.reject(new Error('Wails runtime not available'))
    }

    // Toast helpers
    function showToast(message, type = 'info') {
      toast.message = message
      toast.type = type
      toast.show = true
      setTimeout(() => { toast.show = false }, 3000)
    }

    function showModal(title, message, confirmClass, onConfirm) {
      modal.title = title
      modal.message = message
      modal.confirmClass = confirmClass
      modal.onConfirm = () => {
        modal.show = false
        onConfirm()
      }
      modal.show = true
    }

    // Service operations
    async function refreshServices() {
      try {
        const result = await call('GetInstalledServices')
        services.value = result || []
      } catch (e) {
        showToast('Failed to load services: ' + e.message, 'error')
      }
    }

    function selectService(svc) {
      selectedService.value = svc.name
      selectedServiceSource.value = svc.source
      isEditing.value = true

      if (svc.source === 'file') {
        // Load config from local loadedServices cache
        const cached = loadedServices.value.find(s => s.serviceName === svc.name)
        if (cached) {
          Object.assign(config, cached)
        }
      } else {
        // Load config from Windows SCM
        call('GetServiceConfig', svc.name).then(cfg => {
          if (cfg) {
            Object.assign(config, cfg)
          }
        }).catch(e => {
          showToast('Failed to load service config: ' + e.message, 'error')
        })
      }
    }

    function statusClass(status) {
      switch (status) {
        case 'Running': return 'status-running'
        case 'Stopped': return 'status-stopped'
        case 'Not Installed': return 'status-file'
        default: return 'status-other'
      }
    }

    async function installService() {
      if (!config.serviceName || !config.appPath) {
        showToast('Service name and application path are required', 'warning')
        return
      }
      try {
        if (isEditing.value && selectedServiceSource.value === 'installed') {
          const oldName = selectedService.value || config.serviceName
          await call('ModifyService', oldName, JSON.parse(JSON.stringify(config)))
          showToast('Service updated successfully', 'success')
        } else {
          await call('InstallService', JSON.parse(JSON.stringify(config)))
          showToast('Service installed successfully', 'success')
        }
        await refreshServices()
        selectedService.value = config.serviceName
        selectedServiceSource.value = 'installed'
        isEditing.value = true
      } catch (e) {
        showToast('Failed: ' + e.message, 'error')
      }
    }

    async function startService() {
      try {
        const name = config.serviceName || selectedService.value
        await call('StartService', name)
        showToast('Service started', 'success')
        await refreshServices()
      } catch (e) {
        showToast('Failed to start: ' + e.message, 'error')
      }
    }

    async function stopService() {
      try {
        const name = config.serviceName || selectedService.value
        await call('StopService', name)
        showToast('Service stopped', 'success')
        await refreshServices()
      } catch (e) {
        showToast('Failed to stop: ' + e.message, 'error')
      }
    }

    async function restartService() {
      try {
        const name = config.serviceName || selectedService.value
        await call('RestartService', name)
        showToast('Service restarted', 'success')
        await refreshServices()
      } catch (e) {
        showToast('Failed to restart: ' + e.message, 'error')
      }
    }

    async function removeService() {
      const name = config.serviceName || selectedService.value
      showModal(
        'Remove Service',
        `Are you sure you want to permanently remove service "${name}"?`,
        'btn-danger',
        async () => {
          try {
            await call('RemoveService', name)
            showToast('Service removed', 'success')
            newConfig()
            await refreshServices()
          } catch (e) {
            showToast('Failed to remove: ' + e.message, 'error')
          }
        }
      )
    }

    function newConfig() {
      Object.assign(config, defaultConfig())
      selectedService.value = ''
      selectedServiceSource.value = ''
      isEditing.value = false
    }

    // --- Config file operations (multi-service) ---
    async function saveConfig() {
      try {
        const defaultName = configFilePath.value ? configFilePath.value.split(/[\\/]/).pop() : 'services.json'
        const filePath = await call('ShowSaveDialog', 'Save Config', defaultName)
        if (!filePath) return
        await call('SaveConfigToFile', filePath)
        configFilePath.value = filePath
        showToast(`Saved ${services.value.length} service(s) to ${filePath.split(/[\\/]/).pop()}`, 'success')
      } catch (e) {
        showToast('Failed to save: ' + e.message, 'error')
      }
    }

    async function loadConfig() {
      try {
        const filePath = await call('ShowOpenDialog', 'Open Config File')
        if (!filePath) return
        const configs = await call('LoadConfigFromFile', filePath)
        if (!configs || configs.length === 0) {
          showToast('No service configurations found in file', 'warning')
          return
        }
        loadedServices.value = configs
        configFilePath.value = filePath
        // Populate form with first config
        Object.assign(config, configs[0])
        selectedService.value = configs[0].serviceName
        selectedServiceSource.value = 'file'
        isEditing.value = true
        await refreshServices()
        showToast(`Loaded ${configs.length} service(s) from ${filePath.split(/[\\/]/).pop()}`, 'success')
      } catch (e) {
        showToast('Failed to load: ' + e.message, 'error')
      }
    }

    async function openInExplorer() {
      if (!configFilePath.value) return
      try {
        await call('OpenInExplorer', configFilePath.value)
      } catch (e) {
        showToast('Failed to open: ' + e.message, 'error')
      }
    }

    onMounted(() => {
      refreshServices()
    })

    return {
      services, loadedServices, displayServices,
      configFilePath, selectedService, selectedServiceSource,
      config, isEditing, toast, modal,
      statusClass, refreshServices, selectService,
      installService, startService, stopService, restartService, removeService,
      newConfig, saveConfig, loadConfig, openInExplorer,
    }
  }
}
</script>

<style scoped>
.app-container {
  display: flex;
  flex-direction: column;
  height: 100vh;
  overflow: hidden;
}

/* Header */
.app-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 12px 20px;
  background: var(--bg-secondary);
  border-bottom: 1px solid var(--border);
  flex-shrink: 0;
}

.header-left {
  display: flex;
  align-items: baseline;
  gap: 12px;
}

.app-title {
  font-size: 20px;
  font-weight: 700;
  color: var(--accent);
}

.app-subtitle {
  font-size: 13px;
  color: var(--text-muted);
}

.header-actions {
  display: flex;
  gap: 8px;
  align-items: center;
}

.config-file-label {
  font-size: 12px;
  color: var(--text-secondary);
  background: var(--bg-hover);
  padding: 3px 10px;
  border-radius: var(--radius);
  max-width: 240px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* Body */
.app-body {
  display: flex;
  flex: 1;
  overflow: hidden;
}

/* Sidebar */
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

.service-item:hover {
  background: var(--bg-hover);
}

.service-item.active {
  background: var(--bg-active);
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
}

.status-badge {
  padding: 1px 6px;
  border-radius: 3px;
  font-size: 11px;
  font-weight: 600;
}

.status-running {
  background: rgba(76, 175, 80, 0.2);
  color: var(--success);
}

.status-stopped {
  background: rgba(244, 67, 54, 0.2);
  color: var(--danger);
}

.status-file {
  background: rgba(158, 158, 158, 0.2);
  color: var(--text-muted);
}

.status-other {
  background: rgba(255, 152, 0, 0.2);
  color: var(--warning);
}

.start-type {
  color: var(--text-muted);
}

.empty-state {
  padding: 40px 20px;
  text-align: center;
  color: var(--text-muted);
}

.empty-state .hint {
  font-size: 12px;
  margin-top: 6px;
}

/* Main Content */
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

.form-group.full-width {
  grid-column: 1 / -1;
}

.form-group.two-thirds {
  grid-column: 1 / 3;
}

.form-group label {
  font-size: 12px;
  color: var(--text-secondary);
  font-weight: 500;
}

.form-group input,
.form-group select,
.form-group textarea {
  width: 100%;
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

/* Action Bar */
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

/* Toast */
.toast {
  position: fixed;
  top: 20px;
  right: 20px;
  padding: 12px 20px;
  border-radius: var(--radius);
  font-size: 13px;
  font-weight: 500;
  z-index: 1000;
  animation: slideIn 0.3s ease;
  box-shadow: var(--shadow);
}

.toast-success {
  background: var(--success);
  color: white;
}

.toast-error {
  background: var(--danger);
  color: white;
}

.toast-warning {
  background: var(--warning);
  color: white;
}

.toast-info {
  background: var(--accent);
  color: white;
}

@keyframes slideIn {
  from { transform: translateX(100%); opacity: 0; }
  to { transform: translateX(0); opacity: 1; }
}

/* Modal */
.modal-overlay {
  position: fixed;
  top: 0; left: 0; right: 0; bottom: 0;
  background: rgba(0, 0, 0, 0.6);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 999;
}

.modal {
  background: var(--bg-secondary);
  border: 1px solid var(--border);
  border-radius: 8px;
  padding: 24px;
  min-width: 400px;
  max-width: 500px;
  box-shadow: 0 8px 32px rgba(0, 0, 0, 0.4);
}

.modal h3 {
  font-size: 16px;
  margin-bottom: 12px;
}

.modal p {
  color: var(--text-secondary);
  margin-bottom: 20px;
  line-height: 1.5;
}

.modal-actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}

.icon {
  font-size: 15px;
}
</style>
