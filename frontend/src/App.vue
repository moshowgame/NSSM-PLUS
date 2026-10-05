<template>
  <div class="app-container">
    <!-- Header -->
    <header class="app-header">
      <div class="header-left">
        <h1 class="app-title">{{ t('app.title') }}</h1>
        <span class="app-subtitle">{{ t('app.subtitle') }}</span>
      </div>
      <div class="header-actions">
        <!-- Language Switcher -->
        <div class="lang-switcher">
          <button
            class="btn-sm"
            :class="locale === 'en' ? 'btn-primary' : 'btn-secondary'"
            @click="switchLang('en')"
          >{{ t('lang.en') }}</button>
          <button
            class="btn-sm"
            :class="locale === 'zh' ? 'btn-primary' : 'btn-secondary'"
            @click="switchLang('zh')"
          >{{ t('lang.zh') }}</button>
        </div>
        <span class="header-sep"></span>
        <template v-if="configDir">
          <span class="config-file-label" :title="configDir">
            &#x1F4C2; {{ configDirLabel }}
          </span>
          <span v-if="configErrors.length" class="config-error-badge" :title="configErrorTitles">
            &#x26A0;&#xFE0F; {{ configErrors.length }}
          </span>
          <span v-if="dirty" class="dirty-indicator" :title="t('app.unsaved')">&#x25CF;</span>
          <button class="btn-sm btn-secondary" @click="openInExplorer" title="Open in File Explorer">
            &#x1F4C2;
          </button>
        </template>
        <span class="header-sep" v-if="configDir"></span>
        <a class="header-link" href="https://zhengkai.blog.csdn.net/" target="_blank" title="CSDN Blog">
          <svg viewBox="0 0 24 24" width="16" height="16" fill="currentColor"><path d="M3.5 8.4a1.4 1.4 0 0 0-1.4 1.4v4.2a1.4 1.4 0 0 0 1.4 1.4h1.4a1.4 1.4 0 0 0 1.4-1.4V9.8a1.4 1.4 0 0 0-1.4-1.4H3.5zm7.7-4.2a1.4 1.4 0 0 0-1.4 1.4v8.4a1.4 1.4 0 0 0 1.4 1.4h1.4a1.4 1.4 0 0 0 1.4-1.4V5.6a1.4 1.4 0 0 0-1.4-1.4h-1.4zm7.7 2.8a1.4 1.4 0 0 0-1.4 1.4v5.6a1.4 1.4 0 0 0 1.4 1.4h1.4a1.4 1.4 0 0 0 1.4-1.4V8.4a1.4 1.4 0 0 0-1.4-1.4h-1.4z"/></svg>
          CSDN
        </a>
        <a class="header-link" href="https://github.com/moshowgame/NSSM-PLUS" target="_blank" title="GitHub">
          <svg viewBox="0 0 24 24" width="16" height="16" fill="currentColor"><path d="M12 0C5.37 0 0 5.37 0 12c0 5.31 3.435 9.795 8.205 11.385.6.105.825-.255.825-.57 0-.285-.015-1.23-.015-2.235-3.015.555-3.795-.735-4.035-1.41-.135-.345-.72-1.41-1.23-1.695-.42-.225-1.02-.78-.015-.795.945-.015 1.62.87 1.845 1.23 1.08 1.815 2.805 1.305 3.495.99.105-.78.42-1.305.765-1.605-2.67-.3-5.46-1.335-5.46-5.925 0-1.305.465-2.385 1.23-3.225-.12-.3-.54-1.53.12-3.18 0 0 1.005-.315 3.3 1.23.96-.27 1.98-.405 3-.405s2.04.135 3 .405c2.295-1.56 3.3-1.23 3.3-1.23.66 1.65.24 2.88.12 3.18.765.84 1.23 1.905 1.23 3.225 0 4.605-2.805 5.625-5.475 5.925.435.375.81 1.095.81 2.22 0 1.605-.015 2.895-.015 3.3 0 .315.225.69.825.57A12.02 12.02 0 0 0 24 12c0-6.63-5.37-12-12-12z"/></svg>
          GitHub
        </a>
      </div>
    </header>

    <div class="app-body">
      <ServiceList
        :display-services="displayServices"
        :selected-service="selectedService"
        @new="newConfig"
        @refresh="refreshServices"
        @select="selectService"
        @copy="copyService"
      />

      <ConfigForm
        :config="config"
        :sync-diff="currentSyncDiff"
        @browse-app="browseAppPath"
        @browse-dir="browseWorkDir"
      />
    </div>

    <ActionBar
      :config="config"
      :is-editing="isEditing"
      :source="selectedServiceSource"
      @new="newConfig"
      @import="importConfig"
      @export="exportConfig"
      @save-service="saveService"
      @install="installNewService"
      @reconfigure="reconfigureService"
      @start="startService"
      @stop="stopService"
      @restart="restartService"
      @check="checkService"
      @uninstall="removeService"
      @delete="deleteConfig"
    />

    <!-- Toast Notification -->
    <div v-if="toast.show" class="toast" :class="'toast-' + toast.type">
      {{ toast.message }}
    </div>

    <!-- Modal Overlay -->
    <div v-if="modal.show" class="modal-overlay" @click.self="closeModal">
      <div class="modal">
        <h3>{{ modal.title }}</h3>
        <p>{{ modal.message }}</p>
        <div class="modal-actions">
          <button class="btn-secondary" @click="closeModal">{{ t('modal.cancel') }}</button>
          <button :class="modal.confirmClass || 'btn-danger'" @click="modal.onConfirm">{{ t('modal.confirm') }}</button>
        </div>
      </div>
    </div>

  </div>
