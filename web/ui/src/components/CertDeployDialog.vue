<script setup>
import { computed, ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { api } from '../api'
import { SERVER_TYPES, isPfxServer, filesFor, fileText, buildCommands } from '../cert/deploy'

const props = defineProps({
  visible: Boolean,
  name: String,
  domains: { type: Array, default: () => [] },
  initialTab: { type: String, default: 'cmd' }
})
const emit = defineEmits(['update:visible'])

const dialogVisible = computed({
  get: () => props.visible,
  set: (v) => emit('update:visible', v)
})

const tab = ref('cmd')
const activeFile = ref('')
const server = ref('nginx')
const os = ref('linux')
const writeMode = ref('text')
const files = ref([])
const pfxPassword = ref('')
const contents = ref(null)
const loading = ref(false)

const domain = computed(() => (props.domains[0] || 'cert').replace(/^\*\./, '').replace(/[^a-zA-Z0-9.-]/g, '_'))
const pfxServer = computed(() => isPfxServer(server.value))

watch(() => props.visible, async (v) => {
  if (v) await init()
})

async function init() {
  tab.value = props.initialTab
  server.value = 'nginx'
  os.value = 'linux'
  writeMode.value = 'text'
  pfxPassword.value = ''
  contents.value = null
  resetFiles()
  await loadPem()
}

function resetFiles() {
  files.value = filesFor(server.value, domain.value)
  activeFile.value = files.value[0]?.key || ''
}

async function loadPem() {
  loading.value = true
  try {
    const data = await api('/api/cert/content?name=' + encodeURIComponent(props.name))
    contents.value = data?.data
  } catch (e) {
    ElMessage.error(e.message)
    dialogVisible.value = false
  } finally {
    loading.value = false
  }
}

async function loadPfx() {
  if (!pfxServer.value) return
  loading.value = true
  try {
    const params = new URLSearchParams({ name: props.name, format: 'pfx', password: pfxPassword.value || '' })
    const data = await api('/api/cert/content?' + params.toString())
    contents.value = { ...contents.value, pfx: data?.data?.pfx || '' }
  } catch (e) {
    ElMessage.error(e.message)
  } finally {
    loading.value = false
  }
}

function onServerChange() {
  resetFiles()
  if (pfxServer.value) {
    writeMode.value = 'base64'
    loadPfx()
  }
}

const commands = computed(() => buildCommands({
  os: os.value,
  mode: writeMode.value,
  files: files.value,
  contents: contents.value,
  server: server.value
}))

async function copy(text) {
  try {
    await navigator.clipboard.writeText(text)
    ElMessage.success('已复制')
  } catch {
    ElMessage.warning('复制失败，请手动全选文本复制')
  }
}
</script>

<template>
  <el-dialog v-model="dialogVisible" title="部署证书" width="840px">
    <div v-loading="loading">
      <el-form label-position="top">
        <div class="form-grid">
          <el-form-item label="服务器类型" required>
            <el-select v-model="server" style="width: 100%" @change="onServerChange">
              <el-option v-for="s in SERVER_TYPES" :key="s.value" :label="s.label" :value="s.value" />
            </el-select>
          </el-form-item>

          <template v-if="tab === 'cmd'">
            <el-form-item label="目标终端">
              <el-radio-group v-model="os">
                <el-radio-button value="linux">Linux（bash/sh）</el-radio-button>
                <el-radio-button value="powershell">PowerShell</el-radio-button>
                <el-radio-button value="cmd">CMD</el-radio-button>
              </el-radio-group>
            </el-form-item>
            <el-form-item label="写入方式">
              <el-radio-group v-model="writeMode">
                <el-radio-button value="text" :disabled="pfxServer">文本直写</el-radio-button>
                <el-radio-button value="base64">Base64 解码</el-radio-button>
              </el-radio-group>
            </el-form-item>
          </template>

          <el-form-item v-if="pfxServer" label="PFX 密码（可留空）">
            <el-input
              v-model="pfxPassword"
              type="password"
              show-password
              placeholder="修改后失焦自动重新生成"
              @change="loadPfx"
            />
          </el-form-item>

          <el-form-item v-for="f in files" :key="f.key" :label="f.label + '文件名'">
            <el-input v-model="f.name" />
          </el-form-item>
        </div>
      </el-form>

      <el-tabs v-model="tab">
        <el-tab-pane label="终端命令" name="cmd">
          <el-input
            :model-value="commands"
            type="textarea"
            :rows="16"
            readonly
            class="terminal-output"
            placeholder="正在获取证书内容..."
          />
          <div class="hint" style="margin-top: 8px">
            复制后在目标服务器终端的<strong>当前目录</strong>粘贴执行。文本直写保留 PEM 原文；Base64 解码适合任意内容，PFX 二进制仅支持 Base64。私钥内容敏感，请勿外传。
          </div>
        </el-tab-pane>

        <el-tab-pane label="证书内容" name="content">
          <el-tabs v-model="activeFile">
            <el-tab-pane v-for="f in files" :key="f.key" :label="f.label" :name="f.key">
              <el-input
                :model-value="fileText(f, contents)"
                type="textarea"
                :rows="14"
                readonly
                class="terminal-output"
              />
              <div style="margin-top: 8px; text-align: right">
                <el-button size="small" @click="copy(fileText(f, contents))">复制{{ f.label }}</el-button>
              </div>
            </el-tab-pane>
          </el-tabs>
        </el-tab-pane>
      </el-tabs>
    </div>

    <template #footer>
      <el-button @click="dialogVisible = false">关闭</el-button>
      <el-button v-if="tab === 'cmd'" type="primary" :disabled="!commands" @click="copy(commands)">复制命令</el-button>
    </template>
  </el-dialog>
</template>
