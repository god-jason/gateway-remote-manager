<template>
  <div class="upgrade-page">
    <!-- 发起升级 -->
    <el-card class="trigger-card">
      <div class="trigger">
        <span class="label">发起升级：</span>
        <el-select v-model="form.device_id" placeholder="选择设备" style="width: 220px" filterable>
          <el-option v-for="d in devices" :key="d.device_id" :label="`${d.name} (${d.device_id})`" :value="d.device_id" />
        </el-select>
        <el-select v-model="form.release_id" placeholder="选择版本" style="width: 260px" filterable>
          <el-option
            v-for="r in publishedReleases"
            :key="r.id"
            :label="`${r.version} (${r.os}/${r.arch})`"
            :value="r.id"
          />
        </el-select>
        <el-button type="primary" :disabled="!form.device_id || !form.release_id" @click="doUpgrade">执行升级</el-button>
      </div>
    </el-card>

    <el-card>
      <el-tabs v-model="tab">
        <el-tab-pane label="版本仓库" name="release">
          <div class="toolbar">
            <el-button @click="loadReleases">刷新</el-button>
            <el-button type="primary" @click="openUpload">上传版本</el-button>
          </div>
          <el-table :data="releases" v-loading="releaseLoading" border size="small">
            <el-table-column prop="version" label="版本" width="120" />
            <el-table-column label="平台" width="140">
              <template #default="{ row }">{{ row.os }} / {{ row.arch }}</template>
            </el-table-column>
            <el-table-column prop="file_name" label="文件" min-width="180" show-overflow-tooltip />
            <el-table-column label="大小" width="100">
              <template #default="{ row }">{{ fmtBytes(row.size) }}</template>
            </el-table-column>
            <el-table-column label="SHA256" width="130">
              <template #default="{ row }">{{ short(row.sha256) }}</template>
            </el-table-column>
            <el-table-column label="状态" width="90">
              <template #default="{ row }">
                <el-tag :type="row.published ? 'success' : 'info'" size="small">{{ row.published ? '已发布' : '已停用' }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="notes" label="说明" min-width="140" show-overflow-tooltip />
            <el-table-column label="创建时间" width="170">
              <template #default="{ row }">{{ fmtTime(row.created) }}</template>
            </el-table-column>
            <el-table-column label="操作" width="130" fixed="right">
              <template #default="{ row }">
                <el-button v-if="!row.published" link type="success" @click="publish(row)">发布</el-button>
                <el-button v-else link type="warning" @click="disable(row)">停用</el-button>
                <el-button link type="danger" @click="removeRelease(row)">删除</el-button>
              </template>
            </el-table-column>
          </el-table>
          <el-empty v-if="!releaseLoading && releases.length === 0" description="暂无版本" />
        </el-tab-pane>

        <el-tab-pane label="升级任务" name="task">
          <div class="toolbar">
            <el-button @click="loadTasks">刷新</el-button>
          </div>
          <el-table :data="tasks" v-loading="taskLoading" border size="small">
            <el-table-column prop="device_id" label="设备" width="160" />
            <el-table-column prop="version" label="目标版本" width="120" />
            <el-table-column label="状态" width="110">
              <template #default="{ row }">
                <el-tag :type="taskStatusTag(row.status)" size="small">{{ taskStatusText(row.status) }}</el-tag>
              </template>
            </el-table-column>
            <el-table-column prop="message" label="信息" min-width="180" show-overflow-tooltip />
            <el-table-column label="创建时间" width="170">
              <template #default="{ row }">{{ fmtTime(row.created) }}</template>
            </el-table-column>
          </el-table>
          <el-empty v-if="!taskLoading && tasks.length === 0" description="暂无升级任务" />
        </el-tab-pane>
      </el-tabs>
    </el-card>

    <!-- 上传版本 -->
    <el-dialog v-model="uploadDialog" title="上传版本" width="480px">
      <el-form label-width="80px" @submit.prevent>
        <el-form-item label="版本">
          <el-input v-model="uploadForm.version" placeholder="如 0.2.0" />
        </el-form-item>
        <el-form-item label="OS">
          <el-input v-model="uploadForm.os" placeholder="linux" />
        </el-form-item>
        <el-form-item label="架构">
          <el-input v-model="uploadForm.arch" placeholder="amd64 / arm64" />
        </el-form-item>
        <el-form-item label="文件">
          <el-upload :auto-upload="false" :show-file-list="false" :on-change="onFileChange">
            <el-button>选择文件</el-button>
          </el-upload>
          <span class="file-name">{{ uploadForm.file ? uploadForm.file.name : '未选择' }}</span>
        </el-form-item>
        <el-form-item label="说明">
          <el-input v-model="uploadForm.notes" type="textarea" :rows="2" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="uploadDialog = false">取消</el-button>
        <el-button type="primary" :loading="uploading" :disabled="!uploadForm.file" @click="doUpload">上传</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { releaseSearch, releaseUpload, releaseDelete, releasePublish, releaseDisable, upgradeTaskSearch, upgradeDevice } from '@/api/upgrade'
import { searchDevices } from '@/api/device'

const tab = ref('release')
const releases = ref([])
const releaseLoading = ref(false)
const tasks = ref([])
const taskLoading = ref(false)
const devices = ref([])
const form = ref({ device_id: '', release_id: '' })

const uploadDialog = ref(false)
const uploading = ref(false)
const uploadForm = ref({ version: '', os: '', arch: '', notes: '', file: null })

let timer = null

const publishedReleases = computed(() => releases.value.filter((r) => r.published))

async function loadReleases() {
  releaseLoading.value = true
  try {
    const res = await releaseSearch()
    releases.value = res.data || []
  } catch (e) {
  } finally {
    releaseLoading.value = false
  }
}

async function loadTasks() {
  taskLoading.value = true
  try {
    const res = await upgradeTaskSearch()
    tasks.value = res.data || []
  } catch (e) {
  } finally {
    taskLoading.value = false
  }
}

async function loadDevices() {
  try {
    const res = await searchDevices({ page: 1, page_size: 100 })
    devices.value = res.data || []
  } catch (e) {}
}

async function doUpgrade() {
  try {
    await upgradeDevice(form.value.device_id, form.value.release_id)
    ElMessage.success('升级任务已下发')
    form.value.release_id = ''
    loadTasks()
    tab.value = 'task'
  } catch (e) {}
}

function openUpload() {
  uploadForm.value = { version: '', os: '', arch: '', notes: '', file: null }
  uploadDialog.value = true
}

function onFileChange(file) {
  uploadForm.value.file = file.raw
}

async function doUpload() {
  const uf = uploadForm.value
  if (!uf.version || !uf.os || !uf.arch) {
    ElMessage.warning('请填写版本 / OS / 架构')
    return
  }
  if (!uf.file) {
    ElMessage.warning('请选择文件')
    return
  }
  const fd = new FormData()
  fd.append('file', uf.file)
  fd.append('version', uf.version)
  fd.append('os', uf.os)
  fd.append('arch', uf.arch)
  fd.append('notes', uf.notes)
  uploading.value = true
  try {
    await releaseUpload(fd)
    ElMessage.success('上传成功')
    uploadDialog.value = false
    loadReleases()
  } catch (e) {
  } finally {
    uploading.value = false
  }
}

async function publish(row) {
  await releasePublish(row.id)
  loadReleases()
}

async function disable(row) {
  await releaseDisable(row.id)
  loadReleases()
}

async function removeRelease(row) {
  await ElMessageBox.confirm(`确认删除版本「${row.version}」？`, '警告', { type: 'warning' })
  await releaseDelete(row.id)
  ElMessage.success('已删除')
  loadReleases()
}

function taskStatusText(s) {
  const map = {
    pending: '待下发',
    dispatched: '已下发',
    downloading: '下载中',
    verifying: '校验中',
    backing_up: '备份中',
    replacing: '替换中',
    done: '替换完成',
    success: '成功',
    failed: '失败',
    error: '下发失败'
  }
  return map[s] || s || '-'
}

function taskStatusTag(s) {
  switch (s) {
    case 'success':
      return 'success'
    case 'failed':
    case 'error':
      return 'danger'
    case 'downloading':
    case 'verifying':
    case 'backing_up':
    case 'replacing':
      return 'primary'
    case 'done':
      return 'warning'
    default:
      return 'info'
  }
}

function short(s) {
  return s ? s.slice(0, 14) : '-'
}

function fmtBytes(n) {
  if (n == null) return '-'
  if (n === 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  let i = 0
  let v = n
  while (v >= 1024 && i < units.length - 1) {
    v /= 1024
    i++
  }
  return v.toFixed(1) + ' ' + units[i]
}

function fmtTime(v) {
  if (!v) return '-'
  const d = new Date(v)
  return isNaN(d.getTime()) ? String(v) : d.toLocaleString()
}

onMounted(() => {
  loadReleases()
  loadTasks()
  loadDevices()
  timer = setInterval(loadTasks, 3000)
})
onUnmounted(() => {
  if (timer) clearInterval(timer)
})
</script>

<style scoped>
.trigger-card {
  margin-bottom: 16px;
}
.trigger {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
}
.label {
  color: #606266;
  font-size: 14px;
}
.toolbar {
  margin-bottom: 12px;
}
.file-name {
  margin-left: 12px;
  color: #909399;
  font-size: 13px;
}
</style>