</template>

<script>
import { ref, reactive, computed, onMounted, onUnmounted, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import ServiceList from './components/ServiceList.vue'
import ConfigForm from './components/ConfigForm.vue'
import ActionBar from './components/ActionBar.vue'

export default {
  name: 'App',
  components: { ServiceList, ConfigForm, ActionBar },
  setup() {
    const { t, locale } = useI18n()

    // State
    const services = ref([])
    const serviceFiles = ref([])
    const syncStates = ref({})
    const orphans = ref([])
    const configDir = ref('')
    const selectedService = ref('')
    const selectedServiceSource = ref('')
    const isEditing = ref(false)
    const dirty = ref(false)
    let autoRefreshTimer = null

    // Config directory model: one single-service config file per service.
    // serviceFiles is the raw scan result; files that fail to load carry an
    // error and never affect the other files.
    const loadedServices = computed(() => serviceFiles.value
      .filter(f => !f.error)
      .map(f => f.config))
    const configErrors = computed(() => serviceFiles.value.filter(f => f.error))
    const configDirLabel = computed(() => {
      if (!configDir.value) return ''
      const parts = configDir.value.split(/[\\/]/).filter(Boolean)
      return parts.slice(-2).join('\\')
    })
    const configErrorTitles = computed(() => configErrors.value
      .map(f => `${f.filePath}: ${f.error}`)
      .join('\n'))

    // Merged service list for sidebar display.
    // Show services from the config directory (with real status if installed),
    // plus a per-service sync badge (synced/drifted) from the diff engine.
    // Orphan wrapper configs (config file without a registered service) are appended.
    const displayServices = computed(() => {
      const installedMap = {}
      for (const svc of services.value) {
        installedMap[svc.name] = svc
      }

      let items
      if (configDir.value) {
        // Config directory mode: show services from config files, enriched with real status
        items = loadedServices.value.map(ls => {
          const installed = installedMap[ls.serviceName]
          const diff = syncStates.value[ls.serviceName]
          if (installed) {
            return {
              name: installed.name,
              displayName: installed.displayName,
              status: installed.status,
              startType: installed.startType,
              appPath: installed.appPath || ls.appPath,
              source: 'installed',
              syncState: diff && diff.state !== 'notInstalled' ? diff.state : '',
            }
          }
          return {
            name: ls.serviceName,
            displayName: ls.displayName || ls.serviceName,
            status: 'Not Installed',
            startType: ls.startType || '-',
            appPath: ls.appPath,
            source: 'file',
            syncState: '',
          }
        })
      } else {
        // Config directory not loaded yet: show all installed NSSM-Plus services
        items = services.value.map(svc => ({
          name: svc.name,
          displayName: svc.displayName,
          status: svc.status,
          startType: svc.startType,
          appPath: svc.appPath,
          source: 'installed',
          syncState: '',
        }))
      }

      // Append orphan wrapper configs, skipping names already listed above
      const listed = new Set(items.map(i => i.name.toLowerCase()))
      for (const o of orphans.value) {
        if (listed.has(o.name.toLowerCase())) continue
        items.push({
          name: o.name,
          displayName: o.displayName || o.name,
          status: 'Orphan',
          startType: '-',
          appPath: o.appPath,
          source: 'orphan',
          syncState: '',
        })
      }
      return items
    })

    // Field-level diff of the currently selected service (file vs registered)
    const currentSyncDiff = computed(() => {
      if (!configDir.value) return null
      const d = syncStates.value[selectedService.value]
      if (!d || d.state !== 'drifted' || !d.fields || !d.fields.length) return null
      return d
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

    // Track unsaved changes: set dirty flag when form fields change while a service is selected
    watch(config, () => {
      if (isEditing.value) {
        dirty.value = true
      }
    }, { deep: true })

    const toast = reactive({ show: false, message: '', type: 'info' })
    const modal = reactive({
      show: false,
      title: '',
      message: '',
      confirmClass: '',
      onConfirm: () => {},
      onCancel: () => {},
    })

    function call(method, ...args) {
      if (window.go) {
        return window.go.main.App[method](...args)
      }
      return Promise.reject(new Error('Wails runtime not available'))
    }

    function errorMsg(e) {
      if (typeof e === 'string') return e
      return e?.message || String(e)
    }

    function showToast(message, type = 'info') {
      toast.message = message
      toast.type = type
      toast.show = true
      setTimeout(() => { toast.show = false }, 3000)
    }

    function showModal(title, message, confirmClass, onConfirm, onCancel) {
      modal.title = title
      modal.message = message
      modal.confirmClass = confirmClass
      modal.onConfirm = () => {
        modal.show = false
        if (onConfirm) onConfirm()
      }
      modal.onCancel = () => {
        if (onCancel) onCancel()
      }
      modal.show = true
    }

    function closeModal() {
      modal.show = false
      modal.onCancel()
    }

    // Guard an action: if there are unsaved changes, confirm before discarding.
    function guardAction(action) {
      if (!dirty.value) { action(); return }
      showModal(
        t('modal.unsavedTitle'),
        t('modal.unsavedMessage'),
        'btn-warning',
        () => {
          dirty.value = false
          action()
        }
      )
    }

    // #17 Language switcher
    function switchLang(lang) {
      locale.value = lang
      localStorage.setItem('nssm-plus-lang', lang)
    }

    // #10 File pickers
    async function browseAppPath() {
      try {
        const filePath = await call('ShowOpenAppDialog', t('form.appPath'))
        if (filePath) {
          config.appPath = filePath
        }
      } catch (e) {
        // Dialog cancelled
      }
    }

    async function browseWorkDir() {
      try {
        const dirPath = await call('ShowOpenDirectoryDialog', t('form.workDir'))
        if (dirPath) {
          config.workDir = dirPath
        }
      } catch (e) {
        // Dialog cancelled
      }
    }

    // Service operations
    async function refreshSyncStates() {
      if (!configDir.value || !loadedServices.value.length) {
        syncStates.value = {}
        return
      }
      try {
        const diffs = (await call('GetSyncStates', loadedServices.value)) || []
        const map = {}
        for (const d of diffs) {
          map[d.serviceName] = d
        }
        syncStates.value = map
      } catch (e) {
        syncStates.value = {}
      }
    }

    async function refreshServices() {
      try {
        // Always fetch installed services for status display
        try {
          const result = await call('GetInstalledServices')
          services.value = result || []
        } catch (e) {
          services.value = []
        }
        // Scan the config directory for per-service config files
        if (configDir.value) {
          try {
            serviceFiles.value = (await call('LoadConfigDir')) || []
          } catch (e) {
            serviceFiles.value = []
          }
        }
        // Refresh sync badges (file vs registered) and orphan wrapper configs
        await refreshSyncStates()
        try {
          orphans.value = (await call('GetOrphanConfigs')) || []
        } catch (e) {
          orphans.value = []
        }
      } catch (e) {
        showToast(t('toast.refreshFailed') + ': ' + errorMsg(e), 'error')
      }
    }

    async function selectService(svc) {
      guardAction(async () => {
        // Disable dirty watch during config load to avoid false positives
        isEditing.value = false
        if (svc.source === 'file') {
          const cached = loadedServices.value.find(s => s.serviceName === svc.name)
          if (cached) {
            Object.assign(config, cached)
          }
        } else if (svc.source === 'orphan') {
          try {
            const cfg = await call('GetWrapperConfig', svc.name)
            if (cfg) {
              Object.assign(config, cfg)
            }
          } catch (e) {
            showToast(t('toast.loadConfigFailed') + ': ' + errorMsg(e), 'error')
          }
        } else {
          try {
            const cfg = await call('GetServiceConfig', svc.name)
            if (cfg) {
              Object.assign(config, cfg)
            }
          } catch (e) {
            showToast(t('toast.loadConfigFailed') + ': ' + errorMsg(e), 'error')
          }
        }
        selectedService.value = svc.name
        selectedServiceSource.value = svc.source
        isEditing.value = true
        dirty.value = false
        await refreshServices()
      })
    }

    async function installNewService() {
      if (!config.serviceName || !config.appPath) {
        showToast(t('toast.nameAndPathRequired'), 'warning')
        return
      }
      try {
        await call('InstallService', JSON.parse(JSON.stringify(config)))
        showToast(t('toast.installed'), 'success')
        await refreshServices()
        selectedService.value = config.serviceName
        selectedServiceSource.value = 'installed'
        isEditing.value = true
      } catch (e) {
        showToast(t('toast.installFailed') + ': ' + errorMsg(e), 'error')
      }
    }

    // #16 Enhanced error handling - no longer silently swallowing errors
    async function reconfigureService() {
      if (!config.serviceName || !config.appPath) {
        showToast(t('toast.nameAndPathRequired'), 'warning')
        return
      }
      try {
        const name = config.serviceName || selectedService.value
        try {
          await call('StopService', name)
        } catch (e) {
          console.log('StopService note:', errorMsg(e))
        }
        await call('ModifyService', selectedService.value, JSON.parse(JSON.stringify(config)))
        try {
          await call('StartService', name)
          showToast(t('toast.reconfiguredStarted'), 'success')
        } catch (e) {
          showToast(t('toast.reconfiguredNoStart') + ': ' + errorMsg(e), 'warning')
        }
        await refreshServices()
      } catch (e) {
        showToast(t('toast.reconfigureFailed') + ': ' + errorMsg(e), 'error')
      }
    }

    async function startService() {
      try {
        const name = config.serviceName || selectedService.value
        await call('StartService', name)
        showToast(t('toast.started'), 'success')
        await refreshServices()
      } catch (e) {
        showToast(t('toast.startFailed') + ': ' + errorMsg(e), 'error')
      }
    }

    async function stopService() {
      try {
        const name = config.serviceName || selectedService.value
        await call('StopService', name)
        showToast(t('toast.stopped'), 'success')
        await refreshServices()
      } catch (e) {
        showToast(t('toast.stopFailed') + ': ' + errorMsg(e), 'error')
      }
    }

    async function restartService() {
      try {
        const name = config.serviceName || selectedService.value
        await call('RestartService', name)
        showToast(t('toast.restarted'), 'success')
        await refreshServices()
      } catch (e) {
        showToast(t('toast.restartFailed') + ': ' + errorMsg(e), 'error')
      }
    }

    async function removeService() {
      const name = config.serviceName || selectedService.value
      showModal(
        t('action.uninstall'),
        `Are you sure you want to uninstall service "${name}"?`,
        'btn-danger',
        async () => {
          try {
            await call('RemoveService', name)
            // The config file is kept on purpose: the service now shows as
            // "Not Installed" and can be reinstalled or deleted from here.
            selectedServiceSource.value = 'file'
            showToast(t('toast.uninstalled'), 'success')
            await refreshServices()
          } catch (e) {
            const msg = errorMsg(e)
            showToast(t('toast.uninstallFailed') + ': ' + msg, 'error')
            if (msg.includes('marked for deletion')) {
              setTimeout(() => {
                showToast(t('toast.markedForDeletion'), 'warning')
              }, 500)
            }
          }
        }
      )
    }

    function deleteConfig() {
      const name = config.serviceName || selectedService.value
      if (!name) {
        showToast(t('toast.nameRequired'), 'warning')
        return
      }
      showModal(
        t('action.delete'),
        `Are you sure you want to delete "${name}"? This will remove the config file entirely.`,
        'btn-danger',
        async () => {
          try {
            await call('DeleteServiceConfig', name)
            Object.assign(config, defaultConfig())
            selectedService.value = ''
            selectedServiceSource.value = ''
            isEditing.value = false
            dirty.value = false
            await refreshServices()
            showToast(t('toast.deleted'), 'success')
          } catch (e) {
            showToast(t('toast.deleteFailed') + ': ' + errorMsg(e), 'error')
          }
        }
      )
    }

    async function copyService(svc) {
      if (svc.source === 'file') {
        const cached = loadedServices.value.find(s => s.serviceName === svc.name)
        if (cached) {
          Object.assign(config, cached)
        }
      } else if (svc.source === 'orphan') {
        try {
          const cfg = await call('GetWrapperConfig', svc.name)
          if (cfg) {
            Object.assign(config, cfg)
          }
        } catch (e) {
          showToast(t('toast.loadConfigFailed') + ': ' + errorMsg(e), 'error')
          return
        }
      } else {
        try {
          const cfg = await call('GetServiceConfig', svc.name)
          if (cfg) {
            Object.assign(config, cfg)
          }
        } catch (e) {
          showToast(t('toast.loadConfigFailed') + ': ' + errorMsg(e), 'error')
          return
        }
      }
      config.serviceName = ''
      config.displayName = ''
      selectedService.value = ''
      selectedServiceSource.value = ''
      isEditing.value = true
      showToast(t('toast.copied'), 'info')
    }

    function newConfig() {
      guardAction(() => {
        Object.assign(config, defaultConfig())
        selectedService.value = ''
        selectedServiceSource.value = ''
        isEditing.value = false
        dirty.value = false
      })
    }

    async function checkService() {
      const name = config.serviceName || selectedService.value
      if (!name) return
      try {
        const status = await call('GetServiceStatus', name)
        showToast(`Service "${name}" exists, status: ${status}`, 'success')
        selectedService.value = name
        selectedServiceSource.value = 'installed'
        isEditing.value = true
      } catch (e) {
        showToast(`Service "${name}" does not exist: ${errorMsg(e)}`, 'warning')
      }
    }

    // --- Config file operations ---

    // Export every service config from the config directory into one
    // multi-service JSON bundle file (legacy format, re-importable).
    async function exportConfig() {
      try {
        if (!loadedServices.value.length) {
          showToast(t('toast.noConfigs'), 'warning')
          return
        }
        const filePath = await call('ShowSaveDialog', t('action.exportConfig'), 'services.json')
        if (!filePath) return
        await call('SaveConfigToFile', filePath, loadedServices.value)
        showToast(t('toast.saved', { count: loadedServices.value.length, file: filePath.split(/[\\/]/).pop() }), 'success')
      } catch (e) {
        showToast(t('toast.saveFailed') + ': ' + errorMsg(e), 'error')
      }
    }

    // Save the currently edited service to its own config file. When the
    // service was renamed, the old config file is removed by the backend.
    async function saveService() {
      if (!config.serviceName) {
        showToast(t('toast.nameRequired'), 'warning')
        return
      }
      try {
        const oldName = isEditing.value ? selectedService.value : ''
        await call('SaveServiceConfig', oldName, JSON.parse(JSON.stringify(config)))
        selectedService.value = config.serviceName
        selectedServiceSource.value = 'file'
        isEditing.value = true
        dirty.value = false
        await refreshServices()
        showToast(t('toast.serviceSaved', { file: config.serviceName + '.json' }), 'success')
      } catch (e) {
        showToast(t('toast.saveFailed') + ': ' + errorMsg(e), 'error')
      }
    }

    // Import a legacy multi-service JSON bundle (or single-service object):
    // each service is split into its own config file in the config directory.
    async function importConfig() {
      guardAction(async () => {
        try {
          const filePath = await call('ShowOpenDialog', t('action.importConfig'))
          if (!filePath) return
          const configs = await call('LoadConfigFromFile', filePath)
          if (!configs || configs.length === 0) {
            showToast(t('toast.noConfigs'), 'warning')
            return
          }
          let imported = 0
          for (const cfg of configs) {
            if (!cfg.serviceName) continue
            try {
              await call('SaveServiceConfig', '', cfg)
              imported++
            } catch (e) {
              console.error('Failed to import service config:', cfg.serviceName, e)
            }
          }
          if (imported === 0) {
            showToast(t('toast.importFailed'), 'error')
            return
          }
          isEditing.value = false
          await refreshServices()
          const first = loadedServices.value.find(s => configs.some(c => c.serviceName === s.serviceName))
          if (first) {
            Object.assign(config, first)
            selectedService.value = first.serviceName
            selectedServiceSource.value = 'file'
            isEditing.value = true
            dirty.value = false
          }
          showToast(t('toast.imported', { count: imported, file: filePath.split(/[\\/]/).pop() }), 'success')
        } catch (e) {
          showToast(t('toast.loadConfigFailed') + ': ' + errorMsg(e), 'error')
        }
      })
    }

    async function openInExplorer() {
      if (!configDir.value) return
      try {
        await call('OpenInExplorer', configDir.value)
      } catch (e) {
        showToast(t('toast.fileOpenFailed') + ': ' + errorMsg(e), 'error')
      }
    }

    async function debugInfo() {
      console.group('[NSSM-Plus Debug]')
      console.log('Config:', JSON.parse(JSON.stringify(config)))
      console.log('Selected:', selectedService.value, '| Source:', selectedServiceSource.value)
      console.log('Installed Services:', JSON.parse(JSON.stringify(services.value)))
      console.log('Config Directory:', configDir.value)
      console.log('Service Files:', JSON.parse(JSON.stringify(serviceFiles.value)))
      console.log('Display Services:', JSON.parse(JSON.stringify(displayServices.value)))
      console.groupEnd()
      showToast(t('toast.debugInfo'), 'info')
    }

    function autoFillFromServiceName(field) {
      if (config.serviceName && !config[field]) {
        config[field] = config.serviceName
      }
    }

    onMounted(async () => {
      try {
        configDir.value = (await call('GetConfigDir')) || ''
      } catch (e) {
        configDir.value = ''
      }
      await refreshServices()
      // Auto-select the first service from the config directory, if any
      if (loadedServices.value.length) {
        const first = loadedServices.value[0]
        Object.assign(config, first)
        selectedService.value = first.serviceName
        selectedServiceSource.value = 'file'
        isEditing.value = true
        dirty.value = false
      }
      autoRefreshTimer = setInterval(refreshServices, 10000)
    })

    onUnmounted(() => {
      if (autoRefreshTimer) {
        clearInterval(autoRefreshTimer)
        autoRefreshTimer = null
      }
    })

    return {
      services, serviceFiles, loadedServices, displayServices, currentSyncDiff,
      configDir, configDirLabel, configErrors, configErrorTitles,
      selectedService, selectedServiceSource,
      config, isEditing, dirty, toast, modal,
      locale, t, switchLang,
      refreshServices, selectService, copyService,
      installNewService, reconfigureService, startService, stopService, restartService, removeService,
      newConfig, deleteConfig, checkService, exportConfig, saveService, importConfig, openInExplorer, debugInfo,
      browseAppPath, browseWorkDir,
      closeModal, guardAction,
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

.config-error-badge {
  font-size: 12px;
  color: var(--warning);
  background: var(--bg-hover);
  padding: 3px 8px;
  border-radius: var(--radius);
  cursor: help;
}

.dirty-indicator {
  color: var(--warning);
  font-size: 14px;
  margin-left: 2px;
  animation: pulse 1.2s ease-in-out infinite;
}

@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.4; }
}

/* Language Switcher */
.lang-switcher {
  display: flex;
  gap: 2px;
}
.lang-switcher .btn-sm {
  padding: 3px 8px;
  font-size: 11px;
  border-radius: 3px;
}

/* Body */
.app-body {
  display: flex;
  flex: 1;
  overflow: hidden;
}

.header-sep {
  width: 1px;
  height: 18px;
  background: var(--border);
  margin: 0 4px;
}

.header-link {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 12px;
  color: var(--text-muted);
  text-decoration: none;
  padding: 2px 6px;
  border-radius: var(--radius);
  transition: color 0.15s, background 0.15s;
}
.header-link:hover {
  color: var(--accent);
  background: var(--bg-hover);
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
  max-width: 500px;
}
.toast-success { background: var(--success); color: white; }
.toast-error { background: var(--danger); color: white; }
.toast-warning { background: var(--warning); color: white; }
.toast-info { background: var(--accent); color: white; }

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
.modal h3 { font-size: 16px; margin-bottom: 12px; }
.modal p { color: var(--text-secondary); margin-bottom: 20px; line-height: 1.5; }

.modal-actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}

.icon {
  font-size: 15px;
}
</style>
